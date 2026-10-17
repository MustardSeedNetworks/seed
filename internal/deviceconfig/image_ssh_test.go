package deviceconfig_test

import (
	"bufio"
	"fmt"
	"io"
	"maps"
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

// fakeStaging is the part of an IOS or EOS device staging touches: its flash,
// its boot statements, the read-only commands staging runs on an exec
// channel, and the configuration commands that set the boot image in the CLI.
type fakeStaging struct {
	fakeCLI

	eos   bool
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
	if d.eos {
		platform = deviceconfig.PlatformAristaEOS
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

func (d *fakeStaging) exec(command string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.commands = append(d.commands, command)
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
		used := int64(0)
		for _, body := range d.flash {
			used += int64(len(body))
		}
		return fmt.Sprintf(
			"Directory of flash:/\r\n\r\n%d bytes total (%d bytes free)\r\n",
			flashTotal,
			d.free-used,
		), true
	case "show running-config | include ^boot system":
		return strings.Join(d.boot, "\r\n") + "\r\n", true
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
		var out string
		out, mode = d.answer(cmd, mode)
		if cmd == "exit" && mode == "#" {
			return
		}
		_, _ = io.WriteString(channel, out+"sw1"+mode)
	}
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
