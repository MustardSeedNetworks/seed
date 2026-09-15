package bonjour

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	multicastGroupV4 = "224.0.0.251"
	port             = 5353

	// defaultBrowseWindow is how long a browse listens. DNS-SD responders
	// answer within a second, but a reflector adds a hop and the answer to
	// "is anything arriving from elsewhere" is only as good as the window it
	// was asked over.
	defaultBrowseWindow = 4 * time.Second
	// maxBrowseWindow bounds what a caller may ask for: this is a handheld
	// diagnostic, not a monitor.
	maxBrowseWindow = 15 * time.Second

	readBufferSize   = 65536
	packetBufferSize = 9000
	// queryBufferSize is ample for one question: a DNS name is at most 255
	// bytes and a question adds four more.
	queryBufferSize = 512
)

// errNoInterface is returned when the named interface does not exist, so the
// caller can say which one rather than reporting a generic socket failure.
var errNoInterface = errors.New("interface not found")

// Browser runs one bounded DNS-SD browse over the attached segment.
type Browser struct {
	interfaceName string
	window        time.Duration
	lim           limits
}

// NewBrowser returns a browser for one interface. An empty name lets the
// kernel choose, matching MDNSListener.
func NewBrowser(interfaceName string) *Browser {
	return &Browser{
		interfaceName: interfaceName,
		window:        defaultBrowseWindow,
		lim:           defaultLimits(),
	}
}

// SetWindow bounds the listen window, clamped to maxBrowseWindow.
func (b *Browser) SetWindow(d time.Duration) {
	if d <= 0 {
		d = defaultBrowseWindow
	}
	b.window = min(d, maxBrowseWindow)
}

// Browse enumerates service types, then instances of each, then the SRV, TXT
// and address records of each instance, and classifies what it found against
// the interface's own prefixes.
//
// The three stages are separate queries because a responder is entitled to
// answer a type PTR with the instance PTR alone: macOS's mDNSResponder does
// exactly that, so a browser that only asked twice would list instances with
// no host and no port.
func (b *Browser) Browse(ctx context.Context) (*BrowseResult, error) {
	iface, local, err := b.resolveInterface()
	if err != nil {
		return nil, err
	}

	group := &net.UDPAddr{IP: net.ParseIP(multicastGroupV4), Port: port}
	conn, err := net.ListenMulticastUDP("udp4", iface, group)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetReadBuffer(readBufferSize)

	sender, err := net.DialUDP("udp4", nil, group)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sender.Close() }()

	started := time.Now()
	collected := newCollector(b.lim)
	stages := []func() []dnsmessage.Question{
		func() []dnsmessage.Question { return []dnsmessage.Question{ptrQuestion(dnssdMetaQuery)} },
		func() []dnsmessage.Question { return typeQuestions(collected) },
		func() []dnsmessage.Question { return instanceQuestions(collected) },
	}

	stageWindow := b.window / time.Duration(len(stages))
	for _, stage := range stages {
		questions := stage()
		if len(questions) == 0 {
			continue
		}
		b.ask(sender, questions)
		b.gather(ctx, conn, collected, time.Now().Add(stageWindow))
		if ctx.Err() != nil {
			break
		}
	}

	types, services, reflector := collected.assemble(local)
	elapsed := time.Since(started)
	return &BrowseResult{
		Interface:         b.interfaceName,
		LocalPrefixes:     prefixText(local),
		ServiceTypes:      types,
		Services:          services,
		ReflectorStatus:   reflector,
		ResponsesObserved: collected.responses,
		Truncated:         collected.truncated,
		Duration:          elapsed,
		DurationMs:        elapsed.Milliseconds(),
	}, nil
}

// resolveInterface returns the interface to listen on and the prefixes every
// classification is made against. An unnamed interface classifies against
// every non-loopback prefix on the host, which is the honest answer when the
// caller did not say which segment it meant.
func (b *Browser) resolveInterface() (*net.Interface, []netip.Prefix, error) {
	if b.interfaceName == "" {
		return nil, localPrefixesOf(hostPrefixes()), nil
	}
	iface, err := net.InterfaceByName(b.interfaceName)
	if err != nil {
		return nil, nil, errNoInterface
	}
	return iface, localPrefixesOf(interfacePrefixes(iface)), nil
}

func (b *Browser) ask(sender *net.UDPConn, questions []dnsmessage.Question) {
	for _, q := range questions {
		msg, err := packQuery(q)
		if err != nil {
			continue
		}
		if _, writeErr := sender.Write(msg); writeErr != nil {
			logging.GetLogger().
				Debug("bonjour: query send failed", "name", q.Name.String(), "error", writeErr)
			return
		}
	}
}

func (b *Browser) gather(ctx context.Context, conn *net.UDPConn, into *collector, until time.Time) {
	buf := make([]byte, packetBufferSize)
	for {
		if ctx.Err() != nil || !time.Now().Before(until) {
			return
		}
		_ = conn.SetReadDeadline(until)
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			// A deadline ends the stage; anything else is a frame seed could
			// not read, and the segment gets to send those.
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return
			}
			continue
		}
		src, _ := netip.AddrFromSlice(from.IP)
		into.observe(buf[:n], src.Unmap())
	}
}

func typeQuestions(c *collector) []dnsmessage.Question {
	questions := make([]dnsmessage.Question, 0, len(c.types))
	for t := range c.types {
		questions = append(questions, ptrQuestion(t+"."))
	}
	return questions
}

// instanceQuestions asks for the SRV and TXT of every instance seen so far.
// The address records ride along in the responder's Additionals, so they are
// not asked for separately.
func instanceQuestions(c *collector) []dnsmessage.Question {
	// Two questions per instance: SRV and TXT.
	const questionsPerInstance = 2
	questions := make([]dnsmessage.Question, 0, len(c.instances)*questionsPerInstance)
	for name := range c.instances {
		n, err := dnsmessage.NewName(name + ".")
		if err != nil {
			continue
		}
		questions = append(questions,
			dnsmessage.Question{Name: n, Type: dnsmessage.TypeSRV, Class: dnsmessage.ClassINET},
			dnsmessage.Question{Name: n, Type: dnsmessage.TypeTXT, Class: dnsmessage.ClassINET},
		)
	}
	return questions
}

func ptrQuestion(name string) dnsmessage.Question {
	n, err := dnsmessage.NewName(name)
	if err != nil {
		return dnsmessage.Question{}
	}
	return dnsmessage.Question{Name: n, Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET}
}

func packQuery(q dnsmessage.Question) ([]byte, error) {
	if q.Name.Length == 0 {
		return nil, errors.New("empty question")
	}
	builder := dnsmessage.NewBuilder(make([]byte, 0, queryBufferSize), dnsmessage.Header{})
	if err := builder.StartQuestions(); err != nil {
		return nil, err
	}
	if err := builder.Question(q); err != nil {
		return nil, err
	}
	return builder.Finish()
}

func hostPrefixes() []netip.Prefix {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []netip.Prefix
	for i := range ifaces {
		out = append(out, interfacePrefixes(&ifaces[i])...)
	}
	return out
}

func interfacePrefixes(iface *net.Interface) []netip.Prefix {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	out := make([]netip.Prefix, 0, len(addrs))
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		addr, ok := netip.AddrFromSlice(ipNet.IP)
		if !ok {
			continue
		}
		ones, _ := ipNet.Mask.Size()
		p, prefixErr := addr.Unmap().Prefix(ones)
		if prefixErr != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

func prefixText(prefixes []netip.Prefix) []string {
	if len(prefixes) == 0 {
		return nil
	}
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		out = append(out, p.String())
	}
	return out
}
