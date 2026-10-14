// Package tftp serves files to network devices over TFTP (RFC 1350) for the
// length of one operator-started session (P-D4).
//
// TFTP has no authentication, so a session is kept as small as the protocol
// allows: nothing listens until an admin starts it and nothing restarts it with
// the daemon; it binds one interface's address and answers only the subnets on
// that interface (or the prefixes the admin names); it serves one directory
// through an [os.Root]; it refuses writes unless the session allows uploads,
// and then writes only into a separate upload directory it never serves,
// never overwriting a file; it caps concurrent transfers, uploads and upload
// bytes; and it stops itself once no transfer has completed for the idle
// timeout, or at the latest once it reaches its maximum lifetime.
package tftp

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pintftp "github.com/pin/tftp/v3"
)

const (
	// Port is TFTP's IANA port. A device cannot be pointed at another one:
	// IOS `copy tftp:` has no port option.
	Port = 69

	// DefaultIdleTimeout is how long a session with no transfer stays up.
	DefaultIdleTimeout = 10 * time.Minute

	// MaxUploadBytes bounds one upload. What a device sends is a
	// configuration or a crash file (images travel the other way), so this is
	// far above either and still keeps one transfer from filling the disk.
	MaxUploadBytes = 256 << 20

	// DefaultMaxTransfers bounds transfers in flight. The library opens a UDP
	// socket and a goroutine for every request before the handler runs; this
	// cap is what keeps a flood of requests from holding those open.
	DefaultMaxTransfers = 8

	// DefaultMaxLifetime ends a session however busy it is, so a device that
	// keeps polling cannot hold the port open indefinitely.
	DefaultMaxLifetime = time.Hour

	// DefaultMaxUploads bounds upload attempts in one session.
	DefaultMaxUploads = 32

	// DefaultUploadQuota bounds the bytes all uploads in one session may write.
	DefaultUploadQuota = 1 << 30

	// transferTimeout is the per-block retransmit timeout: long enough for a
	// slow management plane, short enough that Stop does not hang on a
	// client that has gone away.
	transferTimeout = 5 * time.Second

	dirMode    = 0o750
	uploadMode = 0o640
)

// Errors a caller can branch on. The transfer errors are also what the client
// sees in the TFTP ERROR packet, so they name no server path.
var (
	ErrRunning         = errors.New("a TFTP session is already running")
	ErrNotRunning      = errors.New("no TFTP session is running")
	ErrNoInterface     = errors.New("no such interface")
	ErrNoIPv4          = errors.New("interface has no IPv4 address")
	ErrBind            = errors.New("cannot bind the TFTP port")
	ErrNotFound        = errors.New("file not found")
	ErrUploadsDisabled = errors.New("uploads are not enabled")
	ErrExists          = errors.New("file already exists")
	ErrTooLarge        = errors.New("file too large")
	ErrBusy            = errors.New("server busy, try again later")
	ErrAccessDenied    = errors.New("access denied")
	ErrQuota           = errors.New("upload quota reached")
	errWriteFailed     = errors.New("upload could not be saved")
	errStopped         = errors.New("session stopped")
)

// Direction is which way a transfer moved data, from the device's side.
type Direction string

// The two transfer directions.
const (
	Download Direction = "download"
	Upload   Direction = "upload"
)

// StopReason says why a session ended.
type StopReason string

// The ways a session ends.
const (
	StopOperator StopReason = "operator"
	StopIdle     StopReason = "idle"
	StopExpired  StopReason = "expired"
	StopShutdown StopReason = "shutdown"
)

// Transfer is one finished, failed or refused request.
type Transfer struct {
	Direction Direction
	Filename  string
	Remote    string
	Bytes     int64
	Err       error
}

// AuditResource is the audit log resource type of every TFTP event.
const AuditResource = "tftp"

// AuditEvent is one start, stop or transfer for the audit log. Resource is the
// interface for a start or stop and the filename for a transfer; a transfer
// has no user, so Remote, the device's address, identifies it.
type AuditEvent struct {
	Action    string
	User      string
	Resource  string
	Remote    string
	UserAgent string
	Detail    any
}

// Options are what the admin chooses when starting a session.
type Options struct {
	Interface   string
	AllowUpload bool
	// AllowFrom lists the IPv4 prefixes a request may come from. Empty means
	// the subnets configured on Interface; a device behind a router needs its
	// prefix named here.
	AllowFrom []netip.Prefix
}

// Limits bound what one session may consume. Zero fields take the Default
// constants; only tests set others.
type Limits struct {
	MaxTransfers int64
	MaxLifetime  time.Duration
	MaxUploads   int64
	UploadQuota  int64
}

// Config is fixed for the Manager's lifetime.
type Config struct {
	// Dir is the one directory every session serves; Start creates it.
	Dir string
	// UploadDir receives uploads. It must not be inside Dir: nothing a
	// device writes is ever served back. Start creates it.
	UploadDir string
	// IdleTimeout defaults to DefaultIdleTimeout.
	IdleTimeout time.Duration
	// Port defaults to [Port]. Only tests, which cannot bind 69, set it.
	Port   int
	Limits Limits
	// OnTransfer and OnStop are called once per transfer and per session.
	// Either may be nil.
	OnTransfer func(Transfer)
	OnStop     func(Status, StopReason)
	Logger     *slog.Logger
}

// Status is a session's state, or Running false when there is none.
type Status struct {
	Running            bool       `json:"running"`
	Interface          string     `json:"interface,omitempty"`
	Address            string     `json:"address,omitempty"`
	Port               int        `json:"port,omitempty"`
	Directory          string     `json:"directory,omitempty"`
	UploadDirectory    string     `json:"uploadDirectory,omitempty"`
	AllowUpload        bool       `json:"allowUpload"`
	AllowFrom          []string   `json:"allowFrom,omitempty"`
	StartedAt          *time.Time `json:"startedAt,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	IdleTimeoutSeconds int        `json:"idleTimeoutSeconds,omitempty"`
	ActiveTransfers    int64      `json:"activeTransfers"`
}

// Manager owns at most one session.
type Manager struct {
	cfg  Config
	mu   sync.Mutex
	sess *session
}

// NewManager returns a Manager with no session running.
func NewManager(cfg Config) *Manager {
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.Port == 0 {
		cfg.Port = Port
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	l := &cfg.Limits
	if l.MaxTransfers <= 0 {
		l.MaxTransfers = DefaultMaxTransfers
	}
	if l.MaxLifetime <= 0 {
		l.MaxLifetime = DefaultMaxLifetime
	}
	if l.MaxUploads <= 0 {
		l.MaxUploads = DefaultMaxUploads
	}
	if l.UploadQuota <= 0 {
		l.UploadQuota = DefaultUploadQuota
	}
	return &Manager{cfg: cfg}
}

type session struct {
	opts      Options
	allow     []netip.Prefix
	limits    Limits
	addr      net.IP
	port      int
	startedAt time.Time
	root      *os.Root
	inbox     *os.Root
	srv       *pintftp.Server
	served    chan struct{}
	stopped   chan struct{}
	stopping  bool
	active    atomic.Int64
	lastUsed  atomic.Int64
	uploads   atomic.Int64
	quota     atomic.Int64
	onXfer    func(Transfer)
}

// Start opens a session on the interface's first IPv4 address.
func (m *Manager) Start(opts Options) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sess != nil {
		return Status{}, ErrRunning
	}
	addr, subnets, err := interfaceIPv4(opts.Interface)
	if err != nil {
		return Status{}, err
	}
	allow := subnets
	if len(opts.AllowFrom) > 0 {
		allow = opts.AllowFrom
	}
	root, err := openDir(m.cfg.Dir)
	if err != nil {
		return Status{}, err
	}
	inbox, err := openDir(m.cfg.UploadDir)
	if err != nil {
		_ = root.Close()
		return Status{}, err
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: addr, Port: m.cfg.Port})
	if err != nil {
		_ = root.Close()
		_ = inbox.Close()
		bind := net.JoinHostPort(addr.String(), strconv.Itoa(m.cfg.Port))
		return Status{}, fmt.Errorf("%w %s: %w", ErrBind, bind, err)
	}

	s := &session{
		opts:      opts,
		allow:     allow,
		limits:    m.cfg.Limits,
		addr:      addr,
		port:      m.cfg.Port,
		startedAt: time.Now().UTC(),
		root:      root,
		inbox:     inbox,
		served:    make(chan struct{}),
		stopped:   make(chan struct{}),
		onXfer:    m.cfg.OnTransfer,
	}
	s.lastUsed.Store(s.startedAt.UnixNano())
	s.quota.Store(s.limits.UploadQuota)
	s.srv = pintftp.NewServer(s.read, s.write)
	s.srv.SetTimeout(transferTimeout)
	m.sess = s

	go func() {
		defer close(s.served)
		if serveErr := s.srv.Serve(conn); serveErr != nil {
			m.cfg.Logger.Warn("tftp serve ended", "error", serveErr)
		}
	}()
	go m.watchIdle(s)
	return m.statusOf(s), nil
}

// Stop ends the running session, aborting any transfer in flight.
func (m *Manager) Stop() error {
	return m.end(StopOperator)
}

// Close ends the running session, if any, for daemon shutdown.
func (m *Manager) Close() {
	_ = m.end(StopShutdown)
}

func (m *Manager) end(reason StopReason) error {
	m.mu.Lock()
	s := m.sess
	m.mu.Unlock()
	if s == nil {
		return ErrNotRunning
	}
	m.stop(s, reason)
	return nil
}

// Status reports the running session, if any.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sess == nil {
		return Status{}
	}
	return m.statusOf(m.sess)
}

func (m *Manager) statusOf(s *session) Status {
	started := s.startedAt
	expires := started.Add(s.limits.MaxLifetime)
	allow := make([]string, len(s.allow))
	for i, p := range s.allow {
		allow[i] = p.String()
	}
	return Status{
		Running:            true,
		Interface:          s.opts.Interface,
		Address:            s.addr.String(),
		Port:               s.port,
		Directory:          m.cfg.Dir,
		UploadDirectory:    m.cfg.UploadDir,
		AllowUpload:        s.opts.AllowUpload,
		AllowFrom:          allow,
		StartedAt:          &started,
		ExpiresAt:          &expires,
		IdleTimeoutSeconds: int(m.cfg.IdleTimeout / time.Second),
		ActiveTransfers:    s.active.Load(),
	}
}

// stop tears s down once; a second caller (idle racing an operator) returns
// without effect. s stays the Manager's session until it is fully down, so a
// Start cannot race its socket and Running false means the port is free.
func (m *Manager) stop(s *session, reason StopReason) {
	m.mu.Lock()
	if m.sess != s || s.stopping {
		m.mu.Unlock()
		return
	}
	s.stopping = true
	status := m.statusOf(s)
	close(s.stopped)
	m.mu.Unlock()

	// Shutdown waits for transfers; closing stopped makes each one fail its
	// next block, so the wait is bounded by one block's retries, not a file.
	s.srv.Shutdown()
	<-s.served
	_ = s.root.Close()
	_ = s.inbox.Close()
	m.cfg.Logger.Info("tftp session stopped", "event", "tftp.stop", "reason", string(reason))
	if m.cfg.OnStop != nil {
		m.cfg.OnStop(status, reason)
	}

	m.mu.Lock()
	m.sess = nil
	m.mu.Unlock()
}

// watchIdle stops s once no transfer has completed for the idle timeout, and
// in any case once it reaches its maximum lifetime. A transfer in flight holds
// the session open past the idle timeout, never past the lifetime.
func (m *Manager) watchIdle(s *session) {
	expires := s.startedAt.Add(s.limits.MaxLifetime)
	for {
		left := time.Until(expires)
		if left <= 0 {
			m.stop(s, StopExpired)
			return
		}
		wait := m.cfg.IdleTimeout - time.Since(time.Unix(0, s.lastUsed.Load()))
		if wait <= 0 {
			if s.active.Load() == 0 {
				m.stop(s, StopIdle)
				return
			}
			wait = m.cfg.IdleTimeout
		}
		wait = min(wait, left)
		select {
		case <-time.After(wait):
		case <-s.stopped:
			return
		}
	}
}

// begin admits a request from remote and marks it in flight, or refuses it
// at once so the library closes the socket it opened for the request. The
// returned func marks the transfer done; only a transfer that completed counts
// as use for the idle timeout, so refused and failed requests cannot keep a
// session alive.
func (s *session) begin(remote netip.Addr) (func(error), error) {
	if !s.allowed(remote) {
		return nil, ErrAccessDenied
	}
	if s.active.Add(1) > s.limits.MaxTransfers {
		s.active.Add(-1)
		return nil, ErrBusy
	}
	return func(err error) {
		if err == nil {
			s.lastUsed.Store(time.Now().UnixNano())
		}
		s.active.Add(-1)
	}, nil
}

func (s *session) allowed(remote netip.Addr) bool {
	for _, p := range s.allow {
		if p.Contains(remote) {
			return true
		}
	}
	return false
}

func (s *session) read(name string, rf io.ReaderFrom) error {
	remote := remoteOf(rf)
	var n int64
	done, err := s.begin(remote)
	if err == nil {
		n, err = s.download(name, rf)
		done(err)
	}
	s.report(Transfer{Direction: Download, Filename: name, Remote: addrString(remote), Bytes: n, Err: err})
	return err
}

func (s *session) download(name string, rf io.ReaderFrom) (int64, error) {
	f, err := s.root.Open(relative(name))
	if err != nil {
		return 0, ErrNotFound
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return 0, ErrNotFound
	}
	// stopReader hides f's Seek, which is how the library would have found
	// the size for the tsize option.
	if ot, ok := rf.(pintftp.OutgoingTransfer); ok {
		ot.SetSize(info.Size())
	}
	return rf.ReadFrom(stopReader{r: f, stopped: s.stopped})
}

func (s *session) write(name string, wt io.WriterTo) error {
	remote := remoteOf(wt)
	var n int64
	done, err := s.begin(remote)
	if err == nil {
		n, err = s.upload(name, wt)
		done(err)
	}
	s.report(Transfer{Direction: Upload, Filename: name, Remote: addrString(remote), Bytes: n, Err: err})
	return err
}

// upload receives into a hidden partial file and publishes it under its name
// only once complete, so no reader ever sees half a file. The hard link
// publishes atomically and fails if the name exists, where a rename would
// replace it.
func (s *session) upload(name string, wt io.WriterTo) (int64, error) {
	if !s.opts.AllowUpload {
		return 0, ErrUploadsDisabled
	}
	if s.uploads.Add(1) > s.limits.MaxUploads {
		return 0, ErrQuota
	}
	if it, ok := wt.(pintftp.IncomingTransfer); ok {
		if size, known := it.Size(); known && size > MaxUploadBytes {
			return 0, ErrTooLarge
		}
	}
	rel := relative(name)
	if _, err := s.inbox.Lstat(rel); err == nil {
		return 0, ErrExists
	}
	dir, base := path.Split(rel)
	partial := dir + "." + base + ".partial"
	f, err := s.inbox.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_EXCL, uploadMode)
	if errors.Is(err, fs.ErrExist) {
		return 0, ErrExists
	}
	if err != nil {
		return 0, ErrNotFound
	}
	defer func() { _ = s.inbox.Remove(partial) }()

	n, err := wt.WriteTo(&capWriter{w: f, stopped: s.stopped, left: MaxUploadBytes, quota: &s.quota})
	if closeErr := f.Close(); err == nil && closeErr != nil {
		err = errWriteFailed
	}
	if err == nil {
		err = s.inbox.Link(partial, rel)
		switch {
		case errors.Is(err, fs.ErrExist):
			err = ErrExists
		case err != nil:
			err = errWriteFailed
		}
	}
	if err != nil {
		// What failed is gone from disk, so it no longer counts against the
		// session's quota.
		s.quota.Add(n)
	}
	return n, err
}

func (s *session) report(t Transfer) {
	if s.onXfer != nil {
		s.onXfer(t)
	}
}

// relative turns a request name into a path under the root. Devices commonly
// send "/name"; the root refuses absolute paths, and the leading slash means
// the served directory, not the host's.
func relative(name string) string {
	return strings.TrimLeft(name, "/")
}

func remoteOf(v any) netip.Addr {
	type remote interface{ RemoteAddr() net.UDPAddr }
	if r, ok := v.(remote); ok {
		ua := r.RemoteAddr()
		if a, valid := netip.AddrFromSlice(ua.IP); valid {
			return a.Unmap()
		}
	}
	return netip.Addr{}
}

func addrString(a netip.Addr) string {
	if !a.IsValid() {
		return ""
	}
	return a.String()
}

func openDir(dir string) (*os.Root, error) {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return nil, fmt.Errorf("create %s: %w", dir, err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dir, err)
	}
	return root, nil
}

// stopReader fails a download once its session stops.
type stopReader struct {
	r       io.Reader
	stopped <-chan struct{}
}

func (g stopReader) Read(p []byte) (int, error) {
	select {
	case <-g.stopped:
		return 0, errStopped
	default:
		return g.r.Read(p)
	}
}

// capWriter fails an upload once its session stops, it passes its own cap,
// or the session's uploads together pass the quota it debits.
type capWriter struct {
	w       io.Writer
	stopped <-chan struct{}
	left    int64
	quota   *atomic.Int64
}

func (g *capWriter) Write(p []byte) (int, error) {
	select {
	case <-g.stopped:
		return 0, errStopped
	default:
	}
	size := int64(len(p))
	if size > g.left {
		return 0, ErrTooLarge
	}
	if g.quota.Add(-size) < 0 {
		g.quota.Add(size)
		return 0, ErrQuota
	}
	n, err := g.w.Write(p)
	g.left -= int64(n)
	g.quota.Add(size - int64(n))
	return n, err
}

// interfaceIPv4 returns the interface's first IPv4 address, to bind, and
// every IPv4 subnet on it, to answer.
func interfaceIPv4(name string) (net.IP, []netip.Prefix, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s: %w", ErrNoInterface, name, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, fmt.Errorf("interface %q addresses: %w", name, err)
	}
	var first net.IP
	var subnets []netip.Prefix
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipn.IP.To4()
		if ip4 == nil {
			continue
		}
		if first == nil {
			first = ip4
		}
		mask := ipn.Mask
		if len(mask) == net.IPv6len {
			mask = mask[net.IPv6len-net.IPv4len:]
		}
		ones, _ := mask.Size()
		subnets = append(subnets, netip.PrefixFrom(netip.AddrFrom4([4]byte(ip4)), ones).Masked())
	}
	if first == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrNoIPv4, name)
	}
	return first, subnets, nil
}
