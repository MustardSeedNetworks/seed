package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// The two principals that may own the Windows data directory and appear in its
// access list: SYSTEM and Administrators, as SID strings and as the SDDL
// aliases Windows renders them with.
const (
	sidSystem         = "S-1-5-18"
	sidAdministrators = "S-1-5-32-544"
)

var sddlACE = regexp.MustCompile(`\(([^;)]*);[^;)]*;[^;)]*;[^;)]*;[^;)]*;([^;)]*)\)`)

func trustedTrustee(t string) bool {
	switch t {
	case "SY", "BA", sidSystem, sidAdministrators:
		return true
	}
	return false
}

// checkProtectedDACL fails unless owner is SYSTEM or Administrators and the
// SDDL's DACL is protected (inherits nothing) and contains only Allow entries
// for SYSTEM and Administrators. It reads a descriptor; it never writes one.
//
// TODO(#3848): securedir.Verify replaces this.
func checkProtectedDACL(owner, sddl string) error {
	if !trustedTrustee(owner) {
		return fmt.Errorf("owned by %s, not SYSTEM or Administrators", owner)
	}
	_, dacl, found := strings.Cut(sddl, "D:")
	if !found {
		return errors.New("has no access list")
	}
	flags, _, _ := strings.Cut(dacl, "(")
	if !strings.Contains(flags, "P") {
		return errors.New("inherits permissions from its parent")
	}
	aces := sddlACE.FindAllStringSubmatch(dacl, -1)
	if len(aces) == 0 {
		return errors.New("grants nobody access")
	}
	for _, ace := range aces {
		if ace[1] != "A" {
			return fmt.Errorf("has a %q entry, only Allow entries are expected", ace[1])
		}
		if !trustedTrustee(ace[2]) {
			return fmt.Errorf("grants access to %s, not just SYSTEM and Administrators", ace[2])
		}
	}
	return nil
}
