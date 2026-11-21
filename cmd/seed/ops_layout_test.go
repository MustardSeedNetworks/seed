package main

import (
	"path/filepath"
	"testing"
)

func TestResolveLayoutDefaults(t *testing.T) {
	in := newOpsInstall(t, func(doc map[string]any) {
		setPath(doc, "logging.file", "")
		setPath(doc, "server.cert_file", "")
		setPath(doc, "server.key_file", "")
		setPath(doc, "database.path", "")
	})
	state := newCLIState()
	initCommands(state)
	state.cfgFile = in.configPath
	cmd, _, err := state.rootCmd.Find([]string{"status"})
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Flags().Set(workDirFlag, in.workDir); err != nil {
		t.Fatal(err)
	}
	l, err := resolveLayout(state, cmd)
	if err != nil {
		t.Fatalf("resolveLayout: %v", err)
	}
	want := layout{
		ConfigPath: in.configPath,
		WorkDir:    in.workDir,
		LogFile:    filepath.Join(in.workDir, "logs", "seed.log"),
		DBPath:     filepath.Join(in.workDir, "data", "seed.db"),
		CertFile:   filepath.Join(in.workDir, "certs", "server.crt"),
		KeyFile:    filepath.Join(in.workDir, "certs", "server.key"),
		Port:       l.Port,
	}
	if l != want {
		t.Errorf("layout = %+v\nwant     %+v", l, want)
	}
}
