package reporting

import "errors"

// ErrNotImplemented is returned by stub functions pending migration.
var ErrNotImplemented = errors.New("not implemented: pending migration")

// ErrScheduleNotFound is returned when no scheduled report has the given ID.
var ErrScheduleNotFound = errors.New("scheduled report not found")

// ErrInvalidSchedule wraps every reason a scheduled report is rejected; the
// wrapped message names the field.
var ErrInvalidSchedule = errors.New("invalid scheduled report")
