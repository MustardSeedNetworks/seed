package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Schema validation constants.
const (
	// errorMessageSplitParts is the number of parts to split error messages into (path: message).
	errorMessageSplitParts = 2
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
	// Marshal config to JSON
	jsonBytes, err := cfg.marshalForValidation()
	if err != nil {
		return []ValidationError{
			{Path: "", Message: fmt.Sprintf("failed to marshal config: %v", err)},
		}
	}

	// Unmarshal JSON for validation
	var data any
	if unmarshalErr := json.Unmarshal(jsonBytes, &data); unmarshalErr != nil {
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

// marshalForValidation marshals the config to JSON for schema validation.
func (cfg *Config) marshalForValidation() ([]byte, error) {
	// Lock for reading
	cfg.RLock()
	defer cfg.RUnlock()

	// Marshal directly to JSON
	jsonBytes, err := json.Marshal(cfg)
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
		parts := strings.SplitN(fullError, ": ", errorMessageSplitParts)
		message := fullError
		if len(parts) == errorMessageSplitParts {
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

// Validator state accessor functions use closure-encapsulated state for thread-safe singleton access.
// getValidatorState returns the global schema validator state instance.
// setValidatorState sets the global schema validator state instance.
// _ (clearValidatorState) resets the global schema validator state to nil (unused but required for pattern).
//
//nolint:gochecknoglobals // Intentional thread-safe singleton using closure pattern
var (
	getValidatorState, _, _ = func() (
		func() *schemaValidatorState,
		func(*schemaValidatorState),
		func(),
	) {
		var (
			mu    sync.RWMutex
			state *schemaValidatorState
		)
		// Initialize with default state
		state = &schemaValidatorState{}

		return func() *schemaValidatorState {
				mu.RLock()
				defer mu.RUnlock()
				return state
			}, func(s *schemaValidatorState) {
				mu.Lock()
				defer mu.Unlock()
				state = s
			}, func() {
				mu.Lock()
				defer mu.Unlock()
				state = nil
			}
	}()
)

// ValidateWithSchema validates a Config against the embedded JSON schema.
// Returns nil if valid, otherwise returns validation errors.
// This is a convenience function that uses a lazily-initialized validator instance.
func ValidateWithSchema(cfg *Config) []ValidationError {
	state := getValidatorState()
	state.once.Do(func() {
		state.validator, state.err = NewSchemaValidator()
	})

	if state.err != nil {
		// Schema validation unavailable, return nil to fall back to struct validation
		return nil
	}

	return state.validator.ValidateConfig(cfg)
}
