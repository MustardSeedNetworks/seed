//go:build windows

package main

import "testing"

func TestCheckServiceExecutable(t *testing.T) {
	const installDir = `C:\Program Files\Seed`

	tests := []struct {
		name    string
		exe     string
		wantErr bool
	}{
		{"installed binary", `C:\Program Files\Seed\seed.exe`, false},
		{"case differs", `c:\program files\seed\SEED.EXE`, false},
		{"nested under the install directory", `C:\Program Files\Seed\bin\seed.exe`, false},
		{"unclean path inside", `C:\Program Files\Seed\.\bin\..\seed.exe`, false},
		{"download folder", `C:\Users\alice\Downloads\seed.exe`, true},
		{"sibling with the same prefix", `C:\Program Files\SeedEvil\seed.exe`, true},
		{"parent of the install directory", `C:\Program Files\seed.exe`, true},
		{"dot-dot escape", `C:\Program Files\Seed\..\Evil\seed.exe`, true},
		{"the install directory itself", installDir, true},
		{"x86 program files", `C:\Program Files (x86)\Seed\seed.exe`, true},
		{"another drive", `D:\Program Files\Seed\seed.exe`, true},
		{"UNC share", `\\server\share\Seed\seed.exe`, true},
		{"relative", `seed.exe`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkServiceExecutable(tt.exe, installDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkServiceExecutable(%q) error = %v, wantErr %v", tt.exe, err, tt.wantErr)
			}
		})
	}
}

func TestCheckServiceExecutableRejectsRelativeInstallDir(t *testing.T) {
	if err := checkServiceExecutable(`C:\Program Files\Seed\seed.exe`, `Seed`); err == nil {
		t.Fatal("expected an error for a relative install directory")
	}
}
