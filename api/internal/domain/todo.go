package domain

import "time"

type Status string

const (
	NotStarted Status = "not_started"
	InProgress Status = "in_progress"
	Completed  Status = "completed"
	Archived   Status = "archived"
)

type Priority string

const (
	Low    Priority = "low"
	Medium Priority = "medium"
	High   Priority = "high"
)

type Todo struct {
	ID          int64      `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	Description string     `db:"description" json:"description"`
	DueDate     *time.Time `db:"due_date" json:"dueDate"`
	Status      Status     `db:"status" json:"status"`
	Priority    Priority   `db:"priority" json:"priority"`

	RecurUnit  *RecurUnit `db:"recur_unit" json:"recurUnit"`
	RecurEvery *int       `db:"recur_interval" json:"recurInterval"`
	// Not exposed. It is how the schedule is computed, not something the client
	// sets or shows, and sending it invites a caller to re-anchor by hand.
	RecurAnchor *time.Time `db:"recur_anchor" json:"-"`

	Version   int       `db:"version" json:"version"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

func (s Status) Valid() bool {
	switch s {
	case NotStarted, InProgress, Completed, Archived:
		return true
	}
	return false
}

func (p Priority) Valid() bool {
	switch p {
	case Low, Medium, High:
		return true
	}
	return false
}
