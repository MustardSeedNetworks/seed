package main

import (
	"slices"
	"testing"
)

func TestCLIStateStruct(t *testing.T) {
	t.Parallel()

	state := &cliState{
		cfgFile:        "/etc/seed/config.json",
		devMode:        true,
		trustedProxies: "10.0.0.0/8,172.16.0.0/12",
	}

	if state.cfgFile != "/etc/seed/config.json" {
		t.Errorf("cfgFile should be '/etc/seed/config.json', got %q", state.cfgFile)
	}
	if !state.devMode {
		t.Error("devMode should be true")
	}
	if state.trustedProxies != "10.0.0.0/8,172.16.0.0/12" {
		t.Errorf("trustedProxies should be '10.0.0.0/8,172.16.0.0/12', got %q", state.trustedProxies)
	}
}

func TestCLIStateDefaults(t *testing.T) {
	t.Parallel()

	state := &cliState{}

	if state.cfgFile != "" {
		t.Errorf("cfgFile should default to empty string, got %q", state.cfgFile)
	}
	if state.devMode {
		t.Error("devMode should default to false")
	}
	if state.trustedProxies != "" {
		t.Errorf("trustedProxies should default to empty string, got %q", state.trustedProxies)
	}
	if state.rootCmd != nil {
		t.Error("rootCmd should default to nil")
	}
	if state.completionCmd != nil {
		t.Error("completionCmd should default to nil")
	}
}

func TestNewCLIState(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	if state == nil {
		t.Fatal("newCLIState() returned nil")
	}

	// Verify rootCmd is created
	if state.rootCmd == nil {
		t.Error("rootCmd should not be nil")
	}

	// Verify completionCmd is created
	if state.completionCmd == nil {
		t.Error("completionCmd should not be nil")
	}

	// Verify rootCmd properties
	if state.rootCmd.Use != "seed" {
		t.Errorf("rootCmd.Use should be 'seed', got %q", state.rootCmd.Use)
	}

	// Verify completionCmd properties
	if state.completionCmd.Use != "completion [bash|zsh|fish|powershell]" {
		t.Errorf("completionCmd.Use mismatch: got %q", state.completionCmd.Use)
	}
}

func TestNewCLIStateRootCmdShortDescription(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	if state.rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
}

func TestNewCLIStateLongDescription(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	if state.rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}

	// Verify the long description mentions key features
	features := []string{
		"Network device discovery",
		"WiFi site surveys",
		"Speed testing",
		"DHCP rogue detection",
		"Vulnerability scanning",
	}

	for _, feature := range features {
		if !containsSubstring(state.rootCmd.Long, feature) {
			t.Errorf("rootCmd.Long should mention %q", feature)
		}
	}
}

func TestNewCLIStateCompletionValidArgs(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	expectedShells := []string{"bash", "zsh", "fish", "powershell"}

	if len(state.completionCmd.ValidArgs) != len(expectedShells) {
		t.Errorf("Expected %d valid args, got %d", len(expectedShells), len(state.completionCmd.ValidArgs))
	}

	for _, shell := range expectedShells {
		if !slices.Contains(state.completionCmd.ValidArgs, shell) {
			t.Errorf("completionCmd.ValidArgs should contain %q", shell)
		}
	}
}

func TestInitCommands(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Verify version is set
	if state.rootCmd.Version == "" {
		t.Error("rootCmd.Version should be set after initCommands")
	}

	// Verify subcommands are added
	expectedCommands := []string{
		"completion",
		"credentials",
		"export-config",
		"install",
		"mcp",
		"reset-config",
		"serve",
		"setup-wizard",
		"uninstall",
		"validate-config",
		"version",
	}

	commands := state.rootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Use] = true
	}

	for _, expected := range expectedCommands {
		found := false
		for name := range commandNames {
			// Check if the command name starts with the expected name
			// (because Use might include arguments like "completion [bash|zsh|fish|powershell]")
			if len(name) >= len(expected) && name[:len(expected)] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command %q to be registered", expected)
		}
	}
}

func TestInitCommandsAddsFlags(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Check persistent flags
	configFlag := state.rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected --config flag to be defined")
	}

	devFlag := state.rootCmd.PersistentFlags().Lookup("dev")
	if devFlag == nil {
		t.Error("Expected --dev flag to be defined")
	}

	proxyFlag := state.rootCmd.PersistentFlags().Lookup("trusted-proxies")
	if proxyFlag == nil {
		t.Error("Expected --trusted-proxies flag to be defined")
	}
}

func TestCLIStateTrustedProxiesVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  string
		expect string
	}{
		{
			name:   "single IP",
			value:  "192.168.1.1",
			expect: "192.168.1.1",
		},
		{
			name:   "single CIDR",
			value:  "10.0.0.0/8",
			expect: "10.0.0.0/8",
		},
		{
			name:   "multiple CIDRs",
			value:  "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
			expect: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
		},
		{
			name:   "empty",
			value:  "",
			expect: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := &cliState{
				trustedProxies: tc.value,
			}

			if state.trustedProxies != tc.expect {
				t.Errorf("trustedProxies: got %q, want %q", state.trustedProxies, tc.expect)
			}
		})
	}
}
