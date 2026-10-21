package deviceconfig

// The device side of image staging (ADR-0035 §3) on IOS/IOS-XE, EOS and Junos.
// Reading the device and hashing the copy are exec commands; setting the boot
// image types configuration in the CLI, as a push does, and is read back.
// Junos installs the copied package with an exec command instead.

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

var (
	// iosVersion is "Cisco IOS XE Software, Version 17.09.04a" or "Cisco
	// IOS Software, C2960X Software (...), Version 15.2(7)E4, RELEASE ...".
	iosVersion     = regexp.MustCompile(`(?m)^Cisco IOS.*?Version ([^\s,]+)`)
	iosSystemImage = regexp.MustCompile(`System image file is "([^"]+)"`)
	eosVersion     = regexp.MustCompile(`(?m)^Software image version: (\S+)`)
	eosBootImage   = regexp.MustCompile(`(?m)^Software image: (\S+)`)
	// freeBytes ends "dir": "1621966848 bytes total (1199271936 bytes free)".
	freeBytes = regexp.MustCompile(`\((\d+) bytes free\)`)
	// verifiedSHA512 ends "verify /sha512": "verify /sha512 (flash:x) = 3c9e...".
	verifiedSHA512 = regexp.MustCompile(`(?m)= *([0-9a-fA-F]{128})\s*$`)
	// junosVersion is "Junos: 21.4R3-S5.4"; before 15.1, "JUNOS Software
	// Release [12.3R12.4]" or "JUNOS Base OS boot [12.3R12.4]".
	junosVersion = regexp.MustCompile(`(?m)^(?:Junos: (\S+)|JUNOS (?:Software Release|Base OS boot) \[([^\]]+)\])`)
	// junosSHA256 is "file checksum sha-256": "SHA256 (/var/tmp/x) = 3c9e...".
	junosSHA256 = regexp.MustCompile(`(?m)= *([0-9a-fA-F]{64})\s*$`)
	// junosRebootRequired is how "request system software add" reports a
	// package installed for the next boot ("WARNING: A REBOOT IS REQUIRED TO
	// LOAD THIS SOFTWARE CORRECTLY", "A reboot is required to complete the
	// install").
	junosRebootRequired = regexp.MustCompile(`(?i)reboot is required`)
)

// junosStagingDir is where Junos takes a package to install: its SCP server
// takes the path as it is, and /var/tmp is the file system meant for them.
const junosStagingDir = "/var/tmp/"

// ImageFacts reads the running version and image and the free space beside
// the image.
func (SSHFetcher) ImageFacts(ctx context.Context, req DeviceRequest) (ImageFacts, error) {
	var facts ImageFacts
	err := withClient(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(client *ssh.Client) error {
			var err error
			facts, err = imageFacts(client, req.Platform)
			return err
		})
	return facts, err
}

func imageFacts(client *ssh.Client, platform Platform) (ImageFacts, error) {
	if platform == PlatformJuniperJunos {
		return junosImageFacts(client)
	}
	version, err := execCommand(client, "show version")
	if err != nil {
		return ImageFacts{}, err
	}
	var facts ImageFacts
	switch platform {
	case PlatformCiscoIOS:
		facts.Version, facts.RunningImage = submatch(iosVersion, version), submatch(iosSystemImage, version)
		facts.InstallMode = fileName(facts.RunningImage) == "packages.conf"
	case PlatformAristaEOS:
		// EOS names only the image it boots, which is the running image
		// unless the boot image was changed since the last reload.
		boot, bootErr := execCommand(client, "show boot-config")
		if bootErr != nil {
			return ImageFacts{}, bootErr
		}
		facts.Version, facts.RunningImage = submatch(eosVersion, version), submatch(eosBootImage, boot)
	case PlatformCiscoNXOS, PlatformNETCONF, PlatformJuniperJunos:
		return ImageFacts{}, fmt.Errorf("%w: cannot stage an image on %s", ErrCommand, platform)
	}
	if facts.Version == "" || facts.RunningImage == "" {
		return ImageFacts{}, fmt.Errorf("%w: the device did not name its running version and image", ErrCommand)
	}
	facts.Dir = "flash:"
	if i := strings.Index(facts.RunningImage, ":"); i >= 0 {
		facts.Dir = facts.RunningImage[:i+1]
	}
	dir, err := execCommand(client, "dir "+facts.Dir)
	if err != nil {
		return ImageFacts{}, err
	}
	if facts.FreeBytes, err = strconv.ParseInt(submatch(freeBytes, dir), 10, 64); err != nil {
		return ImageFacts{}, fmt.Errorf("%w: dir %s did not report the free space", ErrCommand, facts.Dir)
	}
	return facts, nil
}

// junosImageFacts reads the running release and the free space of the file
// system holding /var/tmp. Junos runs installed packages, not an image file,
// so there is no running image to name.
func junosImageFacts(client *ssh.Client) (ImageFacts, error) {
	version, err := junosExec(client, "show version")
	if err != nil {
		return ImageFacts{}, err
	}
	m := junosVersion.FindStringSubmatch(version)
	if m == nil {
		return ImageFacts{}, fmt.Errorf("%w: the device did not name its running release", ErrCommand)
	}
	storage, err := junosExec(client, "show system storage | display xml")
	if err != nil {
		return ImageFacts{}, err
	}
	free, err := junosFreeBytes(storage, junosStagingDir)
	if err != nil {
		return ImageFacts{}, err
	}
	return ImageFacts{Version: m[1] + m[2], Dir: junosStagingDir, FreeBytes: free}, nil
}

// junosBlockSize is the block "show system storage" counts in, BSD df's.
const junosBlockSize = 512

// junosFilesystem is one file system of "show system storage | display xml".
type junosFilesystem struct {
	Available int64  `xml:"available-blocks"`
	MountedOn string `xml:"mounted-on"`
}

// junosFreeBytes is the free space of the file system holding dir, the one
// mounted on its longest prefix. Junos 21 and later mount /var beneath
// /.mount, so a mount there counts too. A chassis with two Routing Engines
// lists the one logged in to first.
func junosFreeBytes(storage, dir string) (int64, error) {
	d := xml.NewDecoder(strings.NewReader(storage))
	for {
		tok, err := d.Token()
		if err != nil {
			return 0, fmt.Errorf("%w: the device did not list its storage", ErrCommand)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "system-storage-information" {
			continue
		}
		var info struct {
			Filesystems []junosFilesystem `xml:"filesystem"`
		}
		if err = d.DecodeElement(&info, &start); err != nil {
			return 0, fmt.Errorf("%w: read the device's storage: %w", ErrCommand, err)
		}
		best := -1
		for i, fs := range info.Filesystems {
			mount := strings.TrimSpace(fs.MountedOn)
			if !mountHolds(mount, dir) && !mountHolds(mount, "/.mount"+dir) {
				continue
			}
			if best < 0 || len(mount) > len(strings.TrimSpace(info.Filesystems[best].MountedOn)) {
				best = i
			}
		}
		if best < 0 {
			return 0, fmt.Errorf("%w: the device listed no file system holding %s", ErrCommand, dir)
		}
		return info.Filesystems[best].Available * junosBlockSize, nil
	}
}

// mountHolds reports whether the file system mounted on mount holds path.
func mountHolds(mount, path string) bool {
	return mount == "/" || path == mount || strings.HasPrefix(path, strings.TrimSuffix(mount, "/")+"/")
}

// CopyImage copies the image to the device as an SCP sink, and deletes what
// a copy that failed part way left.
func (SSHFetcher) CopyImage(ctx context.Context, req DeviceRequest, img DeviceImage, body io.Reader) error {
	return withClient(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(client *ssh.Client) error {
			path := img.Path()
			if req.Platform == PlatformAristaEOS {
				// EOS's SCP server takes a path in its Linux file system,
				// where flash: is /mnt/flash.
				path = "/mnt/" + strings.TrimSuffix(img.Dir, ":") + "/" + img.Name
			}
			err := scpSink(client, path, img.Size, body)
			if err != nil && ctx.Err() == nil {
				_ = deleteImageFile(client, req.Platform, img)
			}
			return err
		})
}

// VerifyImage has the device hash its copy, with SHA-512 (SHA-256 on Junos),
// and deletes a copy whose hash is not the image's.
func (SSHFetcher) VerifyImage(ctx context.Context, req DeviceRequest, img DeviceImage) error {
	command, run, re, want := "verify /sha512 "+img.Path(), execCommand, verifiedSHA512, img.SHA512
	if req.Platform == PlatformJuniperJunos {
		command, run, re, want = "file checksum sha-256 "+img.Path(), junosExec, junosSHA256, img.SHA256
	}
	return withClient(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(client *ssh.Client) error {
			out, err := run(client, command)
			if err != nil {
				return err
			}
			sum := submatch(re, out)
			if sum == "" {
				return fmt.Errorf("%w: %s printed no hash", ErrCommand, command)
			}
			if !strings.EqualFold(sum, want) {
				return errors.Join(ErrImageCorrupt, deleteImageFile(client, req.Platform, img))
			}
			return nil
		})
}

// SetBoot makes the copy the boot image. IOS keeps the running image as the
// second boot statement, which the device falls back to. EOS boots one image;
// its fallback is the running image, still on flash, booted from Aboot. Junos
// installs the package for the next boot and keeps the running software for
// a rollback.
func (SSHFetcher) SetBoot(ctx context.Context, req DeviceRequest, img DeviceImage) error {
	return withClient(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(client *ssh.Client) error {
			switch req.Platform {
			case PlatformAristaEOS:
				return eosSetBoot(client, img)
			case PlatformJuniperJunos:
				return junosSoftwareAdd(client, img)
			case PlatformCiscoIOS:
				return iosSetBoot(client, img)
			case PlatformCiscoNXOS, PlatformNETCONF:
			}
			return fmt.Errorf("%w: cannot stage an image on %s", ErrCommand, req.Platform)
		})
}

// junosSoftwareAdd installs the package for the next boot. Without "reboot"
// the device only validates and unpacks it, and the install is complete only
// when the device says a reboot is required to load it. Junos validates the
// running configuration against the new release by default; a configuration
// it rejects fails the install and leaves the running software as it was.
func junosSoftwareAdd(client *ssh.Client, img DeviceImage) error {
	command := "request system software add " + img.Path()
	out, err := junosExec(client, command)
	if err != nil {
		return err
	}
	if !junosRebootRequired.MatchString(out) {
		return fmt.Errorf("%w: %s did not report the package installed for the next boot", ErrCommand, command)
	}
	return nil
}

// iosSetBoot replaces the boot statements with the new image and then the
// running one, and saves them. Statements that cannot all be set are put back
// as they were, so the device is never left with no boot statement Seed did
// not mean.
func iosSetBoot(client *ssh.Client, img DeviceImage) error {
	const show = "show running-config | include ^boot system"
	before, err := execCommand(client, show)
	if err != nil {
		return err
	}
	want := []string{"boot system " + img.Path(), "boot system " + img.Fallback}
	err = onSession(client, func(session *ssh.Session) error {
		return iosWriteBoot(session, want, bootStatements(before))
	})
	if err != nil {
		return err
	}
	after, err := execCommand(client, show)
	if err != nil {
		return err
	}
	if !slices.Equal(bootStatements(after), want) {
		return fmt.Errorf("%w: the device's boot statements are not the ones set", ErrCommand)
	}
	return nil
}

// iosWriteBoot sets the boot statements to want and saves them, or puts back
// the statements the device had.
func iosWriteBoot(session *ssh.Session, want, had []string) error {
	s, err := openPrivileged(session)
	if err != nil {
		return err
	}
	if err = s.cliCommand("configure terminal"); err != nil {
		return err
	}
	if err = s.iosBootSystem(want); err != nil {
		err = errors.Join(err, s.iosBootSystem(had))
	}
	if err = errors.Join(err, s.cliCommand("end")); err != nil {
		_ = s.send("exit\n")
		return err
	}
	return s.saveStartup()
}

// iosBootSystem replaces the boot statements in configuration mode.
func (s *cliShell) iosBootSystem(statements []string) error {
	if err := s.cliCommand("no boot system"); err != nil {
		return err
	}
	for _, statement := range statements {
		if err := s.cliCommand(statement); err != nil {
			return err
		}
	}
	return nil
}

// bootStatements returns the "boot system" lines of a configuration.
func bootStatements(config string) []string {
	var out []string
	for line := range strings.SplitSeq(config, "\n") {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "boot system ") {
			out = append(out, line)
		}
	}
	return out
}

// eosSetBoot sets the boot image and reads it back. EOS writes it to its boot
// configuration at once; there is nothing to save.
func eosSetBoot(client *ssh.Client, img DeviceImage) error {
	err := onSession(client, func(session *ssh.Session) error {
		s, err := openPrivileged(session)
		if err != nil {
			return err
		}
		if err = s.cliCommand("configure"); err != nil {
			return err
		}
		err = errors.Join(s.cliCommand("boot system "+img.Path()), s.cliCommand("end"))
		_ = s.send("exit\n")
		return err
	})
	if err != nil {
		return err
	}
	boot, err := execCommand(client, "show boot-config")
	if err != nil {
		return err
	}
	if fileName(submatch(eosBootImage, boot)) != img.Name {
		return fmt.Errorf("%w: the device's boot image is not the one set", ErrCommand)
	}
	return nil
}

// deleteImageFile deletes a staged copy. It never names the running image:
// staging refuses an image of the running image's name.
func deleteImageFile(client *ssh.Client, platform Platform, img DeviceImage) error {
	command, run := "delete /force "+img.Path(), execCommand
	switch platform {
	case PlatformAristaEOS:
		command = "delete " + img.Path()
	case PlatformJuniperJunos:
		command, run = "file delete "+img.Path(), junosExec
	case PlatformCiscoIOS, PlatformCiscoNXOS, PlatformNETCONF:
	}
	_, err := run(client, command)
	return err
}

// execCommand runs a command on a new exec channel and fails on an error the
// device printed rather than exited with.
func execCommand(client *ssh.Client, command string) (string, error) {
	return execChecked(client, command, cliErrors)
}

// junosExec is execCommand for Junos, which prints "error:" lines.
func junosExec(client *ssh.Client, command string) (string, error) {
	return execChecked(client, command, junosErrors)
}

func execChecked(client *ssh.Client, command string, printed func(string) []string) (string, error) {
	var out string
	err := onSession(client, func(session *ssh.Session) error {
		var err error
		out, err = runCommand(session, command)
		return err
	})
	if err != nil {
		return "", err
	}
	if msgs := printed(out); len(msgs) > 0 {
		return "", fmt.Errorf("%w: %s%s", ErrCommand, command, joinMessages(msgs))
	}
	return out, nil
}

func submatch(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}
