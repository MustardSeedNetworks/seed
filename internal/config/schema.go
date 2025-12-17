// Package config handles application configuration with JSON Schema validation.
package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaData []byte

// SchemaValidator validates configuration against the JSON schema.
type SchemaValidator struct {
	schema *jsonschema.Schema
}

// NewSchemaValidator creates a new schema validator.
// It loads the embedded JSON schema and compiles it for validation.
func NewSchemaValidator() (*SchemaValidator, error) {
	// Parse the embedded schema
	var schemaDoc interface{}
	if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse embedded schema: %w", err)
	}

	// Compile schema
	c := jsonschema.NewCompiler()
	if err := c.AddResource("config.schema.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := c.Compile("config.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return &SchemaValidator{schema: schema}, nil
}

// ValidationError represents a single validation failure.
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidateConfig validates a Config struct against the JSON schema.
// It returns a slice of validation errors, or nil if the config is valid.
func (v *SchemaValidator) ValidateConfig(cfg *Config) []ValidationError {
	// First marshal to YAML (which respects the yaml tags)
	yamlBytes, err := cfg.marshalForValidation()
	if err != nil {
		return []ValidationError{{Path: "", Message: fmt.Sprintf("failed to marshal config: %v", err)}}
	}

	// Then unmarshal the YAML as JSON-compatible data
	// YAML is a superset of JSON, so this works
	var data interface{}
	if err := json.Unmarshal(yamlBytes, &data); err != nil {
		return []ValidationError{{Path: "", Message: fmt.Sprintf("failed to unmarshal config: %v", err)}}
	}

	// Validate
	err = v.schema.Validate(data)
	if err == nil {
		return nil
	}

	// Extract validation errors
	var errors []ValidationError
	// Parse the validation error to extract path and message
	// The jsonschema library returns structured errors
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		errors = extractValidationErrors(validationErr)
	} else {
		errors = []ValidationError{{Path: "", Message: err.Error()}}
	}

	return errors
}

// marshalForValidation marshals the config using YAML tags (which are compatible with JSON schema).
// We use YAML marshaling because the Config struct uses yaml tags, not json tags.
// The YAML is then converted to JSON for schema validation.
func (cfg *Config) marshalForValidation() ([]byte, error) {
	// Lock for reading
	cfg.RLock()
	defer cfg.RUnlock()

	// First marshal to YAML (respects yaml struct tags)
	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	// Convert YAML to JSON-compatible map
	var yamlData interface{}
	if err := yaml.Unmarshal(yamlBytes, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Marshal to JSON (this converts the YAML representation to JSON)
	jsonBytes, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return jsonBytes, nil
}

// extractValidationErrors recursively extracts all validation errors from the error tree.
func extractValidationErrors(err *jsonschema.ValidationError) []ValidationError {
	var errors []ValidationError

	// Get the location path (join the array into a string path)
	path := "/" + strings.Join(err.InstanceLocation, "/")

	// If there are no causes, this is a leaf error - add it
	if len(err.Causes) == 0 {
		// Use the Error() method to get the full error message
		// We extract just the message part after the path
		fullError := err.Error()
		// The error format is typically: "path: message"
		// We want just the message part
		parts := strings.SplitN(fullError, ": ", 2)
		message := fullError
		if len(parts) == 2 {
			message = parts[1]
		}

		errors = append(errors, ValidationError{
			Path:    path,
			Message: message,
		})
	}

	// Recursively get child errors
	for _, cause := range err.Causes {
		errors = append(errors, extractValidationErrors(cause)...)
	}

	return errors
}

// Global validator (lazy initialized).
var (
	globalValidator    *SchemaValidator
	globalValidatorErr error
)

// ValidateWithSchema validates a Config against the embedded JSON schema.
// Returns nil if valid, otherwise returns validation errors.
// This is a convenience function that uses a global validator instance.
func ValidateWithSchema(cfg *Config) []ValidationError {
	if globalValidator == nil && globalValidatorErr == nil {
		globalValidator, globalValidatorErr = NewSchemaValidator()
	}

	if globalValidatorErr != nil {
		// Schema validation unavailable, return nil to fall back to struct validation
		return nil
	}

	return globalValidator.ValidateConfig(cfg)
}
