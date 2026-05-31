package api

// JSON response field names used across multiple handlers. Declared
// as constants to satisfy goconst (the linter flags strings repeated
// across 10+ sites) without inventing dependencies between handlers.
const (
	jsonKeyCount   = "count"
	jsonKeyName    = "name"
	jsonKeyEnabled = "enabled"
)
