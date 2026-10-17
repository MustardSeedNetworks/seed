package deviceconfig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	// maxConfigBytes caps one configuration. The largest chassis
	// configurations run to a few megabytes; anything past this is a device
	// streaming something other than its configuration.
	maxConfigBytes = 16 << 20
	// maxStderrBytes keeps enough of a failed command's error output to say
	// why it failed.
	maxStderrBytes = 4 << 10
)

// SSHFetcher retrieves the configuration over SSH: by running the CLI command
// on an exec channel, or with a NETCONF <get-config> on the netconf subsystem.
// It is also the push dry runner.
type SSHFetcher struct{}

// Fetch checks the host key against the pin, logs in, and retrieves the
// configuration.
func (SSHFetcher) Fetch(ctx context.Context, req FetchRequest) (FetchResult, error) {
	var output string
	err := withSession(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(session *ssh.Session) error {
			var runErr error
			if req.NETCONF {
				output, runErr = getRunningConfig(session)
			} else {
				output, runErr = runCommand(session, req.Command)
			}
			return runErr
		})
	if err != nil {
		return FetchResult{}, err
	}
	return FetchResult{Output: output}, nil
}

// DryRun previews a candidate on the device and changes nothing. Only NETCONF
// has a dialogue so far; Pushes refuses the other platforms before a login.
func (SSHFetcher) DryRun(ctx context.Context, req DryRunRequest) (DryRunResult, error) {
	if req.Platform != PlatformNETCONF {
		return DryRunResult{}, &PlanRefusedError{Reason: string(req.Platform) + " has no dry run"}
	}
	var res DryRunResult
	err := withSession(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(session *ssh.Session) error {
			var runErr error
			res, runErr = netconfDryRun(session, req.Mode, req.Candidate)
			return runErr
		})
	return res, err
}

// withSession checks the host key against the pin, logs in with password or
// keyboard-interactive authentication (network operating systems offer one or
// the other for the same password), and runs fn on one session. The host key
// is checked during key exchange, before any authentication method runs, so
// an unpinned or mismatched key never receives the password.
func withSession(
	ctx context.Context, host string, port int, user, password, pinned string, fn func(*ssh.Session) error,
) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer func() { _ = conn.Close() }()
	// The SSH library has no context: closing the socket is what unblocks a
	// handshake or a command when the run is cancelled or the device times out.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	cfg := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			presented := ssh.FingerprintSHA256(key)
			switch {
			case pinned == "":
				return &HostKeyUnconfirmedError{Fingerprint: presented}
			case presented != pinned:
				return fmt.Errorf("%w: device presented %s, pinned %s",
					ErrHostKeyMismatch, presented, pinned)
			}
			return nil
		},
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		return classifyHandshake(ctx, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("%w: open session: %w", ErrCommand, err)
	}
	defer func() { _ = session.Close() }()

	err = fn(session)
	if ctx.Err() != nil {
		return fmt.Errorf("%w: %w", ErrUnreachable, ctx.Err())
	}
	return err
}

// runCommand runs command on the session's exec channel and returns its
// output.
func runCommand(session *ssh.Session, command string) (string, error) {
	stdout := &cappedBuffer{limit: maxConfigBytes}
	stderr := &cappedBuffer{limit: maxStderrBytes}
	session.Stdout = stdout
	session.Stderr = stderr
	runErr := session.Run(command)
	if stdout.overflow {
		return "", fmt.Errorf("%w: output exceeds %d bytes", ErrCommand, maxConfigBytes)
	}
	if runErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("%w: %w", ErrCommand, runErr)
		}
		return "", fmt.Errorf("%w: %w: %s", ErrCommand, runErr, msg)
	}
	return stdout.String(), nil
}

// classifyHandshake names a failed SSH handshake. The library has no typed
// authentication error; "unable to authenticate" is the message its client
// returns once every offered method has been refused.
func classifyHandshake(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, ErrHostKeyMismatch), errors.As(err, new(*HostKeyUnconfirmedError)):
		return err
	case strings.Contains(err.Error(), "unable to authenticate"):
		return fmt.Errorf("%w: %w", ErrAuth, err)
	case ctx.Err() != nil:
		return fmt.Errorf("%w: %w", ErrUnreachable, ctx.Err())
	}
	return fmt.Errorf("ssh handshake: %w", err)
}

// cappedBuffer keeps at most limit bytes and records that more arrived. It
// keeps accepting writes so the remote side is not left blocked on a full
// channel window.
type cappedBuffer struct {
	buf      bytes.Buffer
	limit    int
	overflow bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	room := b.limit - b.buf.Len()
	if len(p) > room {
		b.overflow = true
		if room > 0 {
			_, _ = b.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *cappedBuffer) String() string { return b.buf.String() }
