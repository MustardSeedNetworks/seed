package deviceconfig

// An IOS replace reads its candidate from a file on the device (ADR-0035
// §6.5), which Seed copies there as an SCP sink: it runs "scp -t <path>" on
// the device and speaks the source side of the protocol. The protocol is a
// header line, the bytes and a byte of zero, each answered by the device
// with a zero byte, or a 1 or 2 and the reason on one line.

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/ssh"
)

// scpSink copies content to path on the device, readable by its owner only.
// The last acknowledgement is the device's word that the file is written.
func scpSink(client *ssh.Client, path string, content []byte) error {
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
		for _, part := range []string{fmt.Sprintf("C0600 %d %s\n", len(content), name), string(content) + "\x00"} {
			if _, err = io.WriteString(stdin, part); err != nil {
				return fmt.Errorf("%w: scp: write to the device: %w", ErrCommand, err)
			}
			if err = scpAck(r); err != nil {
				return err
			}
		}
		return nil
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
