package deviceconfig

// The Junos and EOS dry runs (ADR-0035) drive the device's own CLI in an
// interactive shell: the candidate is loaded into a private candidate (Junos)
// or a configuration session (EOS), the device prints its own diff, and the
// candidate is thrown away. Both are scoped to this login, so nothing another
// user is editing is touched, and a Junos private candidate dies with the
// session if the discard never arrives.

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/ssh"
)

const (
	// promptTail is how much of the newest output is searched for a prompt.
	// A prompt is the last line, so the rest of a long reply is never rescanned.
	promptTail = 512
	// maxDeviceMessages caps the device errors one failure reports.
	maxDeviceMessages = 3
	// readChunk is one read from the shell.
	readChunk = 32 << 10
	// The terminal is wide so that neither CLI wraps a long line it echoes.
	termRows, termCols = 200, 511
)

var (
	// anyPrompt is the first prompt after login, before its name is known.
	anyPrompt = regexp.MustCompile(`(?:^|\n)[^\n]*[>#] ?\z`)
	// junosLoadReady is Junos waiting for the configuration text of a load.
	junosLoadReady = regexp.MustCompile(`\[Type \^D at a new line to end input\]\n?\z`)
	// enablePassword is EOS asking for an enable password the push credential
	// does not have.
	enablePassword = regexp.MustCompile(`(?i)password: ?\z`)
	// junosLoadError is one rejected line of a Junos load:
	// "terminal:3:(12) syntax error: <token>". The token is configuration
	// text and is not kept.
	junosLoadError = regexp.MustCompile(`^terminal:(\d+):\(\d+\) ([^:]+)`)
)

// eosSessionControl reports whether an EOS candidate line starting with word
// would end the configuration session (committing it, in the case of commit)
// or run an exec command, rather than add configuration to it.
func eosSessionControl(word string) bool {
	switch word {
	case "abort", "bash", "clear", "commit", "configure", "copy", "delete", "disable", "do", "enable", "end",
		"exit", "reload", "rename", "rollback", "send", "terminal", "write":
		return true
	}
	return false
}

// checkCLICandidate refuses a CLI candidate that could do more than load
// configuration. The device reads it as typed input, so a control character
// is a keystroke (^D ends a Junos load and ^Z leaves EOS configuration mode),
// and an EOS line is a command.
func checkCLICandidate(platform Platform, candidate string) error {
	for i, line := range strings.Split(strings.ReplaceAll(candidate, "\r\n", "\n"), "\n") {
		for _, r := range line {
			if unicode.IsControl(r) && r != '\t' {
				return fmt.Errorf("candidate line %d contains a control character", i+1)
			}
		}
		if platform != PlatformAristaEOS {
			continue
		}
		if word, _, _ := strings.Cut(strings.TrimSpace(line), " "); eosSessionControl(word) {
			return fmt.Errorf(
				"candidate line %d starts with %q, which would leave the configuration session",
				i+1,
				word,
			)
		}
	}
	return nil
}

// cliShell is one interactive CLI session. Output is read until it ends at
// the device's prompt, whose name is learned at login so that configuration
// text is never mistaken for one.
type cliShell struct {
	w       io.Writer
	r       io.Reader
	pending []byte
	// prompt matches the device's prompt in any mode; last is the prompt the
	// most recent read ended at.
	prompt *regexp.Regexp
	last   string
}

func openCLIShell(session *ssh.Session) (*cliShell, error) {
	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCommand, err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCommand, err)
	}
	// A terminal is what puts both CLIs in interactive mode, where a load
	// reads its text from the session.
	if err = session.RequestPty("vt100", termRows, termCols, ssh.TerminalModes{}); err != nil {
		return nil, fmt.Errorf("%w: request a terminal: %w", ErrCommand, err)
	}
	if err = session.Shell(); err != nil {
		return nil, fmt.Errorf("%w: start the CLI: %w", ErrCommand, err)
	}
	s := &cliShell{w: stdin, r: stdout}
	if _, err = s.expect(anyPrompt); err != nil {
		return nil, err
	}
	name := strings.TrimRight(s.last, "># ")
	s.prompt = regexp.MustCompile(`(?:^|\n)` + regexp.QuoteMeta(name) + `(?:\([^)\n]*\))?[>#] ?\z`)
	return s, nil
}

// expect reads until the output ends with a match of re and returns what was
// read, with carriage returns removed.
func (s *cliShell) expect(re *regexp.Regexp) (string, error) {
	chunk := make([]byte, readChunk)
	for {
		tail := s.pending[max(0, len(s.pending)-promptTail):]
		tailText := strings.ReplaceAll(string(tail), "\r", "")
		if loc := re.FindStringIndex(tailText); loc != nil {
			text := strings.ReplaceAll(string(s.pending), "\r", "")
			s.last = strings.TrimPrefix(tailText[loc[0]:], "\n")
			s.pending = s.pending[:0]
			return text, nil
		}
		if len(s.pending) > maxConfigBytes {
			return "", fmt.Errorf("%w: output exceeds %d bytes", ErrCommand, maxConfigBytes)
		}
		n, err := s.r.Read(chunk)
		s.pending = append(s.pending, chunk[:n]...)
		if err != nil {
			return "", fmt.Errorf("%w: the device closed the session: %w", ErrCommand, err)
		}
	}
}

// readThrough reads prompt after prompt until marker has been seen, so a
// prompt left over from an earlier command, or one echoed inside a long
// reply, does not end the read early. It returns what was read, without the
// final prompt, and where the last marker starts in it: the device's own line
// comes after any echo of the same text.
func (s *cliShell) readThrough(marker string) (string, int, error) {
	var acc strings.Builder
	for {
		from := max(0, acc.Len()-len(marker))
		text, err := s.expect(s.prompt)
		if err != nil {
			return "", 0, err
		}
		acc.WriteString(text)
		if acc.Len() > maxConfigBytes {
			return "", 0, fmt.Errorf("%w: output exceeds %d bytes", ErrCommand, maxConfigBytes)
		}
		// Only the new text is searched: a candidate's echo can end at a
		// prompt once per line.
		all := acc.String()
		if i := strings.LastIndex(all[from:], marker); i >= 0 {
			return strings.TrimSuffix(all, s.last), from + i, nil
		}
	}
}

// sendThrough writes text while reading through marker. The device echoes
// what it reads, and a device's receive window can be smaller than a
// candidate: written first, the echo would fill Seed's window while the write
// still waited on the device to read.
func (s *cliShell) sendThrough(text, marker string) (string, int, error) {
	errc := make(chan error, 1)
	go func() { errc <- s.send(text) }()
	all, i, err := s.readThrough(marker)
	if err != nil {
		return "", 0, err
	}
	// The marker follows the last of text, so the write has finished.
	if err = <-errc; err != nil {
		return "", 0, err
	}
	return all, i, nil
}

func (s *cliShell) send(text string) error {
	if _, err := io.WriteString(s.w, text); err != nil {
		return fmt.Errorf("%w: write to the device: %w", ErrCommand, err)
	}
	return nil
}

// run runs one command and returns its output: what followed its echo,
// without the prompt.
func (s *cliShell) run(command string) (string, error) {
	if err := s.send(command + "\n"); err != nil {
		return "", err
	}
	all, i, err := s.readThrough(command + "\n")
	if err != nil {
		return "", err
	}
	return all[i+len(command)+1:], nil
}

// junosDryRun loads candidate into a private candidate, has the device
// compare it with the running configuration, and rolls it back. Replace is
// Junos's "load override": the candidate becomes the whole configuration.
func junosDryRun(session *ssh.Session, mode PushMode, candidate string) (DryRunResult, error) {
	s, err := openCLIShell(session)
	if err != nil {
		return DryRunResult{}, err
	}
	out, err := s.run("configure private")
	if err != nil {
		return DryRunResult{}, err
	}
	// Junos refuses a private candidate while the shared one has uncommitted
	// changes, so this is also what keeps a plan off someone else's work.
	if msgs := junosErrors(out); len(msgs) > 0 {
		return DryRunResult{}, &PlanRefusedError{
			Reason: "the device refused a private candidate: " + strings.Join(msgs, "; "),
		}
	}
	res, err := junosPreview(s, mode, candidate)
	// Leaving with changes would stop at a yes/no question; a failed rollback
	// leaves them to be discarded with the session instead.
	cleanupErr := s.junosCommand("rollback 0")
	if cleanupErr == nil {
		cleanupErr = s.junosCommand("exit configuration-mode")
	}
	_ = s.send("exit\n")
	if err != nil {
		return DryRunResult{}, err
	}
	if cleanupErr != nil {
		return DryRunResult{}, cleanupErr
	}
	return res, nil
}

func junosPreview(s *cliShell, mode PushMode, candidate string) (DryRunResult, error) {
	verb := "merge"
	if mode == PushReplace {
		verb = "override"
	}
	if err := s.send("load " + verb + " terminal\n"); err != nil {
		return DryRunResult{}, err
	}
	if _, err := s.expect(junosLoadReady); err != nil {
		return DryRunResult{}, err
	}
	text := strings.ReplaceAll(candidate, "\r\n", "\n")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	// ^D on a line of its own ends the load.
	const done = "load complete"
	all, i, err := s.sendThrough(text+"\x04", done)
	if err != nil {
		return DryRunResult{}, err
	}
	// "load complete (2 errors)": the rejected lines were reported above it.
	if summary, _, _ := strings.Cut(all[i+len(done):], "\n"); strings.TrimSpace(summary) != "" {
		return DryRunResult{}, fmt.Errorf("%w: the device rejected the candidate (%s%s)%s",
			ErrCommand, done, summary, joinMessages(junosLoadErrors(all[:i])))
	}
	diff, err := s.run("show | compare | no-more")
	if err != nil {
		return DryRunResult{}, err
	}
	if msgs := junosErrors(diff); len(msgs) > 0 {
		return DryRunResult{}, fmt.Errorf("%w: show | compare%s", ErrCommand, joinMessages(msgs))
	}
	return deviceDiff(trimEditBanner(diff)), nil
}

// junosCommand runs a configuration-mode command that prints nothing useful
// and reports the device's error if it printed one.
func (s *cliShell) junosCommand(command string) error {
	out, err := s.run(command)
	if err != nil {
		return err
	}
	if msgs := junosErrors(out); len(msgs) > 0 {
		return fmt.Errorf("%w: %s%s", ErrCommand, command, joinMessages(msgs))
	}
	return nil
}

// junosErrors returns the "error:" lines of a command's output.
func junosErrors(out string) []string {
	var msgs []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "syntax error") {
			msgs = append(msgs, line)
		}
	}
	return msgs
}

// junosLoadErrors names each rejected line of a load by its line number and
// the device's complaint, without the configuration text it quoted.
func junosLoadErrors(out string) []string {
	var msgs []string
	for line := range strings.SplitSeq(out, "\n") {
		if m := junosLoadError.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			msgs = append(msgs, "line "+m[1]+": "+strings.TrimSpace(m[2]))
		}
	}
	return msgs
}

// trimEditBanner drops the "[edit]" line Junos prints above every
// configuration-mode prompt.
func trimEditBanner(out string) string {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if strings.HasPrefix(lines[len(lines)-1], "[edit") {
		lines = lines[:len(lines)-1]
	}
	text := strings.Trim(strings.Join(lines, "\n"), "\n")
	if text == "" {
		return ""
	}
	return text + "\n"
}

// eosDryRun loads candidate into a new configuration session, has the device
// diff the session against the running configuration, and aborts the session.
// Replace starts the session from a clean configuration, so the candidate
// becomes the whole configuration.
func eosDryRun(session *ssh.Session, mode PushMode, candidate string) (DryRunResult, error) {
	s, err := openCLIShell(session)
	if err != nil {
		return DryRunResult{}, err
	}
	if strings.HasSuffix(strings.TrimSpace(s.last), ">") {
		if err = s.eosEnable(); err != nil {
			return DryRunResult{}, err
		}
	}
	name, err := eosSessionName()
	if err != nil {
		return DryRunResult{}, err
	}
	if err = s.eosCommand("configure session " + name); err != nil {
		return DryRunResult{}, err
	}
	res, err := eosPreview(s, mode, candidate)
	// A configuration session outlives the login on EOS, so it is aborted
	// explicitly even after a failure.
	abortErr := s.eosCommand("abort")
	_ = s.send("exit\n")
	if err != nil {
		return DryRunResult{}, err
	}
	if abortErr != nil {
		return DryRunResult{}, abortErr
	}
	return res, nil
}

func eosPreview(s *cliShell, mode PushMode, candidate string) (DryRunResult, error) {
	if mode == PushReplace {
		if err := s.eosCommand("rollback clean-config"); err != nil {
			return DryRunResult{}, err
		}
	}
	// The candidate is sent whole and read back once, at a comment line Seed
	// adds after it: a round trip per line would not finish a chassis
	// configuration inside the device timeout.
	var (
		b    strings.Builder
		sent []int
	)
	for i, line := range strings.Split(strings.ReplaceAll(candidate, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "!") {
			continue
		}
		b.WriteString(strings.TrimRight(line, " \t") + "\n")
		sent = append(sent, i+1)
	}
	marker, err := eosSessionName()
	if err != nil {
		return DryRunResult{}, err
	}
	marker = "! " + marker + "\n"
	all, i, err := s.sendThrough(b.String()+marker, marker)
	if err != nil {
		return DryRunResult{}, err
	}
	if msgs := s.eosLoadErrors(all[:i], sent); len(msgs) > 0 {
		return DryRunResult{}, fmt.Errorf("%w: the device rejected the candidate%s", ErrCommand, joinMessages(msgs))
	}
	if !strings.Contains(s.last, "(config-s-") {
		return DryRunResult{}, fmt.Errorf("%w: the candidate left the configuration session", ErrCommand)
	}
	diff, err := s.run("show session-config diffs | no-more")
	if err != nil {
		return DryRunResult{}, err
	}
	if msgs := eosErrors(diff); len(msgs) > 0 {
		return DryRunResult{}, fmt.Errorf("%w: show session-config diffs%s", ErrCommand, joinMessages(msgs))
	}
	return deviceDiff(diff), nil
}

// eosEnable raises a login that lands in unprivileged exec mode. A push
// credential with an enable password is refused: it should log in at
// privilege 15.
func (s *cliShell) eosEnable() error {
	if err := s.send("enable\n"); err != nil {
		return err
	}
	either := regexp.MustCompile(s.prompt.String() + "|" + enablePassword.String())
	if _, err := s.expect(either); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(s.last), "#") {
		_ = s.send("\x03")
		return &PlanRefusedError{
			Reason: "the push credential needs privilege 15: the device asked for an enable password",
		}
	}
	return nil
}

func (s *cliShell) eosCommand(command string) error {
	out, err := s.run(command)
	if err != nil {
		return err
	}
	if msgs := eosErrors(out); len(msgs) > 0 {
		return fmt.Errorf("%w: %s%s", ErrCommand, command, joinMessages(msgs))
	}
	return nil
}

// eosErrors returns the "% " lines of a command's output, without any token
// of configuration text the device quoted back.
func eosErrors(out string) []string {
	var msgs []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "% ") {
			msg, _, _ := strings.Cut(line, " (at token")
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// eosLoadErrors names each rejected candidate line by its line number. Every
// line the device read is echoed after a prompt, so an error belongs to the
// line echoed last; sent maps the n-th line sent to its candidate line.
func (s *cliShell) eosLoadErrors(out string, sent []int) []string {
	// The prompt before the first line was read with the previous command.
	var (
		msgs []string
		n    int
	)
	for line := range strings.SplitSeq(out, "\n") {
		if s.prompt.MatchString(promptOf(line)) {
			n++
			continue
		}
		if msg := eosErrors(line); len(msg) > 0 && n < len(sent) {
			msgs = append(msgs, "line "+strconv.Itoa(sent[n])+": "+msg[0])
		}
	}
	return msgs
}

// promptOf returns the prompt an echoed line starts with, up to and
// including its first '#' or '>'.
func promptOf(line string) string {
	if i := strings.IndexAny(line, "#>"); i >= 0 {
		return line[:i+1]
	}
	return ""
}

func eosSessionName() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("name the configuration session: %w", err)
	}
	return "seed-plan-" + hex.EncodeToString(b[:]), nil
}

// deviceDiff counts a device's own diff: the "+" and "-" lines, without the
// "+++" and "---" headers of a unified diff.
func deviceDiff(diff string) DryRunResult {
	res := DryRunResult{Diff: diff}
	for line := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "):
		case strings.HasPrefix(line, "+"):
			res.Added++
		case strings.HasPrefix(line, "-"):
			res.Removed++
		}
	}
	return res
}

func joinMessages(msgs []string) string {
	if len(msgs) == 0 {
		return ""
	}
	if len(msgs) > maxDeviceMessages {
		msgs = append(
			msgs[:maxDeviceMessages:maxDeviceMessages],
			fmt.Sprintf("and %d more", len(msgs)-maxDeviceMessages),
		)
	}
	return ": " + strings.Join(msgs, "; ")
}
