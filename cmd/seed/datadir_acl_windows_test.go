//go:build windows

package main

import "testing"

func TestVerifyDirSecurity(t *testing.T) {
	tests := []struct {
		name    string
		sddl    string
		wantErr bool
	}{
		{"install.ps1 descriptor", "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)", false},
		{"users can read", "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FRFX;;;BU)", true},
		{"SYSTEM only is accepted", "D:PAI(A;OICI;FA;;;SY)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			applyDACLForTest(t, dir, tt.sddl)
			if err := verifyDirSecurity(dir); (err != nil) != tt.wantErr {
				t.Errorf("verifyDirSecurity error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// A directory that merely inherits its parent's permissions is what a user can
// create under %ProgramData%, and must be refused.
func TestVerifyDirSecurityRefusesInheritedDACL(t *testing.T) {
	if err := verifyDirSecurity(t.TempDir()); err == nil {
		t.Fatal("an unprotected directory was accepted")
	}
}
