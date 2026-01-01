package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

var exportCmd = &cobra.Command{
	Use:   "export-config",
	Short: "Export configuration",
	Long:  "Export configuration with secrets redacted (safe for sharing)",
	Run:   runExport,
}

const redactedValue = "[REDACTED]"

func init() {
	exportCmd.Flags().StringP("output", "o", "-", "Output file (- for stdout)")
	exportCmd.Flags().StringP("format", "f", "yaml", "Output format (yaml or json)")
	exportCmd.Flags().Bool("no-redact", false, "Do not redact secrets (DANGEROUS)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, _ []string) {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting output flag: %v\n", err)
		os.Exit(1)
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting format flag: %v\n", err)
		os.Exit(1)
	}
	noRedact, err := cmd.Flags().GetBool("no-redact")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting no-redact flag: %v\n", err)
		os.Exit(1)
	}

	// Resolve config path
	configPath := paths.ResolveConfigPath(cfgFile, paths.ModeAuto)

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

	// Marshal to requested format
	var data []byte
	switch format {
	case "json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	case "yaml":
		data, err = yaml.Marshal(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s (use yaml or json)\n", format)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if output == "-" {
		fmt.Println(string(data))
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
