// Package devicegroups is device grouping (P-D10): a client's named set of
// polling targets. An alert rule that names a group fires only for events
// from the group's members.
//
// A member is a polling target id, because the polling target is the device
// Seed monitors: it is what the collectors poll and what a syslog or trap
// source address resolves to. A group never holds another client's target.
package devicegroups

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Field limits. A name is a label in a picker; a description is a sentence.
const (
	maxNameLen        = 64
	maxDescriptionLen = 512
)

var (
	// ErrNotFound is returned when the client has no group with the id.
	ErrNotFound = errors.New("device group not found")
	// ErrNameTaken is returned when the client already has a group of that name.
	ErrNameTaken = errors.New("a device group with this name already exists")
	// ErrInUse is returned when deleting a group an alert rule names.
	ErrInUse = errors.New("the device group is used by an alert rule")
	// ErrUnknownMember is returned when a member is not one of the client's
	// polling targets.
	ErrUnknownMember = errors.New("a member is not a polling target")
	// ErrUnavailable is returned when no store is wired (handler → 503).
	ErrUnavailable = errors.New("device groups unavailable")
)

// ValidationError carries a user-input message the handler returns verbatim.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Group is one device group. Members are polling target ids, sorted.
type Group struct {
	ID          string
	ClientID    string
	Name        string
	Description string
	Members     []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Store is the persistence the use-case needs. Every call is scoped to the
// client, so another client's group is indistinguishable from a missing one.
type Store interface {
	List(ctx context.Context, clientID string) ([]Group, error)
	Get(ctx context.Context, clientID, id string) (Group, error)
	// Save creates g when g.ID is blank, generating the id, and otherwise
	// replaces the group's name, description and members.
	Save(ctx context.Context, g *Group) error
	Delete(ctx context.Context, clientID, id string) error
}

// Service is the device-group use-case.
type Service struct {
	store Store
}

// NewService builds the use-case over its Store port.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns the client's groups ordered by name.
func (s *Service) List(ctx context.Context, clientID string) ([]Group, error) {
	return s.store.List(ctx, clientID)
}

// Get returns one group.
func (s *Service) Get(ctx context.Context, clientID, id string) (Group, error) {
	return s.store.Get(ctx, clientID, id)
}

// Create stores a new group and returns it as stored.
func (s *Service) Create(ctx context.Context, g Group) (Group, error) {
	g.ID = ""
	return s.save(ctx, g)
}

// Update replaces the group at g.ID and returns it as stored.
func (s *Service) Update(ctx context.Context, g Group) (Group, error) {
	return s.save(ctx, g)
}

// Delete removes a group. A group an alert rule names is ErrInUse.
func (s *Service) Delete(ctx context.Context, clientID, id string) error {
	return s.store.Delete(ctx, clientID, id)
}

func (s *Service) save(ctx context.Context, g Group) (Group, error) {
	if err := normalize(&g); err != nil {
		return Group{}, err
	}
	if err := s.store.Save(ctx, &g); err != nil {
		return Group{}, err
	}
	return s.store.Get(ctx, g.ClientID, g.ID)
}

// normalize trims the text fields, checks their lengths and sorts and dedupes
// the members.
func normalize(g *Group) error {
	g.Name = strings.TrimSpace(g.Name)
	g.Description = strings.TrimSpace(g.Description)
	switch {
	case g.Name == "":
		return &ValidationError{Msg: "name is required"}
	case utf8.RuneCountInString(g.Name) > maxNameLen:
		return &ValidationError{Msg: "name must be at most 64 characters"}
	case utf8.RuneCountInString(g.Description) > maxDescriptionLen:
		return &ValidationError{Msg: "description must be at most 512 characters"}
	}
	members := make([]string, 0, len(g.Members))
	for _, m := range g.Members {
		m = strings.TrimSpace(m)
		if m == "" {
			return &ValidationError{Msg: "a member id is empty"}
		}
		members = append(members, m)
	}
	slices.Sort(members)
	g.Members = slices.Compact(members)
	return nil
}
