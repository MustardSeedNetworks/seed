// Package tftp serves files to network devices over TFTP (RFC 1350) for the
// length of one operator-started session (P-D4).
//
// TFTP has no authentication, so a session is kept as small as the protocol
// allows: nothing listens until an admin starts it and nothing restarts it with
// the daemon; it binds one interface's address, serves one directory through
// an [os.Root], refuses writes unless the session allows uploads, never
// overwrites a file, and stops itself once nothing has used it for the idle
// timeout.
package tftp

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
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

// Options are what the admin chooses when starting a session.
type Options struct {
	Interface   string
	AllowUpload bool
}

// Config is fixed for the Manager's lifetime.
type Config struct {
	// Dir is the one directory every session serves; Start creates it.
	Dir string
	// IdleTimeout defaults to DefaultIdleTimeout.
	IdleTimeout time.Duration
	// Port defaults to [Port]. Only tests, which cannot bind 69, set it.
	Port int
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
	AllowUpload        bool       `json:"allowUpload"`
	StartedAt          *time.Time `json:"startedAt,omitempty"`
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
	return &Manager{cfg: cfg}
}

type session struct {
	opts      Options
	addr      net.IP
	port      int
	startedAt time.Time
	root      *os.Root
	srv       *pintftp.Server
	served    chan struct{}
	stopped   chan struct{}
	stopping  bool
	active    atomic.Int64
	lastUsed  atomic.Int64
	onXfer    func(Transfer)
}

// Start opens a session on the interface's first IPv4 address.
func (m *Manager) Start(opts Options) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sess != nil {
		return Status{}, ErrRunning
	}
	addr, err := interfaceIPv4(opts.Interface)
	if err != nil {
		return Status{}, err
	}
	if err = os.MkdirAll(m.cfg.Dir, dirMode); err != nil {
		return Status{}, fmt.Errorf("create %s: %w", m.cfg.Dir, err)
	}
	root, err := os.OpenRoot(m.cfg.Dir)
	if err != nil {
		return Status{}, fmt.Errorf("open %s: %w", m.cfg.Dir, err)
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: addr, Port: m.cfg.Port})
	if err != nil {
		_ = root.Close()
		bind := net.JoinHostPort(addr.String(), strconv.Itoa(m.cfg.Port))
		return Status{}, fmt.Errorf("%w %s: %w", ErrBind, bind, err)
	}

	s := &session{
		opts:      opts,
		addr:      addr,
		port:      m.cfg.Port,
		startedAt: time.Now().UTC(),
		root:      root,
		served:    make(chan struct{}),
		stopped:   make(chan struct{}),
		onXfer:    m.cfg.OnTransfer,
	}
	s.lastUsed.Store(s.startedAt.UnixNano())
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
	return Status{
		Running:            true,
		Interface:          s.opts.Interface,
		Address:            s.addr.String(),
		Port:               s.port,
		Directory:          m.cfg.Dir,
		AllowUpload:        s.opts.AllowUpload,
		StartedAt:          &started,
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
	m.cfg.Logger.Info("tftp session stopped", "event", "tftp.stop", "reason", string(reason))
	if m.cfg.OnStop != nil {
		m.cfg.OnStop(status, reason)
	}

	m.mu.Lock()
	m.sess = nil
	m.mu.Unlock()
}

// watchIdle stops s once no transfer has run for the idle timeout. A transfer
// in flight holds the session open however long it takes.
func (m *Manager) watchIdle(s *session) {
	for {
		wait := m.cfg.IdleTimeout - time.Since(time.Unix(0, s.lastUsed.Load()))
		if wait <= 0 {
			if s.active.Load() == 0 {
				m.stop(s, StopIdle)
				return
			}
			wait = m.cfg.IdleTimeout
		}
		select {
		case <-time.After(wait):
		case <-s.stopped:
			return
		}
	}
}

// begin marks a transfer in flight; the returned func marks it done.
func (s *session) begin() func() {
	s.active.Add(1)
	s.lastUsed.Store(time.Now().UnixNano())
	return func() {
		s.lastUsed.Store(time.Now().UnixNano())
		s.active.Add(-1)
	}
}

func (s *session) read(name string, rf io.ReaderFrom) error {
	defer s.begin()()
	n, err := s.download(name, rf)
	s.report(Transfer{Direction: Download, Filename: name, Remote: remoteOf(rf), Bytes: n, Err: err})
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
	defer s.begin()()
	n, err := s.upload(name, wt)
	s.report(Transfer{Direction: Upload, Filename: name, Remote: remoteOf(wt), Bytes: n, Err: err})
	return err
}

func (s *session) upload(name string, wt io.WriterTo) (int64, error) {
	if !s.opts.AllowUpload {
		return 0, ErrUploadsDisabled
	}
	if it, ok := wt.(pintftp.IncomingTransfer); ok {
		if size, known := it.Size(); known && size > MaxUploadBytes {
			return 0, ErrTooLarge
		}
	}
	rel := relative(name)
	f, err := s.root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, uploadMode)
	if errors.Is(err, fs.ErrExist) {
		return 0, ErrExists
	}
	if err != nil {
		return 0, ErrNotFound
	}
	n, err := wt.WriteTo(&capWriter{w: f, stopped: s.stopped, left: MaxUploadBytes})
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = s.root.Remove(rel)
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

func remoteOf(v any) string {
	type remote interface{ RemoteAddr() net.UDPAddr }
	if r, ok := v.(remote); ok {
		addr := r.RemoteAddr()
		return addr.IP.String()
	}
	return ""
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

// capWriter fails an upload once its session stops or it passes its cap.
type capWriter struct {
	w       io.Writer
	stopped <-chan struct{}
	left    int64
}

func (g *capWriter) Write(p []byte) (int, error) {
	select {
	case <-g.stopped:
		return 0, errStopped
	default:
	}
	if int64(len(p)) > g.left {
		return 0, ErrTooLarge
	}
	n, err := g.w.Write(p)
	g.left -= int64(n)
	return n, err
}

func interfaceIPv4(name string) (net.IP, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrNoInterface, name, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("interface %q addresses: %w", name, err)
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok {
			if ip4 := ipn.IP.To4(); ip4 != nil {
				return ip4, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoIPv4, name)
}
