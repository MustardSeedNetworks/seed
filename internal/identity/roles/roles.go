// Package roles names the user roles (admin, operator, viewer). It is a leaf
// so the database that stores a role and the API that gates on one share the
// vocabulary without the API importing persistence (seed#2750, ADR-0020). The
// database CHECK constraint enforces the same set.
package roles

// The role names, lowest privilege last.
const (
	Admin    = "admin"
	Operator = "operator"
	Viewer   = "viewer"
)

// IsValid reports whether r is one of admin, operator or viewer.
func IsValid(r string) bool {
	return r == Admin || r == Operator || r == Viewer
}
