package management

import (
	"fmt"
	"time"
)

// extractDurationField extracts a duration field from a timing object.
// Returns (duration, true, nil) if found, (nil, false, nil) if not found,
// or (nil, false, error) if the field has invalid type.
func extractDurationField(obj map[string]any, field, parentKey string) (*time.Duration, bool, error) {
	val, exists := obj[field]
	if !exists {
		return nil, false, nil
	}

	num, ok := val.(float64)
	if !ok {
		return nil, false, fmt.Errorf("thresholds.httpTimings.%s.%s must be a number", parentKey, field)
	}

	d := time.Duration(num) * time.Millisecond
	return &d, true, nil
}

// extractBool extracts a boolean value from a map, returning an error if the key exists but is not a bool.
func extractBool(data map[string]any, key, prefix string) (bool, bool, error) {
	val, exists := data[key]
	if !exists {
		return false, false, nil
	}
	b, ok := val.(bool)
	if !ok {
		return false, false, fmt.Errorf("%s.%s must be a boolean", prefix, key)
	}
	return b, true, nil
}

// extractString extracts a string value from a map, returning an error if the key exists but is not a string.
func extractString(data map[string]any, key, prefix string) (string, bool, error) {
	val, exists := data[key]
	if !exists {
		return "", false, nil
	}
	s, ok := val.(string)
	if !ok {
		return "", false, fmt.Errorf("%s.%s must be a string", prefix, key)
	}
	return s, true, nil
}

// extractInt extracts an integer from a float64 value in a map.
func extractInt(data map[string]any, key, prefix string) (int, bool, error) {
	val, exists := data[key]
	if !exists {
		return 0, false, nil
	}
	f, ok := val.(float64)
	if !ok {
		return 0, false, fmt.Errorf("%s.%s must be a number", prefix, key)
	}
	return int(f), true, nil
}
