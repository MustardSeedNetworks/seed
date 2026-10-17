package deviceconfig_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

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
