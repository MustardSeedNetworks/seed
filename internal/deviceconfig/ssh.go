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

// SSHFetcher runs the configuration command over an SSH exec channel.
type SSHFetcher struct{}

// Fetch logs in with password or keyboard-interactive authentication (network
// operating systems offer one or the other for the same password), checks the
// host key against the pin, and runs req.Command.
func (SSHFetcher) Fetch(ctx context.Context, req FetchRequest) (FetchResult, error) {
	addr := net.JoinHostPort(req.Host, strconv.Itoa(req.Port))
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return FetchResult{}, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer func() { _ = conn.Close() }()
	// The SSH library has no context: closing the socket is what unblocks a
	// handshake or a command when the run is cancelled or the device times out.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	var presented string
	cfg := &ssh.ClientConfig{
		User: req.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Password),
			ssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = req.Password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			presented = ssh.FingerprintSHA256(key)
			if req.PinnedHostKey != "" && presented != req.PinnedHostKey {
				return fmt.Errorf("%w: device presented %s, pinned %s",
					ErrHostKeyMismatch, presented, req.PinnedHostKey)
			}
			return nil
		},
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		return FetchResult{}, classifyHandshake(ctx, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return FetchResult{}, fmt.Errorf("%w: open session: %w", ErrCommand, err)
	}
	defer func() { _ = session.Close() }()

	stdout := &cappedBuffer{limit: maxConfigBytes}
	stderr := &cappedBuffer{limit: maxStderrBytes}
	session.Stdout = stdout
	session.Stderr = stderr
	runErr := session.Run(req.Command)
	if ctx.Err() != nil {
		return FetchResult{}, fmt.Errorf("%w: %w", ErrUnreachable, ctx.Err())
	}
	if stdout.overflow {
		return FetchResult{}, fmt.Errorf("%w: output exceeds %d bytes", ErrCommand, maxConfigBytes)
	}
	if runErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return FetchResult{}, fmt.Errorf("%w: %w", ErrCommand, runErr)
		}
		return FetchResult{}, fmt.Errorf("%w: %w: %s", ErrCommand, runErr, msg)
	}
	return FetchResult{Output: stdout.String(), HostKey: presented}, nil
}

// classifyHandshake names a failed SSH handshake. The library has no typed
// authentication error; "unable to authenticate" is the message its client
// returns once every offered method has been refused.
func classifyHandshake(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, ErrHostKeyMismatch):
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
