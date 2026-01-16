package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

const redactedValue = "[REDACTED]"

func initExportCmd(state *cliState) {
	exportCmd := &cobra.Command{
		Use:   "export-config",
		Short: "Export configuration",
		Long:  "Export configuration as JSON with secrets redacted (safe for sharing)",
		Run: func(cmd *cobra.Command, args []string) {
			runExport(cmd, args, state)
		},
	}
	exportCmd.Flags().StringP("output", "o", "-", "Output file (- for stdout)")
	exportCmd.Flags().Bool("no-redact", false, "Do not redact secrets (DANGEROUS)")
	state.rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, _ []string, state *cliState) {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting output flag: %v\n", err)
		os.Exit(1)
	}
	noRedact, err := cmd.Flags().GetBool("no-redact")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting no-redact flag: %v\n", err)
		os.Exit(1)
	}

	// Resolve config path
	configPath := paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto)

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Redact secrets unless --no-redact
	if !noRedact {
		cfg = redactSecrets(cfg)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if output == "-" {
		fmt.Fprintln(os.Stdout, string(data))
	} else {
		err = os.WriteFile(output, data, 0o600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Config exported to: %s\n", output)
	}
}

func redactSecrets(cfg *config.Config) *config.Config {
	// Make a shallow copy of the struct (excluding fields with mutexes)
	redacted := config.Config{
		Version:          cfg.Version,
		Server:           cfg.Server,
		Interface:        cfg.Interface,
		VLAN:             cfg.VLAN,
		IP:               cfg.IP,
		Discovery:        cfg.Discovery,
		NetworkDiscovery: cfg.NetworkDiscovery,
		DNS:              cfg.DNS,
		HealthChecks:     cfg.HealthChecks,
		Speedtest:        cfg.Speedtest,
		Iperf:            cfg.Iperf,
		Thresholds:       cfg.Thresholds,
		Auth:             cfg.Auth,
		Security:         cfg.Security,
		DHCP:             cfg.DHCP,
		SNMP:             cfg.SNMP,
		FABOptions:       cfg.FABOptions,
		DisplayOptions:   cfg.DisplayOptions,
		Logging:          cfg.Logging,
	}

	// Redact auth secrets
	redacted.Auth.DefaultPasswordHash = redactedValue
	redacted.Auth.JWTSecret = redactedValue

	// Redact vulnerability scanning API key
	redacted.Security.VulnerabilityScanning.NVDAPIKey = redactedValue

	// Redact SNMP credentials
	newCreds := make([]config.SNMPv3Credential, len(redacted.SNMP.V3Credentials))
	for i := range redacted.SNMP.V3Credentials {
		newCreds[i] = redacted.SNMP.V3Credentials[i]
		newCreds[i].AuthPassword = redactedValue
		newCreds[i].PrivPassword = redactedValue
	}
	redacted.SNMP.V3Credentials = newCreds

	return &redacted
}
