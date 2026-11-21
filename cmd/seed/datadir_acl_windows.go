//go:build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// verifyDirSecurity fails closed unless path is owned by SYSTEM or
// Administrators and grants only them access, with inheritance off.
//
// TODO(#3848): securedir.Verify replaces this.
func verifyDirSecurity(path string) error {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read permissions of %s: %w", path, err)
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return fmt.Errorf("read owner of %s: %w", path, err)
	}
	if checkErr := checkProtectedDACL(owner.String(), sd.String()); checkErr != nil {
		return fmt.Errorf("%s is not a trusted data directory: it %w", path, checkErr)
	}
	return nil
}
