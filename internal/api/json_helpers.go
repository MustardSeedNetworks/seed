package api

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/krisarmstrong/seed/internal/logging"
)

// SafeMarshal marshals v to JSON, logging errors if they occur.
// Returns the marshaled bytes and any error encountered.
// Use this when you need to handle the error gracefully.
//
// Example:
//
//	data, err := SafeMarshal(logger, myStruct)
//	if err != nil {
//	    return fmt.Errorf("failed to serialize response: %w", err)
//	}
func SafeMarshal(logger *slog.Logger, v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		effectiveLogger := logger
		if effectiveLogger == nil {
			effectiveLogger = logging.GetLogger()
		}
		effectiveLogger.Error("JSON marshal failed",
			"error", err,
			"type", fmt.Sprintf("%T", v),
		)
		return nil, fmt.Errorf("json marshal: %w", err)
	}
	return data, nil
}

// SafeUnmarshal unmarshals data into v, logging errors if they occur.
// Returns any error encountered during unmarshaling.
// Use this when you need to handle the error gracefully and want
// consistent error logging across the codebase.
//
// Example:
//
//	var config MyConfig
//	if err := SafeUnmarshal(logger, data, &config); err != nil {
//	    return fmt.Errorf("invalid configuration: %w", err)
//	}
func SafeUnmarshal(logger *slog.Logger, data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		effectiveLogger := logger
		if effectiveLogger == nil {
			effectiveLogger = logging.GetLogger()
		}
		effectiveLogger.Error("JSON unmarshal failed",
			"error", err,
			"type", fmt.Sprintf("%T", v),
			"data_length", len(data),
		)
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

// MustMarshal marshals v to JSON, panicking on error.
// Use this ONLY for static data that is guaranteed to marshal successfully,
// such as configuration defaults or compile-time constants.
//
// WARNING: Do not use for user input or dynamic data.
//
// Example:
//
//	var defaultConfig = MustMarshal(map[string]string{
//	    "version": "1.0",
//	    "name":    "seed",
//	})
func MustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("MustMarshal: failed to marshal %T: %v", v, err))
	}
	return data
}
