//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

// protectDirForTest gives dir the descriptor install.ps1 creates: owned by
// Administrators, protected, SYSTEM and Administrators full control.
func protectDirForTest(tb testing.TB, dir string) {
	tb.Helper()
	applyDACLForTest(tb, dir, "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
}

func applyDACLForTest(tb testing.TB, dir, sddl string) {
	tb.Helper()
	sd, err := windows.SecurityDescriptorFromString("O:BAG:BA" + sddl)
	if err != nil {
		tb.Fatalf("parse SDDL: %v", err)
	}
	owner, _, _ := sd.Owner()
	dacl, _, _ := sd.DACL()
	var info windows.SECURITY_INFORMATION = windows.OWNER_SECURITY_INFORMATION | windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION
	if err = windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, info, owner, nil, dacl, nil); err != nil {
		tb.Fatalf("set permissions on %s: %v", dir, err)
	}
}
