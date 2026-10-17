package deviceconfig_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

// fakeCLI is the shared half of the scripted Junos and EOS shells: it echoes
// each line it reads, as a terminal does, and records the commands.
type fakeCLI struct {
	mu       sync.Mutex
	commands []string
	done     chan struct{}
}

func (f *fakeCLI) record(cmd string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, cmd)
}

func (f *fakeCLI) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.commands...)
}

// readLine reads one typed line, echoing it. A lone control character (^C)
// is returned as itself.
func readLine(r *bufio.Reader, ch io.Writer) (string, error) {
	var b strings.Builder
	for {
		c, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		switch {
		case c == '\n':
			_, _ = io.WriteString(ch, b.String()+"\r\n")
			return b.String(), nil
		case c == 0x03 && b.Len() == 0:
			return "\x03", nil
		default:
			b.WriteByte(c)
		}
	}
}

// fakeJunos is a Junos CLI with a private candidate: load fills it, show |
// compare prints compare whenever it holds something, rollback 0 empties it.
type fakeJunos struct {
	fakeCLI

	compare string
	// sharedModified refuses configure private, as Junos does while another
	// user has uncommitted changes.
	sharedModified bool

	loaded string
	// commitFails makes every commit fail its check. committed is every
	// commit made, newest last; pending is a commit confirmed not yet
	// confirmed.
	commitFails bool
	committed   []string
	pending     bool
	// history is "show system commit" entry 0.
	history string
}

func (j *fakeJunos) serve(ch ssh.Channel) {
	defer close(j.done)
	r := bufio.NewReader(ch)
	config := false
	prompt := func() string {
		if config {
			return "\r\n[edit]\r\nadmin@r1# "
		}
		return "admin@r1> "
	}
	_, _ = io.WriteString(ch, "--- JUNOS 23.4R1 Kernel 64-bit\r\n"+prompt())
	for {
		line, err := readLine(r, ch)
		if err != nil {
			return
		}
		j.record(line)
		var out string
		switch line {
		case "configure private":
			if j.sharedModified {
				out = "error: shared configuration database modified\r\n" +
					"Users currently editing the configuration:\r\n  bob terminal pts/1 (2026-10-17) [edit]\r\n"
				break
			}
			out, config = "warning: uncommitted changes will be discarded on exit\r\nEntering configuration mode\r\n", true
		case "load merge terminal", "load override terminal":
			_, _ = io.WriteString(ch, "[Type ^D at a new line to end input]\r\n")
			text, readErr := r.ReadString(0x04)
			if readErr != nil {
				return
			}
			text = strings.TrimSuffix(text, "\x04")
			_, _ = io.WriteString(ch, strings.ReplaceAll(text, "\n", "\r\n"))
			j.mu.Lock()
			j.loaded = text
			j.mu.Unlock()
			out = "load complete\r\n"
			if strings.Contains(text, "bogus") {
				out = "terminal:2:(4) syntax error: bogus\r\nload complete (1 errors)\r\n"
			}
		case "show | compare | no-more":
			if j.candidate() != "" {
				out = j.compare
			}
		case "rollback 0":
			j.mu.Lock()
			j.loaded = ""
			j.mu.Unlock()
			out = "load complete\r\n"
		case "rollback 1":
			j.mu.Lock()
			j.loaded = "rollback 1"
			j.mu.Unlock()
			out = "load complete\r\n"
		case "commit confirmed 5", "commit":
			out = j.commit(line)
		case "run show system commit | no-more":
			out = "0   2026-10-17 05:01:02 UTC by pusher via cli " + j.history + "\r\n" +
				"1   2026-10-16 09:00:00 UTC by admin via cli\r\n"
		case "exit configuration-mode":
			out, config = "Exiting configuration mode\r\n", false
		case "exit":
			return
		default:
			out = "syntax error, expecting <command>.\r\n"
		}
		_, _ = io.WriteString(ch, out+prompt())
	}
}

func (j *fakeJunos) commit(command string) string {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.commitFails {
		return "error: configuration check-out failed\r\nerror: commit failed: (statements constraint check failed)\r\n"
	}
	j.committed = append(j.committed, j.loaded)
	j.loaded = ""
	j.pending = command != "commit"
	if j.pending {
		return "commit confirmed will be automatically rolled back in 5 minutes unless confirmed\r\n" +
			"commit complete\r\n\r\n# commit confirmed will be rolled back in 5 minutes"
	}
	return "commit complete"
}

func (j *fakeJunos) state() ([]string, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.committed...), j.pending
}

func (j *fakeJunos) candidate() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.loaded
}

// fakeEOS is an EOS CLI with configuration sessions: lines typed in a session
// are kept until abort, and show session-config diffs prints diff whenever
// the session holds any.
type fakeEOS struct {
	fakeCLI

	diff string
	// unprivileged logs in at ">"; enablePassword then asks for a password.
	unprivileged   bool
	enablePassword bool

	session string
	lines   []string
	aborted bool

	// pending is the session committed under a timer, not yet confirmed;
	// committed is what it holds. confirmed and reverted say how it ended.
	pending   string
	committed []string
	confirmed bool
	reverted  bool
	// saveFails fails write memory.
	saveFails bool
}

func (e *fakeEOS) serve(channel ssh.Channel) {
	defer close(e.done)
	r := bufio.NewReader(channel)
	// Output is flushed once the typed-ahead input is used up, as a device
	// writes in packets rather than one per line.
	ch := bufio.NewWriter(channel)
	defer func() { _ = ch.Flush() }()
	flush := func() {
		if r.Buffered() == 0 {
			_ = ch.Flush()
		}
	}
	mode := "#"
	if e.unprivileged {
		mode = ">"
	}
	_, _ = io.WriteString(ch, "Last login: Fri Oct 16 2026\r\nr1"+mode)
	_ = ch.Flush()
	for {
		line, err := readLine(r, ch)
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(line)
		if !strings.HasPrefix(cmd, "!") {
			e.record(cmd)
		}
		var out string
		switch {
		case cmd == "enable" && e.enablePassword:
			_, _ = io.WriteString(ch, "Password: ")
			_ = ch.Flush()
			if line, err = readLine(r, ch); err != nil || line != "\x03" {
				return
			}
			out = "\r\n"
		case cmd == "enable":
			mode = "#"
		case strings.HasPrefix(cmd, "configure session ") && strings.Count(cmd, " ") == 3:
			out = e.endPending(cmd)
		case cmd == "write memory":
			out = "Copy completed successfully.\r\n"
			if e.saveFails {
				out = "% Error copying system:/running-config to flash:/startup-config (No space left on device)\r\n"
			}
		case strings.HasPrefix(cmd, "configure session "):
			e.mu.Lock()
			e.session = strings.TrimPrefix(cmd, "configure session ")
			e.mu.Unlock()
			mode = "(config-s-" + e.session[:12] + ")#"
		case cmd == "exit":
			return
		case strings.HasPrefix(mode, "(config-s-"):
			out, mode = e.inSession(cmd, mode)
		default:
			out = "% Invalid input\r\n"
		}
		_, _ = io.WriteString(ch, out+"r1"+mode)
		flush()
	}
}

// inSession answers a line typed in a configuration session.
func (e *fakeEOS) inSession(cmd, mode string) (string, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch {
	case cmd == "abort":
		e.aborted, e.lines = true, nil
		return "", "#"
	case strings.HasPrefix(cmd, "commit timer "):
		e.pending, e.committed, e.lines = e.session, e.lines, nil
		return "", "#"
	case cmd == "show session-config diffs | no-more":
		if len(e.lines) > 0 {
			return e.diff, mode
		}
	case strings.HasPrefix(cmd, "!"), cmd == "rollback clean-config":
	case strings.HasPrefix(cmd, "bogus"):
		return "% Invalid input (at token 0: 'bogus')\r\n", mode
	default:
		e.lines = append(e.lines, cmd)
		if strings.HasPrefix(cmd, "interface ") {
			return "", "(config-s-" + e.session[:12] + "-if-Et1)#"
		}
	}
	return "", mode
}

// endPending answers "configure session <name> commit|abort" in exec mode.
func (e *fakeEOS) endPending(cmd string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	fields := strings.Fields(cmd)
	if e.pending == "" || fields[2] != e.pending {
		return "% No pending commit timer for session " + fields[2] + "\r\n"
	}
	e.pending = ""
	if fields[3] == "commit" {
		e.confirmed = true
	} else {
		e.reverted = true
	}
	return ""
}

func (e *fakeEOS) outcome() (string, []string, bool, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.pending, append([]string(nil), e.committed...), e.confirmed, e.reverted
}

func (e *fakeEOS) held() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.lines...)
}

func startShell(t *testing.T, serve func(ssh.Channel), platform deviceconfig.Platform) deviceconfig.DeviceRequest {
	t.Helper()
	dev := startSSHDevice(t, deviceOpts{password: "wr1te", shell: serve})
	fr := dev.request(t, "wr1te", "", dev.fingerprint)
	return deviceconfig.DeviceRequest{
		Host: fr.Host, Port: fr.Port, User: "pusher", Password: "wr1te", PinnedHostKey: dev.fingerprint,
		Platform: platform, Mode: deviceconfig.PushMerge,
	}
}

const junosCompare = "[edit system]\r\n-  host-name edge;\r\n+  host-name core;\r\n" +
	"[edit interfaces ge-0/0/0]\r\n+   description uplink;\r\n"

// TestJunosDryRunLeavesTheDeviceAsItWas: the candidate is loaded into a
// private candidate, compared by the device, and rolled back before the CLI
// leaves configuration mode.
func TestJunosDryRunLeavesTheDeviceAsItWas(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode deviceconfig.PushMode
		load string
	}{
		{deviceconfig.PushMerge, "load merge terminal"},
		{deviceconfig.PushReplace, "load override terminal"},
	} {
		t.Run(string(tc.mode), func(t *testing.T) {
			t.Parallel()
			j := &fakeJunos{done: make(chan struct{}), compare: junosCompare}
			req := startShell(t, j.serve, deviceconfig.PlatformJuniperJunos)
			req.Mode = tc.mode
			req.Candidate = "system {\n    host-name core;\n}"

			res, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
			require.NoError(t, err)
			<-j.done
			require.Equal(t, []string{
				"configure private",
				tc.load,
				"show | compare | no-more",
				"rollback 0",
				"exit configuration-mode",
				"exit",
			}, j.recorded())
			require.Empty(t, j.candidate(), "the private candidate is rolled back")
			require.Equal(t, 2, res.Added)
			require.Equal(t, 1, res.Removed)
			require.Equal(t, strings.ReplaceAll(junosCompare, "\r", ""), res.Diff)
		})
	}
}

func TestJunosDryRunWithNoChange(t *testing.T) {
	t.Parallel()
	j := &fakeJunos{done: make(chan struct{})}
	req := startShell(t, j.serve, deviceconfig.PlatformJuniperJunos)
	req.Candidate = "system {\n    host-name edge;\n}\n"

	res, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.DryRunResult{}, res)
}

func TestJunosDryRunFailures(t *testing.T) {
	t.Parallel()
	t.Run("someone else is editing", func(t *testing.T) {
		t.Parallel()
		j := &fakeJunos{done: make(chan struct{}), sharedModified: true}
		req := startShell(t, j.serve, deviceconfig.PlatformJuniperJunos)
		req.Candidate = "system {}"

		_, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
		refused, ok := errors.AsType[*deviceconfig.PlanRefusedError](err)
		require.True(t, ok, "%v", err)
		require.Contains(t, refused.Reason, "shared configuration database modified")
		require.NotContains(t, refused.Reason, "bob", "who is editing stays on the device")
		require.Equal(t, []string{"configure private"}, j.recorded())
	})
	t.Run("the device rejects the candidate, which is still rolled back", func(t *testing.T) {
		t.Parallel()
		j := &fakeJunos{done: make(chan struct{}), compare: junosCompare}
		req := startShell(t, j.serve, deviceconfig.PlatformJuniperJunos)
		req.Candidate = "system {\n    bogus secret-token;\n}\n"

		_, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "load complete (1 errors)")
		require.ErrorContains(t, err, "line 2: syntax error")
		require.NotContains(t, err.Error(), "bogus", "configuration text stays out of the error")
		<-j.done
		require.Equal(t, []string{
			"configure private", "load merge terminal", "rollback 0", "exit configuration-mode", "exit",
		}, j.recorded())
		require.Empty(t, j.candidate())
	})
}

// TestJunosCommitConfirmAndRollback: a commit is a commit confirmed with the
// device's own revert, from a private candidate; confirming and rolling back
// are commits from new logins, and a rollback first checks the pending commit
// is still the newest.
func TestJunosCommitConfirmAndRollback(t *testing.T) {
	t.Parallel()
	const candidate = "system {\n    host-name core;\n}"
	pending := "commit confirmed, rollback in 5mins"
	tests := []struct {
		name      string
		step      func(deviceconfig.SSHFetcher, context.Context, deviceconfig.DeviceRequest) error
		history   string
		commands  []string
		committed []string
		pending   bool
		err       string
	}{
		{
			name:      "commit",
			step:      deviceconfig.SSHFetcher.Commit,
			commands:  []string{"configure private", "load merge terminal", "commit confirmed 5"},
			committed: []string{candidate + "\n"},
			pending:   true,
		},
		{
			name:      "confirm",
			step:      deviceconfig.SSHFetcher.Confirm,
			commands:  []string{"configure private", "commit"},
			committed: []string{""},
		},
		{
			name:      "rollback",
			step:      deviceconfig.SSHFetcher.Rollback,
			history:   pending,
			commands:  []string{"configure private", "run show system commit | no-more", "rollback 1", "commit"},
			committed: []string{"rollback 1"},
		},
		{
			name:     "rollback after someone else committed",
			step:     deviceconfig.SSHFetcher.Rollback,
			history:  "commit",
			commands: []string{"configure private", "run show system commit | no-more"},
			err:      "no longer the pending one",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			j := &fakeJunos{done: make(chan struct{}), history: tc.history}
			req := startShell(t, j.serve, deviceconfig.PlatformJuniperJunos)
			req.Candidate = candidate

			err := tc.step(deviceconfig.SSHFetcher{}, fetchCtx(t), req)
			if tc.err != "" {
				require.ErrorIs(t, err, deviceconfig.ErrCommand)
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			<-j.done
			require.Equal(t, append(tc.commands, "rollback 0", "exit configuration-mode", "exit"), j.recorded())
			committed, isPending := j.state()
			require.Equal(t, tc.committed, committed)
			require.Equal(t, tc.pending, isPending)
		})
	}
}

func TestJunosCommitThatFailsItsCheck(t *testing.T) {
	t.Parallel()
	j := &fakeJunos{done: make(chan struct{}), commitFails: true}
	req := startShell(t, j.serve, deviceconfig.PlatformJuniperJunos)
	req.Candidate = "system {\n    host-name core;\n}"

	err := deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req)
	require.ErrorIs(t, err, deviceconfig.ErrCommand)
	require.ErrorContains(t, err, "commit confirmed 5 did not complete: error: configuration check-out failed")
	<-j.done
	require.Empty(t, j.candidate(), "the private candidate is rolled back")
	committed, _ := j.state()
	require.Empty(t, committed)
}

const eosDiff = "--- system:/running-config\r\n+++ session:/seed-plan-session-config\r\n" +
	"@@ -1,3 +1,4 @@\r\n-hostname edge\r\n+hostname core\r\n interface Ethernet1\r\n+   description uplink\r\n"

// TestEOSDryRunLeavesTheDeviceAsItWas: the candidate is typed into a fresh
// configuration session, the device prints the session's diff, and the
// session is aborted.
func TestEOSDryRunLeavesTheDeviceAsItWas(t *testing.T) {
	t.Parallel()
	candidate := "hostname core\n!\ninterface Ethernet1\n   description uplink\n\n"
	for _, tc := range []struct {
		name         string
		mode         deviceconfig.PushMode
		unprivileged bool
		before       []string
		load         []string
	}{
		{name: "merge", mode: deviceconfig.PushMerge},
		{
			name: "replace starts from a clean configuration", mode: deviceconfig.PushReplace,
			load: []string{"rollback clean-config"},
		},
		{name: "unprivileged login", mode: deviceconfig.PushMerge, unprivileged: true, before: []string{"enable"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := &fakeEOS{done: make(chan struct{}), diff: eosDiff, unprivileged: tc.unprivileged}
			req := startShell(t, e.serve, deviceconfig.PlatformAristaEOS)
			req.Mode = tc.mode
			req.Candidate = candidate

			res, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
			require.NoError(t, err)
			<-e.done
			want := append(append([]string{}, tc.before...), "configure session "+e.session)
			want = append(append(want, tc.load...),
				"hostname core", "interface Ethernet1", "description uplink",
				"show session-config diffs | no-more", "abort", "exit")
			require.Equal(t, want, e.recorded())
			require.Regexp(t, `^seed-plan-[0-9a-f]{12}$`, e.session)
			require.True(t, e.aborted, "the session is aborted")
			require.Empty(t, e.held())
			require.Equal(t, 2, res.Added)
			require.Equal(t, 1, res.Removed)
			require.Equal(t, strings.ReplaceAll(eosDiff, "\r", ""), res.Diff)
		})
	}
}

func TestEOSDryRunFailures(t *testing.T) {
	t.Parallel()
	t.Run("an enable password is refused", func(t *testing.T) {
		t.Parallel()
		e := &fakeEOS{done: make(chan struct{}), unprivileged: true, enablePassword: true}
		req := startShell(t, e.serve, deviceconfig.PlatformAristaEOS)
		req.Candidate = "hostname core\n"

		_, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
		refused, ok := errors.AsType[*deviceconfig.PlanRefusedError](err)
		require.True(t, ok, "%v", err)
		require.Contains(t, refused.Reason, "privilege 15")
		require.Equal(t, []string{"enable"}, e.recorded(), "no session is opened")
	})
	t.Run("the device rejects a line, and the session is still aborted", func(t *testing.T) {
		t.Parallel()
		e := &fakeEOS{done: make(chan struct{}), diff: eosDiff}
		req := startShell(t, e.serve, deviceconfig.PlatformAristaEOS)
		req.Candidate = "hostname core\n!\ninterface Ethernet1\n   bogus secret-token\n"

		_, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "line 4: % Invalid input")
		require.NotContains(t, err.Error(), "bogus", "configuration text stays out of the error")
		<-e.done
		require.True(t, e.aborted)
		require.Equal(t, "abort", e.recorded()[len(e.recorded())-2])
	})
}

// TestEOSDryRunOfALargeCandidate: a chassis-sized candidate ends at a prompt
// once per line, and is read back well inside the device timeout.
func TestEOSDryRunOfALargeCandidate(t *testing.T) {
	t.Parallel()
	e := &fakeEOS{done: make(chan struct{}), diff: eosDiff}
	req := startShell(t, e.serve, deviceconfig.PlatformAristaEOS)
	var b strings.Builder
	for i := 0; b.Len() < deviceconfig.MaxCandidateBytes-64; i++ {
		fmt.Fprintf(&b, "ip host h%06d 192.0.2.%d\n", i, i%250)
	}
	req.Candidate = b.String()

	res, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
	require.NoError(t, err)
	require.Equal(t, 2, res.Added)
	<-e.done
	require.True(t, e.aborted)
}

// TestEOSCommitConfirmAndRollback: a commit is a session named after the
// commit, committed under a timer; confirming commits that session again and
// saves the startup configuration, and a rollback aborts it, so nothing else
// committed on the device is undone.
func TestEOSCommitConfirmAndRollback(t *testing.T) {
	t.Parallel()
	const commitID = "seed-0123456789abcdef0123456789abcdef"
	held := []string{"hostname core", "interface Ethernet1", "description uplink"}
	tests := []struct {
		name      string
		step      func(deviceconfig.SSHFetcher, context.Context, deviceconfig.DeviceRequest) error
		pending   string
		commands  []string
		after     string
		committed []string
		confirmed bool
		reverted  bool
		saveFails bool
		err       error
	}{
		{
			name: "commit",
			step: deviceconfig.SSHFetcher.Commit,
			commands: append(append([]string{"configure session " + commitID}, held...),
				"commit timer 00:05:00", "exit"),
			after:     commitID,
			committed: held,
		},
		{
			name:      "confirm",
			step:      deviceconfig.SSHFetcher.Confirm,
			pending:   commitID,
			commands:  []string{"configure session " + commitID + " commit", "write memory", "exit"},
			confirmed: true,
		},
		{
			name:      "confirm without saving the startup configuration",
			step:      deviceconfig.SSHFetcher.Confirm,
			pending:   commitID,
			saveFails: true,
			commands:  []string{"configure session " + commitID + " commit", "write memory", "exit"},
			confirmed: true,
			err:       deviceconfig.ErrStartupNotSaved,
		},
		{
			name:     "rollback",
			step:     deviceconfig.SSHFetcher.Rollback,
			pending:  commitID,
			commands: []string{"configure session " + commitID + " abort", "exit"},
			reverted: true,
		},
		{
			name:     "rollback of a session that is no longer pending",
			step:     deviceconfig.SSHFetcher.Rollback,
			commands: []string{"configure session " + commitID + " abort", "exit"},
			err:      deviceconfig.ErrCommand,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := &fakeEOS{done: make(chan struct{}), pending: tc.pending, saveFails: tc.saveFails}
			req := startShell(t, e.serve, deviceconfig.PlatformAristaEOS)
			req.Candidate = "hostname core\n!\ninterface Ethernet1\n   description uplink\n"
			req.CommitID = commitID

			err := tc.step(deviceconfig.SSHFetcher{}, fetchCtx(t), req)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			<-e.done
			require.Equal(t, tc.commands, e.recorded())
			pending, committed, confirmed, reverted := e.outcome()
			require.Equal(t, tc.after, pending)
			require.Equal(t, tc.confirmed, confirmed)
			require.Equal(t, tc.reverted, reverted)
			require.Equal(t, tc.committed, committed)
		})
	}
}

// TestEOSCommitOfARejectedCandidate: the session is aborted, never committed.
func TestEOSCommitOfARejectedCandidate(t *testing.T) {
	t.Parallel()
	e := &fakeEOS{done: make(chan struct{})}
	req := startShell(t, e.serve, deviceconfig.PlatformAristaEOS)
	req.Candidate = "hostname core\nbogus secret-token\n"
	req.CommitID = "seed-0123456789abcdef0123456789abcdef"

	err := deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req)
	require.ErrorIs(t, err, deviceconfig.ErrCommand)
	require.ErrorContains(t, err, "line 2: % Invalid input")
	require.NotContains(t, err.Error(), "bogus")
	require.True(t, e.aborted)
	pending, committed, _, _ := e.outcome()
	require.Empty(t, pending)
	require.Empty(t, committed)
	require.NotContains(t, e.recorded(), "commit timer 00:05:00")
}

// fakeIOS is an IOS CLI with the archive's revert timer: configure terminal
// revert timer archives the running configuration, lines typed in
// configuration mode change it at once, and configure revert now restores
// the archive.
type fakeIOS struct {
	fakeCLI

	running []string
	archive []string
	pending bool
	saved   bool
	// noArchive refuses the revert timer, as IOS does without an archive path.
	noArchive bool
	// flash holds the files on the device by path, and copies every file
	// copied to it. scpRefused is the reason the SCP server refuses a copy,
	// as IOS does without "ip scp server enable".
	flash      map[string]string
	copies     []string
	scpRefused string
}

// startIOS serves d's CLI and its SCP server on one device.
func startIOS(t *testing.T, d *fakeIOS) deviceconfig.DeviceRequest {
	t.Helper()
	dev := startSSHDevice(t, deviceOpts{password: "wr1te", shell: d.serve, scp: d.scp})
	fr := dev.request(t, "wr1te", "", dev.fingerprint)
	return deviceconfig.DeviceRequest{
		Host: fr.Host, Port: fr.Port, User: "pusher", Password: "wr1te", PinnedHostKey: dev.fingerprint,
		Platform: deviceconfig.PlatformCiscoIOS, Mode: deviceconfig.PushReplace, CommitID: "seed-c1",
	}
}

// scp is the sink side of one copy.
func (d *fakeIOS) scp(ch ssh.Channel, path string) {
	if d.scpRefused != "" {
		_, _ = io.WriteString(ch, "\x01"+d.scpRefused+"\n")
		return
	}
	r := bufio.NewReader(ch)
	_, _ = ch.Write([]byte{0})
	header, err := r.ReadString('\n')
	if err != nil {
		return
	}
	var (
		perm, name string
		size       int
	)
	if _, err = fmt.Sscanf(header, "C%s %d %s", &perm, &size, &name); err != nil || perm != "0600" {
		_, _ = io.WriteString(ch, "\x01bad header\n")
		return
	}
	_, _ = ch.Write([]byte{0})
	body := make([]byte, size+1)
	if _, err = io.ReadFull(r, body); err != nil || body[size] != 0 {
		return
	}
	d.mu.Lock()
	d.flash[path] = string(body[:size])
	d.copies = append(d.copies, string(body[:size]))
	d.mu.Unlock()
	_, _ = ch.Write([]byte{0})
}

// replace is configure replace: the timer archives the running
// configuration, and the file's lines become it. A rejected line stops the
// replace with the lines before it applied and the timer running.
func (d *fakeIOS) replace(file string) string {
	text, ok := d.flash[file]
	switch {
	case !ok:
		return "%Error opening " + file + " (No such file or directory)\r\n"
	case d.noArchive:
		return "%Archive feature not enabled\r\n"
	}
	d.archive, d.pending, d.running = append([]string(nil), d.running...), true, nil
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "", line == "end", strings.HasPrefix(line, "!"):
		case strings.HasPrefix(line, "bogus"):
			return "%Rollback failed: a line was rejected\r\n"
		default:
			d.running = append(d.running, line)
		}
	}
	return "Total number of passes: 1\r\nRollback Done\r\n"
}

// fileCommand answers the commands that name a file on flash, and reports
// whether cmd was one.
func (d *fakeIOS) fileCommand(cmd string) (string, bool) {
	if file, ok := strings.CutPrefix(cmd, "configure replace "); ok {
		if file, ok = strings.CutSuffix(file, " force time 5"); ok {
			return d.replace(file), true
		}
	}
	file, ok := strings.CutPrefix(cmd, "delete /force ")
	if !ok {
		return "", false
	}
	if _, ok = d.flash[file]; !ok {
		return "%Error deleting " + file + " (No such file or directory)\r\n", true
	}
	delete(d.flash, file)
	return "", true
}

func (d *fakeIOS) files() map[string]string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return maps.Clone(d.flash)
}

func (d *fakeIOS) serve(channel ssh.Channel) {
	defer close(d.done)
	r := bufio.NewReader(channel)
	ch := bufio.NewWriter(channel)
	defer func() { _ = ch.Flush() }()
	mode := "#"
	_, _ = io.WriteString(ch, "\r\nr1"+mode)
	_ = ch.Flush()
	for {
		line, err := readLine(r, ch)
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(line)
		if !strings.HasPrefix(cmd, "!") {
			d.record(cmd)
		}
		var out string
		out, mode = d.answer(cmd, mode)
		if cmd == "exit" && mode == "#" {
			return
		}
		_, _ = io.WriteString(ch, out+"r1"+mode)
		if r.Buffered() == 0 {
			_ = ch.Flush()
		}
	}
}

func (d *fakeIOS) answer(cmd, mode string) (string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	const noTimer = "%No Rollback Confirmed Change pending\r\n"
	if mode != "#" {
		switch {
		case cmd == "end":
			return "", "#"
		case strings.HasPrefix(cmd, "!"):
		case strings.HasPrefix(cmd, "bogus"):
			return "                ^\r\n% Invalid input detected at '^' marker.\r\n\r\n", mode
		case strings.HasPrefix(cmd, "interface "):
			d.running = append(d.running, cmd)
			return "", "(config-if)#"
		default:
			d.running = append(d.running, cmd)
		}
		return "", mode
	}
	if out, ok := d.fileCommand(cmd); ok {
		return out, mode
	}
	switch cmd {
	case "configure terminal revert timer 5":
		if d.noArchive {
			return "%Archive feature not enabled\r\n", mode
		}
		d.archive, d.pending = append([]string(nil), d.running...), true
		return "Enter configuration commands, one per line.  End with CNTL/Z.\r\n", "(config)#"
	case "configure confirm":
		if !d.pending {
			return noTimer, mode
		}
		d.pending = false
	case "show archive config rollback timer":
		if !d.pending {
			return noTimer, mode
		}
		return "Timer type: Confirm\r\nTimer value: 5 min\r\n", mode
	case "configure revert now":
		if !d.pending {
			return noTimer, mode
		}
		d.running, d.pending = d.archive, false
		return "Rollback Done\r\n", mode
	case "write memory":
		d.saved = true
		return "Building configuration...\r\n[OK]\r\n", mode
	case "end", "exit":
		return "% Invalid input detected at '^' marker.\r\n", mode
	}
	return "", mode
}

func (d *fakeIOS) state() ([]string, bool, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.running...), d.pending, d.saved
}

// TestIOSCommitConfirmAndRollback: a commit types the candidate under the
// revert timer, a confirm ends the timer and saves, and a rollback reverts
// now only while a timer is pending.
func TestIOSCommitConfirmAndRollback(t *testing.T) {
	t.Parallel()
	candidate := "interface Gi1/0/1\n description uplink\n!\nend\n"
	tests := []struct {
		name     string
		step     func(deviceconfig.SSHFetcher, context.Context, deviceconfig.DeviceRequest) error
		pending  bool
		commands []string
		running  []string
		after    bool
		saved    bool
		err      string
	}{
		{
			name: "commit",
			step: deviceconfig.SSHFetcher.Commit,
			commands: []string{
				"configure terminal revert timer 5", "interface Gi1/0/1", "description uplink", "end", "exit",
			},
			running: []string{"hostname edge", "interface Gi1/0/1", "description uplink"},
			after:   true,
		},
		{
			name:     "confirm",
			step:     deviceconfig.SSHFetcher.Confirm,
			pending:  true,
			commands: []string{"configure confirm", "write memory", "exit"},
			running:  []string{"hostname edge"},
			saved:    true,
		},
		{
			name:     "rollback",
			step:     deviceconfig.SSHFetcher.Rollback,
			pending:  true,
			commands: []string{"show archive config rollback timer", "configure revert now", "exit"},
			running:  []string{"hostname edge"},
		},
		{
			name:     "rollback with no timer pending",
			step:     deviceconfig.SSHFetcher.Rollback,
			commands: []string{"show archive config rollback timer", "exit"},
			running:  []string{"hostname edge"},
			err:      "no revert timer is pending",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &fakeIOS{done: make(chan struct{}), running: []string{"hostname edge"}, pending: tc.pending}
			d.archive = d.running
			req := startShell(t, d.serve, deviceconfig.PlatformCiscoIOS)
			req.Candidate = candidate

			err := tc.step(deviceconfig.SSHFetcher{}, fetchCtx(t), req)
			if tc.err != "" {
				require.ErrorIs(t, err, deviceconfig.ErrCommand)
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			<-d.done
			require.Equal(t, tc.commands, d.recorded())
			running, pending, saved := d.state()
			require.Equal(t, tc.running, running)
			require.Equal(t, tc.after, pending)
			require.Equal(t, tc.saved, saved)
		})
	}
}

func TestIOSCommitFailures(t *testing.T) {
	t.Parallel()
	t.Run("a rejected line is reverted at once", func(t *testing.T) {
		t.Parallel()
		d := &fakeIOS{done: make(chan struct{}), running: []string{"hostname edge"}}
		req := startShell(t, d.serve, deviceconfig.PlatformCiscoIOS)
		req.Candidate = "hostname core\n bogus secret-token\n"

		err := deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "line 2: % Invalid input detected")
		require.NotContains(t, err.Error(), "bogus")
		<-d.done
		running, pending, _ := d.state()
		require.Equal(t, []string{"hostname edge"}, running, "the lines before it are reverted too")
		require.False(t, pending)
		require.Contains(t, d.recorded(), "configure revert now")
	})
	t.Run("no archive", func(t *testing.T) {
		t.Parallel()
		d := &fakeIOS{done: make(chan struct{}), running: []string{"hostname edge"}, noArchive: true}
		req := startShell(t, d.serve, deviceconfig.PlatformCiscoIOS)
		req.Candidate = "hostname core\n"

		err := deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "%Archive feature not enabled")
		require.Equal(t, []string{"configure terminal revert timer 5"}, d.recorded(), "nothing is typed")
	})
}

// TestIOSReplace: a replace copies the candidate to flash under the commit's
// name, replaces the running configuration from it under the revert timer,
// and deletes the copy, which carries the whole configuration, whatever the
// replace did.
func TestIOSReplace(t *testing.T) {
	t.Parallel()
	const file = "flash:seed-c1.cfg"
	t.Run("commit", func(t *testing.T) {
		t.Parallel()
		d := &fakeIOS{done: make(chan struct{}), running: []string{"hostname edge"}, flash: map[string]string{}}
		req := startIOS(t, d)
		req.Candidate = "hostname core\r\n!\nenable secret 9 $9$abc\nend"

		require.NoError(t, deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req))
		<-d.done
		require.Equal(t, []string{"configure replace " + file + " force time 5", "delete /force " + file, "exit"},
			d.recorded())
		running, pending, saved := d.state()
		require.Equal(t, []string{"hostname core", "enable secret 9 $9$abc"}, running)
		require.True(t, pending, "the revert timer runs until the confirm")
		require.False(t, saved)
		require.Empty(t, d.files(), "the copy is deleted")
		require.Equal(t, []string{"hostname core\n!\nenable secret 9 $9$abc\nend\n"}, d.copies)
	})
	t.Run("no archive", func(t *testing.T) {
		t.Parallel()
		d := &fakeIOS{
			done: make(chan struct{}), running: []string{"hostname edge"}, flash: map[string]string{}, noArchive: true,
		}
		req := startIOS(t, d)
		req.Candidate = "hostname core\n"

		err := deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req)
		require.ErrorContains(t, err, "%Archive feature not enabled")
		<-d.done
		require.Equal(t, []string{
			"configure replace " + file + " force time 5",
			"show archive config rollback timer",
			"delete /force " + file,
			"exit",
		}, d.recorded())
		running, pending, _ := d.state()
		require.Equal(t, []string{"hostname edge"}, running)
		require.False(t, pending)
		require.Empty(t, d.files())
	})
	t.Run("a rejected line is reverted at once", func(t *testing.T) {
		t.Parallel()
		d := &fakeIOS{done: make(chan struct{}), running: []string{"hostname edge"}, flash: map[string]string{}}
		req := startIOS(t, d)
		req.Candidate = "hostname core\nbogus secret-token\n"

		err := deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "configure replace did not complete: %Rollback failed")
		require.NotContains(t, err.Error(), "secret-token")
		<-d.done
		require.Contains(t, d.recorded(), "configure revert now")
		running, pending, _ := d.state()
		require.Equal(t, []string{"hostname edge"}, running)
		require.False(t, pending)
		require.Empty(t, d.files())
	})
	t.Run("the device refuses the copy", func(t *testing.T) {
		t.Parallel()
		d := &fakeIOS{
			done: make(chan struct{}), running: []string{"hostname edge"}, flash: map[string]string{},
			scpRefused: "Privilege denied",
		}
		req := startIOS(t, d)
		req.Candidate = "hostname core\n"

		err := deviceconfig.SSHFetcher{}.Commit(fetchCtx(t), req)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "the device refused the copy: Privilege denied")
		<-d.done
		require.Equal(t, []string{"delete /force " + file, "exit"}, d.recorded(), "nothing is replaced")
		running, pending, _ := d.state()
		require.Equal(t, []string{"hostname edge"}, running)
		require.False(t, pending)
	})
}

// TestIOSDryRunIsTheRunningConfigurationsPrediction: the post-check reads the
// running configuration from a new login and predicts the candidate against
// it, so a device holding the change has nothing left to change.
func TestIOSDryRunIsTheRunningConfigurationsPrediction(t *testing.T) {
	t.Parallel()
	running := "hostname edge\n!\narchive\n path flash:seed-archive\n!\ninterface Gi1/0/1\n description uplink\n!\nend\n"
	dev := startSSHDevice(t, deviceOpts{password: "wr1te", commands: map[string]string{"show running-config": running}})
	fr := dev.request(t, "wr1te", "", dev.fingerprint)
	req := deviceconfig.DeviceRequest{
		Host: fr.Host, Port: fr.Port, User: "pusher", Password: "wr1te", PinnedHostKey: dev.fingerprint,
		Platform: deviceconfig.PlatformCiscoIOS, Mode: deviceconfig.PushMerge,
	}

	req.Candidate = "interface Gi1/0/1\n description uplink\nend\n"
	res, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
	require.NoError(t, err)
	require.Equal(t, 0, res.Added+res.Removed)

	req.Candidate = "interface Gi1/0/1\n description core\n"
	res, err = deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
	require.NoError(t, err)
	require.Equal(t, 1, res.Added)
	require.Equal(t, 1, res.Removed)

	req.Mode = deviceconfig.PushReplace
	req.Candidate = "hostname edge\narchive\n path flash:seed-archive\ninterface Gi1/0/1\n description uplink\n"
	res, err = deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
	require.NoError(t, err)
	require.Equal(t, 0, res.Added+res.Removed, "a device running the replacement has nothing left to change")
}
