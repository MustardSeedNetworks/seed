package deviceconfig

// An IOS replace reads its candidate from a file on the device, and an image
// is staged as a file on the device (ADR-0035 §2, §3). Seed copies both as an
// SCP sink: it runs "scp -t <path>" on the device and speaks the source side
// of the protocol. The protocol is a header line, the bytes and a byte of
// zero, each answered by the device with a zero byte, or a 1 or 2 and the
// reason on one line.

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/ssh"
)

// scpSink copies size bytes of body to path on the device, readable by its
// owner only. The body is streamed, so an image is never held in memory. The
// last acknowledgement is the device's word that the file is written.
func scpSink(client *ssh.Client, path string, size int64, body io.Reader) error {
	return onSession(client, func(session *ssh.Session) error {
		stdin, err := session.StdinPipe()
		if err != nil {
			return fmt.Errorf("%w: scp: %w", ErrCommand, err)
		}
		stdout, err := session.StdoutPipe()
		if err != nil {
			return fmt.Errorf("%w: scp: %w", ErrCommand, err)
		}
		if err = session.Start("scp -t " + path); err != nil {
			return fmt.Errorf("%w: start scp on the device: %w", ErrCommand, err)
		}
		r := bufio.NewReader(stdout)
		if err = scpAck(r); err != nil {
			return err
		}
		name := path[strings.LastIndexAny(path, ":/")+1:]
		if _, err = fmt.Fprintf(stdin, "C0600 %d %s\n", size, name); err != nil {
			return fmt.Errorf("%w: scp: write to the device: %w", ErrCommand, err)
		}
		if err = scpAck(r); err != nil {
			return err
		}
		n, err := io.Copy(stdin, io.LimitReader(body, size))
		switch {
		case err != nil:
			return fmt.Errorf("%w: scp: write to the device: %w", ErrCommand, err)
		case n != size:
			return fmt.Errorf("%w: scp: the file ended after %d of %d bytes", ErrCommand, n, size)
		}
		if _, err = stdin.Write([]byte{0}); err != nil {
			return fmt.Errorf("%w: scp: write to the device: %w", ErrCommand, err)
		}
		return scpAck(r)
	})
}

// scpAck reads the device's answer to the last thing sent.
func scpAck(r *bufio.Reader) error {
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("%w: scp: the device closed the copy: %w", ErrCommand, err)
	}
	if b == 0 {
		return nil
	}
	msg, _ := r.ReadString('\n')
	return fmt.Errorf("%w: scp: the device refused the copy: %s", ErrCommand, strings.TrimSpace(msg))
}
