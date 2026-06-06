package anomaly

import (
	"fmt"
	"sort"
)

// Catalog is an immutable, validated set of Def entries keyed by ID. A
// rule source registers its definitions once; the engine rejects detections that
// reference an unknown ID, so a rule can never emit an undefined anomaly.
type Catalog struct {
	defs map[string]Def
}

// NewCatalog validates and indexes defs. It errors on a duplicate ID, a missing
// required field (id/title/category/defaultSeverity/description/recommendation),
// an unknown category or severity, or a malformed follow-up — so a typo in the
// data-driven catalog fails fast at startup rather than shipping a blank card.
func NewCatalog(defs ...Def) (*Catalog, error) {
	c := &Catalog{defs: make(map[string]Def, len(defs))}
	for _, d := range defs {
		if err := validateDef(d); err != nil {
			return nil, err
		}
		if _, dup := c.defs[d.ID]; dup {
			return nil, fmt.Errorf("anomaly: duplicate catalog id %q", d.ID)
		}
		c.defs[d.ID] = d
	}
	return c, nil
}

func validateDef(d Def) error {
	switch {
	case d.ID == "":
		return fmt.Errorf("anomaly: catalog entry missing id (title %q)", d.Title)
	case d.Title == "":
		return fmt.Errorf("anomaly: catalog entry %q missing title", d.ID)
	case d.Description == "":
		return fmt.Errorf("anomaly: catalog entry %q missing description", d.ID)
	case d.Recommendation == "":
		return fmt.Errorf("anomaly: catalog entry %q missing recommendation", d.ID)
	case !validCategory(d.Category):
		return fmt.Errorf("anomaly: catalog entry %q has unknown category %q", d.ID, d.Category)
	case !d.DefaultSeverity.valid():
		return fmt.Errorf(
			"anomaly: catalog entry %q has invalid defaultSeverity %q",
			d.ID,
			d.DefaultSeverity,
		)
	}
	for i, f := range d.FollowUps {
		if f.Kind != FollowUpAuto && f.Kind != FollowUpPrompt {
			return fmt.Errorf(
				"anomaly: catalog entry %q follow-up %d has invalid kind %q",
				d.ID,
				i,
				f.Kind,
			)
		}
		if f.Label == "" {
			return fmt.Errorf("anomaly: catalog entry %q follow-up %d missing label", d.ID, i)
		}
	}
	return nil
}

func validCategory(c Category) bool {
	switch c {
	case CategorySecurity, CategoryRF, CategoryRoaming, CategoryCapacity,
		CategoryStandards, CategoryAuthorization, CategoryNetHealth:
		return true
	default:
		return false
	}
}

// Lookup returns the definition for id.
func (c *Catalog) Lookup(id string) (Def, bool) {
	d, ok := c.defs[id]
	return d, ok
}

// Defs returns all definitions sorted by ID (deterministic for tests/audit).
func (c *Catalog) Defs() []Def {
	out := make([]Def, 0, len(c.defs))
	for _, d := range c.defs {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Len is the number of catalog entries.
func (c *Catalog) Len() int { return len(c.defs) }
