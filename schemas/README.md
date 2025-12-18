# Configuration Schemas

This directory contains JSON Schema definitions for The Seed configuration files.

## config.schema.json

JSON Schema (Draft 2020-12) for The Seed configuration file (`config.yaml`).

### Usage

#### Validation with VS Code

Add to `.vscode/settings.json`:

````json
{
  "yaml.schemas": {
    "./schemas/config.schema.json": "config.yaml"
  }
}
```bash

#### Validation with command-line tools

```bash
# Using ajv-cli
npm install -g ajv-cli
ajv validate -s schemas/config.schema.json -d config.yaml

# Using check-jsonschema
pip install check-jsonschema
check-jsonschema --schemafile schemas/config.schema.json config.yaml
```python

### Schema Details

- **Version:** JSON Schema Draft 2020-12
- **Source:** Generated from `internal/config/config.go`
- **Coverage:** All configuration sections with validation rules from `config.Validate()`

### Key Features

- **Type Safety:** Strict type checking for all configuration values
- **Validation Rules:** Port ranges, VLAN IDs, duration formats, enum values
- **Default Values:** Includes default values from `DefaultConfig()`
- **Documentation:** Descriptions for all fields
- **Format Validation:** Email, hostname, IPv4, URI formats

### Reusable Definitions

The schema defines reusable types in `$defs`:

- `Duration`: Go duration string pattern (e.g., "5s", "100ms", "1h")
- `PortNumber`: Valid TCP/UDP port (1-65535)
- `Threshold`: Warning/critical duration thresholds
- `IntThreshold`: Warning/critical integer thresholds
- `SignalThreshold`: Signal strength thresholds in dBm (-100 to 0)

### Validation Constraints

The schema enforces the same validation rules as `config.Validate()`:

- `server.port`: 1-65535
- `server.http_redirect_port`: 0-65535
- `vlan.id`: 1-4094 (when enabled)
- `ip.mode`: "dhcp" or "static"
- `network_discovery.arp_scan_workers`: 1-500
- `network_discovery.profile`: "stealth", "standard", "full_scan", "custom"
- `snmp.port`: 1-65535
- `snmp.retries`: 0-10
- `logging.level`: "debug", "info", "warn", "warning", "error"
- `logging.format`: "text" or "json"

### Updating the Schema

When making changes to `internal/config/config.go`:

1. Update this schema to match
2. Increment `ConfigVersion` if breaking changes
3. Test with real config files
4. Document any new validation rules
````
