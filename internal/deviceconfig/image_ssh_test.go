package deviceconfig_test

import (
	"bufio"
	"fmt"
	"io"
	"maps"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

const (
	iosOldImage = "flash:cat9k_iosxe.17.09.04a.SPA.bin"
	eosOldImage = "flash:/EOS-4.31.2F.swi"
	flashTotal  = 1621966848
)

// fakeStaging is the part of an IOS, EOS or Junos device staging touches: its
// flash, its boot statements, the read-only commands staging runs on an exec
// channel, and the configuration commands that set the boot image in the CLI.
type fakeStaging struct {
	fakeCLI

	eos   bool
	junos bool
	free  int64
	flash map[string]string
	boot  []string
	saved bool
	// corrupt hashes every copy wrongly; refuseCopy refuses every copy part
	// way, leaving what it received; rejectBoot rejects a boot statement
	// naming the new image.
	corrupt, refuseCopy, rejectBoot bool
	// running is the image show version names (IOS).
	running string
	// junosVersion is what Junos's show version prints; added is the package
	// "request system software add" installed, and addOut what it printed.
	junosVersion, added, addOut string
	// modified is a running configuration with unsaved changes, which the
	// reload asks to save; reloaded is a reload the device went on with.
	modified, reloaded bool
}

// newFakeJunos is an EX switch running 21.4R3-S5.4 whose /var has 1 GiB free.
func newFakeJunos() *fakeStaging {
	return &fakeStaging{
		done: make(chan struct{}), junos: true, free: 1 << 30, flash: map[string]string{},
		junosVersion: "Hostname: ex1\r\nModel: ex2300-24t\r\nJunos: 21.4R3-S5.4\r\n",
		addOut: "Verified junos-install-ex-x86-64-23.4R2.13.tgz signed by PackageProductionECP256_2024\r\n" +
			"WARNING: A reboot is required to install the software\r\n" +
			"WARNING:     Use the 'request system reboot' command immediately\r\n",
	}
}

func newFakeStaging(eos bool) *fakeStaging {
	d := &fakeStaging{
		done: make(chan struct{}), eos: eos, free: 1 << 30,
		flash: map[string]string{}, running: iosOldImage,
	}
	if eos {
		d.boot = []string{eosOldImage}
	} else {
		d.boot = []string{"boot system " + iosOldImage}
	}
	return d
}

func (d *fakeStaging) start(t *testing.T) deviceconfig.DeviceRequest {
	t.Helper()
	dev := startSSHDevice(t, deviceOpts{password: "wr1te", shell: d.serve, scp: d.scp, exec: d.exec})
	fr := dev.request(t, "wr1te", "", dev.fingerprint)
	platform := deviceconfig.PlatformCiscoIOS
	switch {
	case d.eos:
		platform = deviceconfig.PlatformAristaEOS
	case d.junos:
		platform = deviceconfig.PlatformJuniperJunos
	}
	return deviceconfig.DeviceRequest{
		Host: fr.Host, Port: fr.Port, User: "pusher", Password: "wr1te", PinnedHostKey: dev.fingerprint,
		Platform: platform,
	}
}

// flashPath is the CLI's name for a path the SCP server was given: EOS's
// SCP server takes /mnt/flash/x for flash:x.
func flashPath(path string) string {
	if rest, ok := strings.CutPrefix(path, "/mnt/flash/"); ok {
		return "flash:" + rest
	}
	return path
}

func (d *fakeStaging) scp(ch ssh.Channel, path string) {
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
	if _, err = fmt.Sscanf(header, "C%s %d %s", &perm, &size, &name); err != nil {
		return
	}
	_, _ = ch.Write([]byte{0})
	if d.refuseCopy {
		// Flash fills half way: the device keeps what it wrote.
		part := make([]byte, size/2)
		_, _ = io.ReadFull(r, part)
		d.mu.Lock()
		d.flash[flashPath(path)] = string(part)
		d.mu.Unlock()
		_, _ = io.WriteString(ch, "\x01flash is full\n")
		return
	}
	body := make([]byte, size+1)
	if _, err = io.ReadFull(r, body); err != nil || body[size] != 0 {
		return
	}
	d.mu.Lock()
	d.flash[flashPath(path)] = string(body[:size])
	d.mu.Unlock()
	_, _ = ch.Write([]byte{0})
}

func (d *fakeStaging) used() int64 {
	used := int64(0)
	for _, body := range d.flash {
		used += int64(len(body))
	}
	return used
}

func (d *fakeStaging) exec(command string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.commands = append(d.commands, command)
	if d.junos {
		return d.junosExec(command)
	}
	if file, ok := strings.CutPrefix(command, "verify /sha512 "); ok {
		body, have := d.flash[file]
		if !have {
			return "%Error opening " + file + " (No such file or directory)\r\n", true
		}
		if d.corrupt {
			body += "x"
		}
		return "....................Done!\r\nverify /sha512 (" + file + ") = " + sha512Hex(body) + "\r\n", true
	}
	for _, prefix := range []string{"delete /force ", "delete "} {
		if file, ok := strings.CutPrefix(command, prefix); ok {
			delete(d.flash, file)
			return "", true
		}
	}
	switch command {
	case "show version":
		if d.eos {
			return "Arista DCS-7050SX3-48YC8\r\nSoftware image version: 4.31.2F\r\nArchitecture: x86_64\r\n", true
		}
		return "Cisco IOS XE Software, Version 17.09.04a\r\nCisco IOS Software [Cupertino], Catalyst L3 Switch " +
			"Software (CAT9K_IOSXE), Version 17.9.4a, RELEASE SOFTWARE (fc3)\r\n" +
			"System image file is \"" + d.running + "\"\r\n", true
	case "show boot-config":
		return "Software image: " + d.boot[0] + "\r\nConsole speed: (not set)\r\n", true
	case "dir flash:":
		return fmt.Sprintf(
			"Directory of flash:/\r\n\r\n%d bytes total (%d bytes free)\r\n",
			flashTotal,
			d.free-d.used(),
		), true
	case "show running-config | include ^boot system":
		return strings.Join(d.boot, "\r\n") + "\r\n", true
	}
	return "", false
}

// junosExec answers as Junos 21 does, where /var is mounted beneath /.mount
// and a smaller root and /tmp are listed around it.
func (d *fakeStaging) junosExec(command string) (string, bool) {
	if file, ok := strings.CutPrefix(command, "file checksum sha-256 "); ok {
		body, have := d.flash[file]
		if !have {
			return "error: could not resolve file: " + file + "\r\n", true
		}
		if d.corrupt {
			body += "x"
		}
		return "SHA256 (" + file + ") = " + sha256Hex(body) + "\r\n", true
	}
	if file, ok := strings.CutPrefix(command, "file delete "); ok {
		delete(d.flash, file)
		return "", true
	}
	if file, ok := strings.CutPrefix(command, "request system software add "); ok {
		if _, have := d.flash[file]; !have {
			return "error: file does not exist: " + file + "\r\n", true
		}
		if !strings.Contains(d.addOut, "ERROR:") {
			d.added = file
		}
		return d.addOut, true
	}
	switch command {
	case "show version":
		return d.junosVersion, true
	case "show system storage | display xml":
		fs := func(name, mount string, avail int64) string {
			return "<filesystem><filesystem-name>" + name + "</filesystem-name>" +
				`<available-blocks junos:format="x">` + strconv.FormatInt(avail/512, 10) + "</available-blocks>" +
				"<mounted-on>" + mount + "</mounted-on></filesystem>"
		}
		return `<rpc-reply xmlns:junos="http://xml.juniper.net/junos/21.4R3/junos">` + "\r\n" +
			"<system-storage-information>" +
			fs("/dev/gpt/junos", "/.mount", 24<<20) +
			fs("tmpfs", "/.mount/tmp", 512<<20) +
			fs("/dev/gpt/var", "/.mount/var", d.free-d.used()) +
			"</system-storage-information></rpc-reply>\r\n{master:0}\r\n", true
	}
	return "", false
}

func (d *fakeStaging) serve(channel ssh.Channel) {
	defer close(d.done)
	r := bufio.NewReader(channel)
	mode := "#"
	_, _ = io.WriteString(channel, "\r\nsw1"+mode)
	for {
		line, err := readLine(r, channel)
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(line)
		d.record(cmd)
		if cmd == "reload" || cmd == "request system reboot" {
			if d.reload(r, channel, cmd) {
				return
			}
			_, _ = io.WriteString(channel, "\r\nsw1"+mode)
			continue
		}
		var out string
		out, mode = d.answer(cmd, mode)
		if cmd == "exit" && mode == "#" {
			return
		}
		_, _ = io.WriteString(channel, out+"sw1"+mode)
	}
}

// reload asks what the device's CLI asks before it reloads and reports
// whether the device went on with it.
func (d *fakeStaging) reload(r *bufio.Reader, ch ssh.Channel, cmd string) bool {
	question, yes := "Proceed with reload? [confirm]", ""
	switch {
	case cmd == "request system reboot":
		question, yes = "Reboot the system ? [yes,no] (no) ", "yes"
	case d.modified && d.eos:
		question = "System configuration has been modified. Save? [yes/no/cancel/diff]:"
	case d.modified:
		question = "System configuration has been modified. Save? [yes/no]: "
	}
	_, _ = io.WriteString(ch, question)
	answer, err := readLine(r, ch)
	if err != nil || answer != yes {
		return false
	}
	d.mu.Lock()
	d.reloaded = true
	d.mu.Unlock()
	return true
}

func (d *fakeStaging) answer(cmd, mode string) (string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	const invalid = "% Invalid input detected at '^' marker.\r\n"
	if mode == "#" {
		switch cmd {
		case "configure terminal", "configure":
			return "", "(config)#"
		case "write memory":
			d.saved = true
			return "Building configuration...\r\n[OK]\r\n", mode
		case "exit":
			return "", mode
		}
		return invalid, mode
	}
	image, isBoot := strings.CutPrefix(cmd, "boot system ")
	switch {
	case cmd == "end":
		return "", "#"
	case cmd == "no boot system" && !d.eos:
		d.boot = nil
	case isBoot && d.rejectBoot && !strings.Contains(image, "17.09") && !strings.Contains(image, "4.31"):
		return "%Error: " + image + " is not a valid image\r\n", mode
	case isBoot && d.eos:
		d.boot = []string{image}
	case isBoot:
		d.boot = append(d.boot, cmd)
	default:
		return invalid, mode
	}
	return "", mode
}

func (d *fakeStaging) junosState() (map[string]string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return maps.Clone(d.flash), d.added
}

func (d *fakeStaging) state() (map[string]string, []string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return maps.Clone(d.flash), append([]string(nil), d.boot...), d.saved
}

// TestStageAnImageOnTheDevice is the device half of the slice's acceptance
// on both platforms: the facts are read, the image is copied as an SCP sink,
// the device's SHA-512 of its copy is checked, and the boot image is set and
// read back, keeping the running image as IOS's fallback.
func TestStageAnImageOnTheDevice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		eos   bool
		facts deviceconfig.ImageFacts
		image string
		boot  []string
		saved bool
	}{
		{
			name: "ios",
			facts: deviceconfig.ImageFacts{
				Version: "17.09.04a", RunningImage: iosOldImage, Dir: "flash:", FreeBytes: 1 << 30,
				BootImage: iosOldImage,
			},
			image: "cat9k_iosxe.17.12.04.SPA.bin",
			boot:  []string{"boot system flash:cat9k_iosxe.17.12.04.SPA.bin", "boot system " + iosOldImage},
			saved: true,
		},
		{
			name: "eos",
			eos:  true,
			facts: deviceconfig.ImageFacts{
				Version: "4.31.2F", RunningImage: eosOldImage, Dir: "flash:", FreeBytes: 1 << 30,
				BootImage: eosOldImage,
			},
			image: "EOS-4.32.1F.swi",
			boot:  []string{"flash:EOS-4.32.1F.swi"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := newFakeStaging(tt.eos)
			req := d.start(t)
			ctx := fetchCtx(t)
			var dev deviceconfig.SSHFetcher

			facts, err := dev.ImageFacts(ctx, req)
			require.NoError(t, err)
			require.Equal(t, tt.facts, facts)

			img := deviceconfig.DeviceImage{
				Dir: facts.Dir, Name: tt.image, Size: int64(len(imageBytes)), SHA512: sha512Hex(imageBytes),
				Fallback: facts.RunningImage,
			}
			require.NoError(t, dev.CopyImage(ctx, req, img, strings.NewReader(imageBytes)))
			require.NoError(t, dev.VerifyImage(ctx, req, img))
			require.NoError(t, dev.SetBoot(ctx, req, img))

			flash, boot, saved := d.state()
			require.Equal(t, map[string]string{"flash:" + tt.image: imageBytes}, flash)
			require.Equal(t, tt.boot, boot)
			require.Equal(t, tt.saved, saved, "IOS saves its boot statements; EOS has nothing to save")
		})
	}
}

// TestStageImageFailures: a copy the device hashes differently or refuses
// part way is deleted, and a boot statement the device rejects puts the
// statements back as they were.
func TestStageImageFailures(t *testing.T) {
	t.Parallel()
	img := deviceconfig.DeviceImage{
		Dir: "flash:", Name: "cat9k_iosxe.17.12.04.SPA.bin", Size: int64(len(imageBytes)),
		SHA512: sha512Hex(imageBytes), Fallback: iosOldImage,
	}

	t.Run("corrupt copy", func(t *testing.T) {
		t.Parallel()
		d := newFakeStaging(false)
		d.corrupt = true
		req := d.start(t)
		var dev deviceconfig.SSHFetcher
		require.NoError(t, dev.CopyImage(fetchCtx(t), req, img, strings.NewReader(imageBytes)))
		require.ErrorIs(t, dev.VerifyImage(fetchCtx(t), req, img), deviceconfig.ErrImageCorrupt)
		flash, _, _ := d.state()
		require.Empty(t, flash, "the corrupt copy is deleted")
	})

	t.Run("copy refused part way", func(t *testing.T) {
		t.Parallel()
		d := newFakeStaging(false)
		d.refuseCopy = true
		req := d.start(t)
		err := deviceconfig.SSHFetcher{}.CopyImage(fetchCtx(t), req, img, strings.NewReader(imageBytes))
		require.ErrorContains(t, err, "flash is full")
		flash, _, _ := d.state()
		require.Empty(t, flash, "what the copy left is deleted")
	})

	t.Run("boot rejected", func(t *testing.T) {
		t.Parallel()
		d := newFakeStaging(false)
		d.rejectBoot = true
		req := d.start(t)
		err := deviceconfig.SSHFetcher{}.SetBoot(fetchCtx(t), req, img)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "is not a valid image")
		_, boot, saved := d.state()
		require.Equal(t, []string{"boot system " + iosOldImage}, boot, "the old statements are put back")
		require.False(t, saved, "nothing is saved")
	})

	t.Run("install mode", func(t *testing.T) {
		t.Parallel()
		d := newFakeStaging(false)
		d.running = "flash:packages.conf"
		facts, err := deviceconfig.SSHFetcher{}.ImageFacts(fetchCtx(t), d.start(t))
		require.NoError(t, err)
		require.True(t, facts.InstallMode)
	})
}

const junosPackage = "junos-install-ex-x86-64-23.4R2.13.tgz"

// TestStageAJunosPackage is the device half of slice 5b-3: the release and
// the free space of /var/tmp's file system are read, the package is copied
// there as an SCP sink, the device's SHA-256 of its copy is checked, and the
// package is installed for the next boot.
func TestStageAJunosPackage(t *testing.T) {
	t.Parallel()
	d := newFakeJunos()
	req := d.start(t)
	ctx := fetchCtx(t)
	var dev deviceconfig.SSHFetcher

	facts, err := dev.ImageFacts(ctx, req)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.ImageFacts{Version: "21.4R3-S5.4", Dir: "/var/tmp/", FreeBytes: 1 << 30}, facts)

	img := deviceconfig.DeviceImage{
		Dir: facts.Dir, Name: junosPackage, Size: int64(len(imageBytes)),
		SHA256: sha256Hex(imageBytes), SHA512: sha512Hex(imageBytes),
	}
	require.NoError(t, dev.CopyImage(ctx, req, img, strings.NewReader(imageBytes)))
	require.NoError(t, dev.VerifyImage(ctx, req, img))
	require.NoError(t, dev.SetBoot(ctx, req, img))

	flash, added := d.junosState()
	require.Equal(t, map[string]string{"/var/tmp/" + junosPackage: imageBytes}, flash)
	require.Equal(t, "/var/tmp/"+junosPackage, added)
}

// TestJunosStagingFailures: a release show version does not name, a copy
// the device hashes differently, and an install the device refuses or does
// not finish.
func TestJunosStagingFailures(t *testing.T) {
	t.Parallel()
	img := deviceconfig.DeviceImage{
		Dir: "/var/tmp/", Name: junosPackage, Size: int64(len(imageBytes)),
		SHA256: sha256Hex(imageBytes), SHA512: sha512Hex(imageBytes),
	}
	staged := func(d *fakeStaging) { d.flash[img.Path()] = imageBytes }

	t.Run("release before 15.1", func(t *testing.T) {
		t.Parallel()
		d := newFakeJunos()
		d.junosVersion = "Hostname: ex2200\r\nModel: ex2200-24t-4g\r\nJUNOS Base OS boot [12.3R12.4]\r\n"
		facts, err := deviceconfig.SSHFetcher{}.ImageFacts(fetchCtx(t), d.start(t))
		require.NoError(t, err)
		require.Equal(t, "12.3R12.4", facts.Version)
	})

	t.Run("no release", func(t *testing.T) {
		t.Parallel()
		d := newFakeJunos()
		d.junosVersion = "Hostname: ex1\r\n"
		_, err := deviceconfig.SSHFetcher{}.ImageFacts(fetchCtx(t), d.start(t))
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "did not name its running release")
	})

	t.Run("corrupt copy", func(t *testing.T) {
		t.Parallel()
		d := newFakeJunos()
		d.corrupt = true
		staged(d)
		require.ErrorIs(t, deviceconfig.SSHFetcher{}.VerifyImage(fetchCtx(t), d.start(t), img),
			deviceconfig.ErrImageCorrupt)
		flash, _ := d.junosState()
		require.Empty(t, flash, "the corrupt copy is deleted")
	})

	t.Run("install refused", func(t *testing.T) {
		t.Parallel()
		d := newFakeJunos()
		d.addOut = "Checking compatibility with configuration\r\n" +
			"ERROR: Validation failed for the configuration\r\n"
		staged(d)
		err := deviceconfig.SSHFetcher{}.SetBoot(fetchCtx(t), d.start(t), img)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "ERROR: Validation failed")
		_, added := d.junosState()
		require.Empty(t, added)
	})

	t.Run("install not finished", func(t *testing.T) {
		t.Parallel()
		d := newFakeJunos()
		d.addOut = "Installing package '/var/tmp/" + junosPackage + "' ...\r\n"
		staged(d)
		err := deviceconfig.SSHFetcher{}.SetBoot(fetchCtx(t), d.start(t), img)
		require.ErrorIs(t, err, deviceconfig.ErrCommand)
		require.ErrorContains(t, err, "did not report the package installed for the next boot")
	})
}

// TestReloadTheDevice: IOS and EOS go on with a reload at their confirm,
// Junos at its yes/no question, and a modified running configuration is left
// as it is, unsaved, with no reload.
func TestReloadTheDevice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		device   func() *fakeStaging
		modified bool
		command  string
	}{
		{name: "ios", device: func() *fakeStaging { return newFakeStaging(false) }, command: "reload"},
		{name: "eos", device: func() *fakeStaging { return newFakeStaging(true) }, command: "reload"},
		{name: "junos", device: newFakeJunos, command: "request system reboot"},
		{name: "ios modified", device: func() *fakeStaging { return newFakeStaging(false) }, modified: true},
		{name: "eos modified", device: func() *fakeStaging { return newFakeStaging(true) }, modified: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := tt.device()
			d.modified = tt.modified
			err := deviceconfig.SSHFetcher{}.Reload(fetchCtx(t), d.start(t))
			if tt.modified {
				require.ErrorIs(t, err, deviceconfig.ErrUnsavedChanges)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.command, d.recorded()[0])
			}
			<-d.done
			d.mu.Lock()
			defer d.mu.Unlock()
			require.Equal(t, !tt.modified, d.reloaded)
			require.False(t, d.saved, "a reload never saves the configuration")
		})
	}
}
