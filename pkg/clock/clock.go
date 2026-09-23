package clock

import (
	"time"
)

type Clock interface{ Now() time.Time }
type System struct{}

// Fixed is a deterministic clock for tests and replayable jobs.
type Fixed struct{ Time time.Time }

func (System) Now() time.Time  { return time.Now().UTC() }
func (f Fixed) Now() time.Time { return f.Time.UTC() }
func New() Clock               { return System{} }
