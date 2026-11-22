//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

// testDirOptions trusts the account running the tests: a temporary directory
// is owned by that account, not by SYSTEM or Administrators.
func testDirOptions(tb testing.TB) securedir.Options {
	tb.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		tb.Fatalf("read process token user: %v", err)
	}
	return securedir.Options{ExtraTrustedSIDs: []string{user.User.Sid.String()}}
}

// applyDACLForTest gives dir the owner and DACL in sddl, for refusal cases.
func applyDACLForTest(tb testing.TB, dir, sddl string) {
	tb.Helper()
	sd, err := windows.SecurityDescriptorFromString("O:BAG:BA" + sddl)
	if err != nil {
		tb.Fatalf("parse SDDL: %v", err)
	}
	owner, _, _ := sd.Owner()
	dacl, _, _ := sd.DACL()
	var info windows.SECURITY_INFORMATION = windows.OWNER_SECURITY_INFORMATION |
		windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION
	if err = windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, info, owner, nil, dacl, nil); err != nil {
		tb.Fatalf("set permissions on %s: %v", dir, err)
	}
}
