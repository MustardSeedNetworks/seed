// Package config handles application configuration with JSON Schema validation.
package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

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
	var schemaDoc any
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
		return []ValidationError{
			{Path: "", Message: fmt.Sprintf("failed to marshal config: %v", err)},
		}
	}

	// Then unmarshal the YAML as JSON-compatible data
	// YAML is a superset of JSON, so this works
	var data any
	if unmarshalErr := json.Unmarshal(yamlBytes, &data); unmarshalErr != nil {
		return []ValidationError{
			{Path: "", Message: fmt.Sprintf("failed to unmarshal config: %v", unmarshalErr)},
		}
	}

	// Validate
	err = v.schema.Validate(data)
	if err == nil {
		return nil
	}

	// Extract validation errors
	var validationErrs []ValidationError
	// Parse the validation error to extract path and message
	// The jsonschema library returns structured errors
	var validationErr *jsonschema.ValidationError
	if errors.As(err, &validationErr) {
		validationErrs = extractValidationErrors(validationErr)
	} else {
		validationErrs = []ValidationError{{Path: "", Message: err.Error()}}
	}

	return validationErrs
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
	var yamlData any
	if unmarshalErr := yaml.Unmarshal(yamlBytes, &yamlData); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", unmarshalErr)
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

// schemaValidatorState holds the lazy-initialized schema validator to satisfy gochecknoglobals.
type schemaValidatorState struct {
	once      sync.Once
	validator *SchemaValidator
	err       error
}

// validatorState provides thread-safe lazy initialization of the schema validator.
var validatorState = func() *schemaValidatorState {
	return &schemaValidatorState{}
}()

// ValidateWithSchema validates a Config against the embedded JSON schema.
// Returns nil if valid, otherwise returns validation errors.
// This is a convenience function that uses a lazily-initialized validator instance.
func ValidateWithSchema(cfg *Config) []ValidationError {
	validatorState.once.Do(func() {
		validatorState.validator, validatorState.err = NewSchemaValidator()
	})

	if validatorState.err != nil {
		// Schema validation unavailable, return nil to fall back to struct validation
		return nil
	}

	return validatorState.validator.ValidateConfig(cfg)
}
