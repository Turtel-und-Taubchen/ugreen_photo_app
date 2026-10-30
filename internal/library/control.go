package library

import "database/sql"

// Control is the scheduler plus read access to the index it fills; together they are what the
// HTTP layer needs to start and report scans.
type Control struct {
	*Scheduler
	DB *sql.DB
}

// Count returns the number of indexed files.
func (c Control) Count() (int, error) { return Count(c.DB) }

// Unreadable returns up to limit files with a read error and their total.
func (c Control) Unreadable(limit int) ([]UnreadableFile, int, error) {
	return Unreadable(c.DB, limit)
}
