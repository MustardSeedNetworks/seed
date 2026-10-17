package deviceconfig_test

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

func (e *fakeEOS) held() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.lines...)
}

func startShell(t *testing.T, serve func(ssh.Channel), platform deviceconfig.Platform) deviceconfig.DryRunRequest {
	t.Helper()
	dev := startSSHDevice(t, deviceOpts{password: "wr1te", shell: serve})
	fr := dev.request(t, "wr1te", "", dev.fingerprint)
	return deviceconfig.DryRunRequest{
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
