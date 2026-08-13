package domain

// CanTransition decides whether a task may move from one status to another.
//
// Completed is guarded as well as In Progress. The brief only names In
// Progress, but a gate you can walk around by skipping a step is not a gate:
// leaving Completed open would let anyone finish a blocked task by choosing the
// further destination.
func CanTransition(from, to Status, unmetDeps int, blockers []Blocker, id int64) error {
	if !to.Valid() {
		return Invalid("status", "Status must be one of not_started, in_progress, completed, archived")
	}
	if from == to {
		return nil
	}

	// Archiving is always allowed. Shelving a task you cannot start is exactly
	// what you want to do with it.
	if to == Archived {
		return nil
	}
	// Archived means shelved, not finished, so it comes back to Not started
	// rather than straight to done.
	if from == Archived && to != NotStarted {
		return Invalid("status", "Unarchive this task to not_started before changing it further")
	}

	if (to == InProgress || to == Completed) && unmetDeps > 0 {
		return &BlockedError{Target: id, Blockers: blockers}
	}
	return nil
}
