//go:build windows

package main

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

const (
	installerACL = "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
	usersRead    = "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FRFX;;;BU)"
	usersWrite   = "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;BU)"
)

func dirSDDL(t *testing.T, dir string) string {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatalf("read permissions of %s: %v", dir, err)
	}
	return sd.String()
}

func TestVerifyDataDirOnWindows(t *testing.T) {
	tests := []struct {
		name    string
		sddl    string
		wantErr bool
	}{
		{"install.ps1 descriptor", installerACL, false},
		{"users can write", usersWrite, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := privateTempDir(t)
			applyDACLForTest(t, dir, tt.sddl)
			err := verifyDataDir(dir, testDirOptions(t))
			if (err != nil) != tt.wantErr {
				t.Fatalf("verifyDataDir error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, securedir.ErrUnsafeAccess) {
				t.Errorf("error = %v, want ErrUnsafeAccess", err)
			}
		})
	}
}

// securedir judges who can change a tree, so a Users read grant passes Verify;
// the service closes reads by taking the tree over with Ensure at start.
func TestPrepareServiceDirsClosesAUsersReadGrant(t *testing.T) {
	opts := testDirOptions(t)
	sp := servicePathsUnder(privateTempDir(t))
	applyDACLForTest(t, sp.Root, usersRead)
	if !strings.Contains(dirSDDL(t, sp.Root), ";BU)") {
		t.Fatal("test setup: the Users entry was not applied")
	}

	if _, err := prepareServiceDirs(sp, opts); err != nil {
		t.Fatalf("prepareServiceDirs: %v", err)
	}
	if sddl := dirSDDL(t, sp.Root); strings.Contains(sddl, ";BU)") || !strings.Contains(sddl, "D:P") {
		t.Errorf("root still grants Users access or inherits: %s", sddl)
	}
}

// A Users-writable directory under the root is content nobody vouched for:
// Ensure takes a tree over only when everything in it already passes, so the
// service refuses to start rather than adopting it.
func TestPrepareServiceDirsRefusesAWritableCertDir(t *testing.T) {
	opts := testDirOptions(t)
	sp := servicePathsUnder(privateTempDir(t))
	if _, err := prepareServiceDirs(sp, opts); err != nil {
		t.Fatal(err)
	}
	applyDACLForTest(t, sp.CertDir, usersWrite)

	if _, err := prepareServiceDirs(sp, opts); !errors.Is(err, securedir.ErrUnsafeAccess) {
		t.Fatalf("prepareServiceDirs error = %v, want ErrUnsafeAccess", err)
	}
}
