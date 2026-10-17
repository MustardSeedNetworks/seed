package deviceconfig

// The Junos, EOS and IOS push dialogues (ADR-0035) drive the device's own CLI
// in an interactive shell. For a dry run the candidate is loaded into a
// private candidate (Junos) or a configuration session (EOS), where the device
// prints its own diff and the candidate is thrown away. Both are scoped to
// this login, so nothing another user is editing is touched, and a Junos
// private candidate dies with the session if the discard never arrives. A
// push commits under a timer the device reverts by itself on: Junos from the
// private candidate, EOS from a session named after the commit, and IOS by
// typing the candidate under a revert timer.

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
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

// iosModeControl reports whether an IOS candidate line starting with word
// would leave configuration mode or run an exec command. A final "end", as
// "show running-config" prints, is the exception: Seed ends the load itself.
func iosModeControl(word string) bool {
	return word == "do" || word == "end" || word == "exit"
}

// checkCLICandidate refuses a CLI candidate that could do more than load
// configuration. The device reads it as typed input, so a control character
// is a keystroke (^D ends a Junos load and ^Z leaves EOS and IOS
// configuration mode), and an EOS or IOS line is a command.
func checkCLICandidate(platform Platform, candidate string) error {
	lines := strings.Split(strings.ReplaceAll(candidate, "\r\n", "\n"), "\n")
	last := lastIOSLine(lines)
	for i, line := range lines {
		for _, r := range line {
			if unicode.IsControl(r) && r != '\t' {
				return fmt.Errorf("candidate line %d contains a control character", i+1)
			}
		}
		word, _, _ := strings.Cut(strings.TrimSpace(line), " ")
		var control bool
		switch platform {
		case PlatformAristaEOS:
			control = eosSessionControl(word)
		case PlatformCiscoIOS:
			control = iosModeControl(word) && (i != last || word != "end")
		case PlatformJuniperJunos, PlatformCiscoNXOS, PlatformNETCONF:
		}
		if control {
			return fmt.Errorf(
				"candidate line %d starts with %q, which would leave configuration mode",
				i+1,
				word,
			)
		}
	}
	return nil
}

// lastIOSLine returns the index of the last line that is neither blank nor a
// "!" separator, or -1.
func lastIOSLine(lines []string) int {
	for i, line := range slices.Backward(lines) {
		if t := strings.TrimSpace(line); t != "" && t != "!" {
			return i
		}
	}
	return -1
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

// junosDryRun loads the candidate into a private candidate, has the device
// compare it with the running configuration, and rolls it back.
func junosDryRun(session *ssh.Session, req DeviceRequest) (DryRunResult, error) {
	var res DryRunResult
	err := junosPrivate(session, func(s *cliShell) error {
		if err := s.junosLoad(req.Mode, req.Candidate); err != nil {
			return err
		}
		diff, err := s.run("show | compare | no-more")
		if err != nil {
			return err
		}
		if msgs := junosErrors(diff); len(msgs) > 0 {
			return fmt.Errorf("%w: show | compare%s", ErrCommand, joinMessages(msgs))
		}
		res = deviceDiff(trimEditBanner(diff))
		return nil
	})
	return res, err
}

// junosCommit loads the candidate and commits it with "commit confirmed",
// which Junos rolls back by itself unless a commit follows in time.
func junosCommit(session *ssh.Session, req DeviceRequest) error {
	return junosPrivate(session, func(s *cliShell) error {
		if err := s.junosLoad(req.Mode, req.Candidate); err != nil {
			return err
		}
		return s.junosCommitCommand("commit confirmed " + strconv.Itoa(int(ConfirmWindow.Minutes())))
	})
}

// junosConfirm confirms a pending "commit confirmed": any commit does.
func junosConfirm(session *ssh.Session, _ DeviceRequest) error {
	return junosPrivate(session, func(s *cliShell) error { return s.junosCommitCommand("commit") })
}

// junosRollback commits rollback 1, the configuration before the pending
// commit, which also ends its timer. That holds only while the pending commit
// is the newest: a commit someone else made on the device since then would be
// undone with it, so the commit history is checked first.
func junosRollback(session *ssh.Session, _ DeviceRequest) error {
	return junosPrivate(session, func(s *cliShell) error {
		history, err := s.run("run show system commit | no-more")
		if err != nil {
			return err
		}
		if !junosNewestCommitPending(history) {
			return fmt.Errorf("%w: the newest commit on the device is no longer the pending one", ErrCommand)
		}
		if err = s.junosCommand("rollback 1"); err != nil {
			return err
		}
		return s.junosCommitCommand("commit")
	})
}

// junosNewestCommitPending reports whether entry 0 of "show system commit" is
// a confirmed commit still waiting: "0   2026-10-17 05:01:02 UTC by pusher
// via cli commit confirmed, rollback in 5mins".
func junosNewestCommitPending(history string) bool {
	for line := range strings.SplitSeq(history, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == "0" {
			return strings.Contains(line, "commit confirmed, rollback in")
		}
	}
	return false
}

// junosPrivate runs fn in a private candidate and leaves it empty. Leaving
// with changes would stop at a yes/no question; a failed rollback leaves them
// to be discarded with the session instead.
func junosPrivate(session *ssh.Session, fn func(*cliShell) error) error {
	s, err := openCLIShell(session)
	if err != nil {
		return err
	}
	out, err := s.run("configure private")
	if err != nil {
		return err
	}
	// Junos refuses a private candidate while the shared one has uncommitted
	// changes, so this is also what keeps a push off someone else's work.
	if msgs := junosErrors(out); len(msgs) > 0 {
		return &PlanRefusedError{Reason: "the device refused a private candidate: " + strings.Join(msgs, "; ")}
	}
	err = fn(s)
	cleanupErr := s.junosCommand("rollback 0")
	if cleanupErr == nil {
		cleanupErr = s.junosCommand("exit configuration-mode")
	}
	_ = s.send("exit\n")
	if err != nil {
		return err
	}
	return cleanupErr
}

// junosLoad loads the candidate. Replace is Junos's "load override": the
// candidate becomes the whole configuration.
func (s *cliShell) junosLoad(mode PushMode, candidate string) error {
	verb := "merge"
	if mode == PushReplace {
		verb = "override"
	}
	if err := s.send("load " + verb + " terminal\n"); err != nil {
		return err
	}
	if _, err := s.expect(junosLoadReady); err != nil {
		return err
	}
	text := strings.ReplaceAll(candidate, "\r\n", "\n")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	// ^D on a line of its own ends the load.
	const done = "load complete"
	all, i, err := s.sendThrough(text+"\x04", done)
	if err != nil {
		return err
	}
	// "load complete (2 errors)": the rejected lines were reported above it.
	if summary, _, _ := strings.Cut(all[i+len(done):], "\n"); strings.TrimSpace(summary) != "" {
		return fmt.Errorf("%w: the device rejected the candidate (%s%s)%s",
			ErrCommand, done, summary, joinMessages(junosLoadErrors(all[:i])))
	}
	return nil
}

// junosCommitCommand runs a commit and requires Junos to report it complete:
// a commit check failure prints its errors and no "commit complete".
func (s *cliShell) junosCommitCommand(command string) error {
	out, err := s.run(command)
	if err != nil {
		return err
	}
	if msgs := junosErrors(out); len(msgs) > 0 || !strings.Contains(out, "commit complete") {
		return fmt.Errorf("%w: %s did not complete%s", ErrCommand, command, joinMessages(msgs))
	}
	return nil
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
func eosDryRun(session *ssh.Session, req DeviceRequest) (DryRunResult, error) {
	s, err := openPrivileged(session)
	if err != nil {
		return DryRunResult{}, err
	}
	name, err := eosSessionName()
	if err != nil {
		return DryRunResult{}, err
	}
	var res DryRunResult
	err = s.eosSession(name, func() error {
		if loadErr := s.eosLoad(req.Mode, req.Candidate); loadErr != nil {
			return loadErr
		}
		diff, runErr := s.run("show session-config diffs | no-more")
		if runErr != nil {
			return runErr
		}
		if msgs := cliErrors(diff); len(msgs) > 0 {
			return fmt.Errorf("%w: show session-config diffs%s", ErrCommand, joinMessages(msgs))
		}
		res = deviceDiff(diff)
		return nil
	})
	return res, err
}

// eosSession runs fn in a new configuration session and aborts the session
// afterwards. A configuration session outlives the login on EOS, so it is
// aborted explicitly even after a failure.
func (s *cliShell) eosSession(name string, fn func() error) error {
	if err := s.cliCommand("configure session " + name); err != nil {
		return err
	}
	err := fn()
	abortErr := s.cliCommand("abort")
	_ = s.send("exit\n")
	return errors.Join(err, abortErr)
}

// eosCommit loads the candidate into a session named after the commit and
// commits it under a timer: EOS reverts the session unless it is committed
// again, by name, before the timer runs out. A candidate the device rejects
// is aborted with its session and never committed.
func eosCommit(session *ssh.Session, req DeviceRequest) error {
	s, err := openPrivileged(session)
	if err != nil {
		return err
	}
	if err = s.cliCommand("configure session " + req.CommitID); err != nil {
		return err
	}
	if err = s.eosLoad(req.Mode, req.Candidate); err != nil {
		return errors.Join(err, s.cliCommand("abort"))
	}
	err = s.cliCommand("commit timer " + time.Time{}.Add(ConfirmWindow).Format(time.TimeOnly))
	_ = s.send("exit\n")
	return err
}

// eosConfirm commits the pending session again, which ends its timer, and
// saves the startup configuration.
func eosConfirm(session *ssh.Session, req DeviceRequest) error {
	s, err := openPrivileged(session)
	if err != nil {
		return err
	}
	if err = s.cliCommand("configure session " + req.CommitID + " commit"); err != nil {
		return err
	}
	return s.saveStartup()
}

// eosRollback aborts the pending session, which reverts it now. The session
// is named after this push's commit, so nothing anyone else committed is
// undone with it.
func eosRollback(session *ssh.Session, req DeviceRequest) error {
	s, err := openPrivileged(session)
	if err != nil {
		return err
	}
	err = s.cliCommand("configure session " + req.CommitID + " abort")
	_ = s.send("exit\n")
	return err
}

// eosLoad types the candidate into the open session. Replace starts the
// session from a clean configuration first.
func (s *cliShell) eosLoad(mode PushMode, candidate string) error {
	if mode == PushReplace {
		if err := s.cliCommand("rollback clean-config"); err != nil {
			return err
		}
	}
	return s.typeCandidate(candidate, "(config-s-")
}

// iosDryRun predicts the candidate against the configuration the device runs
// now. IOS has no preview; after an apply this is the post-check, which a
// device holding the change passes with nothing left to change.
func iosDryRun(session *ssh.Session, req DeviceRequest) (DryRunResult, error) {
	running, err := runCommand(session, "show running-config")
	if err != nil {
		return DryRunResult{}, err
	}
	return predictIOS(Backup{Config: running}, req.Mode, req.Candidate)
}

// iosCommit applies the candidate under a revert timer: IOS restores the
// configuration it archived when the timer started unless "configure
// confirm" arrives first. A merge is typed; a replace is copied to flash and
// read from there.
func iosCommit(client *ssh.Client, req DeviceRequest) error {
	if req.Mode == PushReplace {
		return iosReplace(client, req)
	}
	return onSession(client, func(session *ssh.Session) error { return iosMerge(session, req) })
}

// iosMerge types the candidate under the revert timer. A candidate the device
// rejects has been entered up to the rejected line, so it is reverted at once
// rather than at the timer.
func iosMerge(session *ssh.Session, req DeviceRequest) error {
	s, err := openPrivileged(session)
	if err != nil {
		return err
	}
	timer := "configure terminal revert timer " + strconv.Itoa(int(ConfirmWindow.Minutes()))
	if err = s.cliCommand(timer); err != nil {
		return err
	}
	loadErr := s.typeCandidate(req.Candidate, "(config")
	if loadErr == nil {
		err = s.cliCommand("end")
	} else {
		// A rejected line can have left configuration mode already, so the
		// end is allowed to fail.
		_ = s.cliCommand("end")
		err = s.cliCommand("configure revert now")
	}
	_ = s.send("exit\n")
	return errors.Join(loadErr, err)
}

// iosReplace copies the candidate to flash under the commit's name and
// replaces the running configuration with it under the revert timer. The
// copy carries the whole configuration, secrets included, so it is deleted
// again whatever the replace did: the revert restores from the archive, not
// from the copy, and a copy that cannot be deleted fails the commit, which the
// timer then reverts. A replace that fails is reverted at once if it started
// the timer.
func iosReplace(client *ssh.Client, req DeviceRequest) error {
	file := "flash:" + req.CommitID + ".cfg"
	text := strings.ReplaceAll(req.Candidate, "\r\n", "\n")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	copyErr := scpSink(client, file, int64(len(text)), strings.NewReader(text))
	return onSession(client, func(session *ssh.Session) error {
		s, err := openPrivileged(session)
		if err != nil {
			return errors.Join(copyErr, err)
		}
		var replaceErr, deleteErr error
		if copyErr == nil {
			replaceErr = s.iosConfigureReplace(file)
			deleteErr = s.cliCommand("delete /force " + file)
		} else {
			// A copy that failed part way can have left part of the file.
			_ = s.cliCommand("delete /force " + file)
		}
		_ = s.send("exit\n")
		return errors.Join(copyErr, replaceErr, deleteErr)
	})
}

// iosConfigureReplace runs "configure replace" with "force", which skips its
// yes/no question, and requires IOS to report the replace done.
func (s *cliShell) iosConfigureReplace(file string) error {
	command := "configure replace " + file + " force time " + strconv.Itoa(int(ConfirmWindow.Minutes()))
	out, err := s.run(command)
	if err != nil {
		return err
	}
	if msgs := cliErrors(out); len(msgs) > 0 || !strings.Contains(out, "Rollback Done") {
		return errors.Join(
			fmt.Errorf("%w: configure replace did not complete%s", ErrCommand, joinMessages(msgs)),
			s.iosRevertPending(),
		)
	}
	return nil
}

// iosConfirm confirms the pending change, which ends its revert timer, and
// saves the startup configuration.
func iosConfirm(session *ssh.Session, _ DeviceRequest) error {
	s, err := openPrivileged(session)
	if err != nil {
		return err
	}
	if err = s.cliCommand("configure confirm"); err != nil {
		return err
	}
	return s.saveStartup()
}

// iosRollback reverts the pending change now. Without a pending timer there
// is nothing of Seed's to revert, and "configure revert now" is not sent.
func iosRollback(session *ssh.Session, _ DeviceRequest) error {
	s, err := openPrivileged(session)
	if err != nil {
		return err
	}
	err = s.iosRevertPending()
	_ = s.send("exit\n")
	return err
}

// iosRevertPending reverts the change pending under a revert timer now. With
// no timer pending there is nothing of Seed's to revert, and "configure
// revert now" is not sent.
func (s *cliShell) iosRevertPending() error {
	if err := s.cliCommand("show archive config rollback timer"); err != nil {
		return fmt.Errorf("%w: no revert timer is pending on the device", ErrCommand)
	}
	return s.cliCommand("configure revert now")
}

// saveStartup writes the running configuration to the startup
// configuration, so a confirmed push survives a reload.
func (s *cliShell) saveStartup() error {
	err := s.cliCommand("write memory")
	_ = s.send("exit\n")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStartupNotSaved, err)
	}
	return nil
}

// typeCandidate types the candidate in configuration mode. It is sent whole
// and read back once, at a comment line Seed adds after it: a round trip per
// line would not finish a chassis configuration inside the device timeout.
// Blank and "!" lines are not sent, and neither is a final "end". The device
// must still be in configuration mode, which mode names, afterwards.
func (s *cliShell) typeCandidate(candidate, mode string) error {
	var (
		b    strings.Builder
		sent []int
	)
	lines := strings.Split(strings.ReplaceAll(candidate, "\r\n", "\n"), "\n")
	last := lastIOSLine(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "!") || (i == last && trimmed == "end") {
			continue
		}
		b.WriteString(strings.TrimRight(line, " \t") + "\n")
		sent = append(sent, i+1)
	}
	marker, err := eosSessionName()
	if err != nil {
		return err
	}
	marker = "! " + marker + "\n"
	all, i, err := s.sendThrough(b.String()+marker, marker)
	if err != nil {
		return err
	}
	if msgs := s.loadErrors(all[:i], sent); len(msgs) > 0 {
		return fmt.Errorf("%w: the device rejected the candidate%s", ErrCommand, joinMessages(msgs))
	}
	if !strings.Contains(s.last, mode) {
		return fmt.Errorf("%w: the candidate left configuration mode", ErrCommand)
	}
	return nil
}

// openPrivileged opens the CLI at privilege 15, raising a login that lands in
// unprivileged exec mode. A push credential with an enable password is
// refused: it should log in at privilege 15.
func openPrivileged(session *ssh.Session) (*cliShell, error) {
	s, err := openCLIShell(session)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(strings.TrimSpace(s.last), ">") {
		return s, nil
	}
	if err = s.send("enable\n"); err != nil {
		return nil, err
	}
	either := regexp.MustCompile(s.prompt.String() + "|" + enablePassword.String())
	if _, err = s.expect(either); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(strings.TrimSpace(s.last), "#") {
		_ = s.send("\x03")
		return nil, &PlanRefusedError{
			Reason: "the push credential needs privilege 15: the device asked for an enable password",
		}
	}
	return s, nil
}

// cliCommand runs an EOS or IOS command and reports the device's error if it
// printed one.
func (s *cliShell) cliCommand(command string) error {
	out, err := s.run(command)
	if err != nil {
		return err
	}
	if msgs := cliErrors(out); len(msgs) > 0 {
		return fmt.Errorf("%w: %s%s", ErrCommand, command, joinMessages(msgs))
	}
	return nil
}

// cliErrors returns the "%" lines of an EOS or IOS command's output, without
// any token of configuration text the device quoted back.
func cliErrors(out string) []string {
	var msgs []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "%") {
			msg, _, _ := strings.Cut(line, " (at token")
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// loadErrors names each rejected candidate line by its line number. Every
// line the device read is echoed after a prompt, so an error belongs to the
// line echoed last; sent maps the n-th line sent to its candidate line.
func (s *cliShell) loadErrors(out string, sent []int) []string {
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
		if msg := cliErrors(line); len(msg) > 0 && n < len(sent) {
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
