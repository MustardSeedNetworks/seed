package deviceconfig_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

// sshDevice is an in-process SSH server standing in for a switch: it accepts
// one user and password and answers exec requests from a command table.
type sshDevice struct {
	addr        string
	fingerprint string
}

type deviceOpts struct {
	password string
	// keyboardInteractiveOnly offers no "password" method, as some network
	// operating systems do.
	keyboardInteractiveOnly bool
	commands                map[string]string
}

func startSSHDevice(t *testing.T, opts deviceOpts) sshDevice {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)

	cfg := &ssh.ServerConfig{}
	if opts.keyboardInteractiveOnly {
		cfg.KeyboardInteractiveCallback = func(_ ssh.ConnMetadata, challenge ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
			answers, chErr := challenge("", "", []string{"Password: "}, []bool{false})
			if chErr != nil || len(answers) != 1 || answers[0] != opts.password {
				return nil, errors.New("denied")
			}
			return &ssh.Permissions{}, nil
		}
	} else {
		cfg.PasswordCallback = func(_ ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if string(pw) != opts.password {
				return nil, errors.New("denied")
			}
			return &ssh.Permissions{}, nil
		}
	}
	cfg.AddHostKey(signer)

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go serveSSH(conn, cfg, opts.commands)
		}
	}()
	return sshDevice{addr: ln.Addr().String(), fingerprint: ssh.FingerprintSHA256(signer.PublicKey())}
}

func serveSSH(conn net.Conn, cfg *ssh.ServerConfig, commands map[string]string) {
	defer func() { _ = conn.Close() }()
	_, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		ch, chReqs, acceptErr := nc.Accept()
		if acceptErr != nil {
			return
		}
		for req := range chReqs {
			if req.Type != "exec" {
				_ = req.Reply(false, nil)
				continue
			}
			_ = req.Reply(true, nil)
			// The exec payload is an SSH string: a uint32 length and the bytes.
			status := uint32(0)
			if out, ok := commands[string(req.Payload[4:])]; ok {
				_, _ = ch.Write([]byte(out))
			} else {
				_, _ = ch.Stderr().Write([]byte("% Invalid input detected"))
				status = 1
			}
			payload := make([]byte, 4)
			binary.BigEndian.PutUint32(payload, status)
			_, _ = ch.SendRequest("exit-status", false, payload)
			_ = ch.Close()
		}
	}
}

func (d sshDevice) request(t *testing.T, password, command, pinned string) deviceconfig.FetchRequest {
	t.Helper()
	host, portStr, err := net.SplitHostPort(d.addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return deviceconfig.FetchRequest{
		Host: host, Port: port, User: "backup", Password: password, Command: command, PinnedHostKey: pinned,
	}
}

func fetchCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestSSHFetcherReturnsTheConfigurationAndHostKey(t *testing.T) {
	t.Parallel()
	for _, kbd := range []bool{false, true} {
		dev := startSSHDevice(t, deviceOpts{
			password:                "s3cret",
			keyboardInteractiveOnly: kbd,
			commands:                map[string]string{"show running-config": "hostname core\r\n"},
		})

		res, err := deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t), dev.request(t, "s3cret", "show running-config", ""))
		require.NoError(t, err, "keyboard-interactive only = %v", kbd)
		require.Equal(t, "hostname core\r\n", res.Output)
		require.Equal(t, dev.fingerprint, res.HostKey)

		// The pinned key is accepted on the next login.
		_, err = deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t),
			dev.request(t, "s3cret", "show running-config", dev.fingerprint))
		require.NoError(t, err)
	}
}

func TestSSHFetcherClassifiesFailures(t *testing.T) {
	t.Parallel()
	dev := startSSHDevice(t, deviceOpts{
		password: "s3cret",
		commands: map[string]string{"show running-config": "hostname core\n"},
	})

	_, err := deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t), dev.request(t, "wrong", "show running-config", ""))
	require.ErrorIs(t, err, deviceconfig.ErrAuth)

	_, err = deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t),
		dev.request(t, "s3cret", "show running-config", "SHA256:someoneelse"))
	require.ErrorIs(t, err, deviceconfig.ErrHostKeyMismatch)

	_, err = deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t), dev.request(t, "s3cret", "show configuration", ""))
	require.ErrorIs(t, err, deviceconfig.ErrCommand)
	require.ErrorContains(t, err, "Invalid input detected", "the device's own error is kept")

	// A port nothing listens on.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	closed := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	_, err = deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t), deviceconfig.FetchRequest{
		Host: "127.0.0.1", Port: closed, User: "backup", Password: "x", Command: "show running-config",
	})
	require.ErrorIs(t, err, deviceconfig.ErrUnreachable)
}

// TestSSHFetcherHonoursCancellation: a device that accepts TCP and never
// speaks SSH must not hold a run past its deadline.
func TestSSHFetcherHonoursCancellation(t *testing.T) {
	t.Parallel()
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			// Read and say nothing until the client gives up and closes.
			go func() {
				_, _ = io.Copy(io.Discard, conn)
				_ = conn.Close()
			}()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = deviceconfig.SSHFetcher{}.Fetch(ctx, deviceconfig.FetchRequest{
		Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, User: "u", Password: "p", Command: "c",
	})
	require.ErrorIs(t, err, deviceconfig.ErrUnreachable)
	require.Less(t, time.Since(start), 5*time.Second)
}
