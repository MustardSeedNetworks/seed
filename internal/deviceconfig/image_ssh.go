package deviceconfig

// The device side of image staging (ADR-0035 §3) on IOS/IOS-XE and EOS.
// Reading the device and hashing the copy are exec commands; setting the boot
// image types configuration in the CLI, as a push does, and is read back.

import (
	"context"
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
)

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
	case PlatformJuniperJunos, PlatformCiscoNXOS, PlatformNETCONF:
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

// VerifyImage has the device hash its copy with SHA-512 and deletes a copy
// whose hash is not the image's.
func (SSHFetcher) VerifyImage(ctx context.Context, req DeviceRequest, img DeviceImage) error {
	return withClient(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(client *ssh.Client) error {
			out, err := execCommand(client, "verify /sha512 "+img.Path())
			if err != nil {
				return err
			}
			sum := submatch(verifiedSHA512, out)
			if sum == "" {
				return fmt.Errorf("%w: verify /sha512 printed no hash", ErrCommand)
			}
			if !strings.EqualFold(sum, img.SHA512) {
				return errors.Join(ErrImageCorrupt, deleteImageFile(client, req.Platform, img))
			}
			return nil
		})
}

// SetBoot makes the copy the boot image. IOS keeps the running image as the
// second boot statement, which the device falls back to. EOS boots one image;
// its fallback is the running image, still on flash, booted from Aboot.
func (SSHFetcher) SetBoot(ctx context.Context, req DeviceRequest, img DeviceImage) error {
	return withClient(ctx, req.Host, req.Port, req.User, req.Password, req.PinnedHostKey,
		func(client *ssh.Client) error {
			if req.Platform == PlatformAristaEOS {
				return eosSetBoot(client, img)
			}
			return iosSetBoot(client, img)
		})
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
	command := "delete /force " + img.Path()
	if platform == PlatformAristaEOS {
		command = "delete " + img.Path()
	}
	_, err := execCommand(client, command)
	return err
}

// execCommand runs a command on a new exec channel and fails on an error the
// device printed rather than exited with.
func execCommand(client *ssh.Client, command string) (string, error) {
	var out string
	err := onSession(client, func(session *ssh.Session) error {
		var err error
		out, err = runCommand(session, command)
		return err
	})
	if err != nil {
		return "", err
	}
	if msgs := cliErrors(out); len(msgs) > 0 {
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
