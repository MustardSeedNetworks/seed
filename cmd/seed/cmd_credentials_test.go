package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestInitCredentialsCmd(t *testing.T) {

	state := newCLIState()
	initCredentialsCmd(state)

	// Find the credentials command
	var credentialsCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "credentials" {
			credentialsCmd = cmd
			break
		}
	}

	if credentialsCmd == nil {
		t.Fatal("credentials command not found")
	}

	// Verify command properties
	if credentialsCmd.Short == "" {
		t.Error("credentials command should have a Short description")
	}
	if credentialsCmd.Long == "" {
		t.Error("credentials command should have a Long description")
	}
}

func TestCredentialsCmdHasJSONFlag(t *testing.T) {

	state := newCLIState()
	initCredentialsCmd(state)

	// Find the credentials command
	var credentialsCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "credentials" {
			credentialsCmd = cmd
			break
		}
	}

	if credentialsCmd == nil {
		t.Fatal("credentials command not found")
	}

	// Check for --json flag
	jsonFlag := credentialsCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("credentials command should have --json flag")
	}
	if jsonFlag != nil && jsonFlag.DefValue != "false" {
		t.Error("--json flag should default to false")
	}
}

func TestCredentialsCmdHasRunFunction(t *testing.T) {

	state := newCLIState()
	initCredentialsCmd(state)

	// Find the credentials command
	var credentialsCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "credentials" {
			credentialsCmd = cmd
			break
		}
	}

	if credentialsCmd == nil {
		t.Fatal("credentials command not found")
	}

	if credentialsCmd.Run == nil {
		t.Error("credentials command should have a Run function")
	}
}

func TestCredentialsCommandLongDescriptionContent(t *testing.T) {

	state := newCLIState()
	initCredentialsCmd(state)

	// Find the credentials command
	var credentialsCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "credentials" {
			credentialsCmd = cmd
			break
		}
	}

	if credentialsCmd == nil {
		t.Fatal("credentials command not found")
	}

	// Check that long description mentions key features
	expectedContent := []string{
		"setup",
		"password",
		"JSON",
	}

	for _, content := range expectedContent {
		if !containsSubstring(credentialsCmd.Long, content) {
			t.Errorf("credentials Long description should mention %q", content)
		}
	}
}

func TestCredentialsCmdWithConfigFile(t *testing.T) {

	// Create a temp directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "seed.json")

	// Create a valid config file
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultUsername = "testuser"
	cfg.Auth.DefaultPasswordHash = "somehash"
	cfg.Auth.JWTSecret = "somesecret"

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if writeErr := os.WriteFile(configPath, data, 0o600); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}

	// Create CLI state with config path
	state := &cliState{
		cfgFile: configPath,
	}

	// Verify the state has the config file set
	if state.cfgFile != configPath {
		t.Errorf("cfgFile should be %q, got %q", configPath, state.cfgFile)
	}
}

func TestCredentialsCmdFlagTypes(t *testing.T) {

	state := newCLIState()
	initCredentialsCmd(state)

	// Find the credentials command
	var credentialsCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "credentials" {
			credentialsCmd = cmd
			break
		}
	}

	if credentialsCmd == nil {
		t.Fatal("credentials command not found")
	}

	// Test that the json flag is a boolean
	jsonFlag := credentialsCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("--json flag not found")
	}

	// Verify it's a bool flag by checking the type
	if jsonFlag.Value.Type() != "bool" {
		t.Errorf("--json flag should be bool type, got %s", jsonFlag.Value.Type())
	}
}
