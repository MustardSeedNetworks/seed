package management

import (
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// applyHealthChecksUpdates applies health check toggle updates.
// Returns error if healthChecks key exists but has invalid type (fixes #784, G3).
func applyHealthChecksUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["healthChecks"]
	if !exists {
		return nil
	}
	healthChecks, ok := val.(map[string]any)
	if !ok {
		return errors.New("healthChecks must be an object")
	}

	if perfVal, perfExists := healthChecks["runPerformance"]; perfExists {
		runPerformance, perfOK := perfVal.(bool)
		if !perfOK {
			return errors.New("healthChecks.runPerformance must be a boolean")
		}
		cfg.HealthChecks.RunPerformance = runPerformance
	}

	if speedVal, speedExists := healthChecks["runSpeedtest"]; speedExists {
		runSpeedtest, speedOK := speedVal.(bool)
		if !speedOK {
			return errors.New("healthChecks.runSpeedtest must be a boolean")
		}
		cfg.HealthChecks.RunSpeedtest = runSpeedtest
	}

	if iperfVal, iperfExists := healthChecks["runIperf"]; iperfExists {
		runIperf, iperfOK := iperfVal.(bool)
		if !iperfOK {
			return errors.New("healthChecks.runIperf must be a boolean")
		}
		cfg.HealthChecks.RunIperf = runIperf
	}

	if discVal, discExists := healthChecks["runDiscovery"]; discExists {
		runDiscovery, discOK := discVal.(bool)
		if !discOK {
			return errors.New("healthChecks.runDiscovery must be a boolean")
		}
		cfg.HealthChecks.RunDiscovery = runDiscovery
	}

	return nil
}

// applySpeedtestUpdates applies speedtest configuration updates.
// Returns error if speedtest key exists but has invalid type (fixes #784, G3).
func applySpeedtestUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["speedtest"]
	if !exists {
		return nil
	}
	speedtest, ok := val.(map[string]any)
	if !ok {
		return errors.New("speedtest must be an object")
	}

	if serverIDVal, serverIDExists := speedtest["serverId"]; serverIDExists {
		serverID, serverIDOK := serverIDVal.(string)
		if !serverIDOK {
			return errors.New("speedtest.serverId must be a string")
		}
		cfg.Speedtest.ServerID = serverID
	}

	if autoRunVal, autoRunExists := speedtest["autoRunOnLink"]; autoRunExists {
		autoRunOnLink, autoRunOK := autoRunVal.(bool)
		if !autoRunOK {
			return errors.New("speedtest.autoRunOnLink must be a boolean")
		}
		cfg.Speedtest.AutoRunOnLink = autoRunOnLink
	}

	return nil
}

// applyIperfUpdates applies iperf configuration updates.
// Returns error if iperf key exists but has invalid type (fixes #784, G3).
func applyIperfUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["iperf"]
	if !exists {
		return nil
	}
	iperf, ok := val.(map[string]any)
	if !ok {
		return errors.New("iperf must be an object")
	}

	return applyIperfFields(iperf, cfg)
}

// applyIperfFields applies individual iperf configuration fields.
func applyIperfFields(iperf map[string]any, cfg *config.Config) error {
	const prefix = "iperf"

	if autoRun, found, err := extractBool(iperf, "autoRunOnLink", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.AutoRunOnLink = autoRun
	}

	if server, found, err := extractString(iperf, "server", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Server = server
	}

	if port, found, err := extractInt(iperf, "port", prefix); err != nil {
		return err
	} else if found {
		if validationErr := validation.ValidatePort(port); validationErr != nil {
			return fmt.Errorf("iperf.port: %w", validationErr)
		}
		cfg.Iperf.Port = port
	}

	if proto, found, err := extractString(iperf, "protocol", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Protocol = proto
	}

	if dir, found, err := extractString(iperf, "direction", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Direction = dir
	}

	if dur, found, err := extractInt(iperf, "duration", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.Duration = dur
	}

	if srvPort, found, err := extractInt(iperf, "serverPort", prefix); err != nil {
		return err
	} else if found && validation.ValidatePort(srvPort) == nil {
		cfg.Iperf.ServerPort = srvPort
	}

	if enable, found, err := extractBool(iperf, "enableServer", prefix); err != nil {
		return err
	} else if found {
		cfg.Iperf.EnableServer = enable
	}

	return nil
}

// applyFABOptionsUpdates applies FAB options updates.
// Returns error if fabOptions key exists but has invalid type (fixes #784, G3).
func applyFABOptionsUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["fabOptions"]
	if !exists {
		return nil
	}
	fabOptions, ok := val.(map[string]any)
	if !ok {
		return errors.New("fabOptions must be an object")
	}

	const prefix = "fabOptions"

	if err := applyFABRunOptions(fabOptions, prefix, cfg); err != nil {
		return err
	}

	return applyFABMiscOptions(fabOptions, prefix, cfg)
}

// applyFABRunOptions applies the "run*" boolean options for FAB.
func applyFABRunOptions(fabOptions map[string]any, prefix string, cfg *config.Config) error {
	// Define field mappings: key -> pointer to config field
	type boolField struct {
		key   string
		field *bool
	}

	fields := []boolField{
		{"runLink", &cfg.FABOptions.RunLink},
		{"runSwitch", &cfg.FABOptions.RunSwitch},
		{"runVLAN", &cfg.FABOptions.RunVLAN},
		{"runIPConfig", &cfg.FABOptions.RunIPConfig},
		{"runGateway", &cfg.FABOptions.RunGateway},
		{"runDNS", &cfg.FABOptions.RunDNS},
	}

	for _, f := range fields {
		if val, found, err := extractBool(fabOptions, f.key, prefix); err != nil {
			return err
		} else if found {
			*f.field = val
		}
	}

	return nil
}

// applyFABMiscOptions applies the remaining FAB boolean options.
func applyFABMiscOptions(fabOptions map[string]any, prefix string, cfg *config.Config) error {
	type boolField struct {
		key   string
		field *bool
	}

	fields := []boolField{
		{"runHealthChecks", &cfg.FABOptions.RunHealthChecks},
		{"runNetworkDiscovery", &cfg.FABOptions.RunNetworkDiscovery},
		{"runSpeedtest", &cfg.FABOptions.RunSpeedtest},
		{"runIperf", &cfg.FABOptions.RunIperf},
		{"runPerformance", &cfg.FABOptions.RunPerformance},
		{"autoScanOnLink", &cfg.FABOptions.AutoScanOnLink},
	}

	for _, f := range fields {
		if val, found, err := extractBool(fabOptions, f.key, prefix); err != nil {
			return err
		} else if found {
			*f.field = val
		}
	}

	return nil
}

// applyDisplayOptionsUpdates applies display options updates.
// Returns error if displayOptions key exists but has invalid type (fixes #784, G3).
func applyDisplayOptionsUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["displayOptions"]
	if !exists {
		return nil
	}
	displayOptions, ok := val.(map[string]any)
	if !ok {
		return errors.New("displayOptions must be an object")
	}

	if pubIPVal, pubIPExists := displayOptions["showPublicIP"]; pubIPExists {
		showPublicIP, pubIPOK := pubIPVal.(bool)
		if !pubIPOK {
			return errors.New("displayOptions.showPublicIP must be a boolean")
		}
		cfg.DisplayOptions.ShowPublicIP = showPublicIP
	}

	if unitVal, unitExists := displayOptions["unitSystem"]; unitExists {
		unitSystem, unitOK := unitVal.(string)
		if !unitOK {
			return errors.New("displayOptions.unitSystem must be a string")
		}
		// Validate unit system (only "sae" or "metric" allowed)
		if unitSystem == "sae" || unitSystem == "metric" {
			cfg.DisplayOptions.UnitSystem = unitSystem
		}
	}

	return nil
}
