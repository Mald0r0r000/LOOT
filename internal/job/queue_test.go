package job

import (
	"loot/internal/config"
	"testing"
)

func TestQueue_Add(t *testing.T) {
	q := NewQueue()

	cfg := config.DefaultConfig()
	j := NewJob(cfg)

	q.Add(j)

	if len(q.Pending) != 1 {
		t.Errorf("Queue should have 1 pending job, got %d", len(q.Pending))
	}

	if q.Pending[0] != j {
		t.Error("Job mismatch in pending queue")
	}
}

func TestQueue_Snapshot(t *testing.T) {
	q := NewQueue()
	cfg := config.DefaultConfig()
	j1 := NewJob(cfg)
	j2 := NewJob(cfg)

	q.Add(j1)
	q.Add(j2)

	active, pending, completed, failed := q.Snapshot()

	if active != nil {
		t.Error("Active should be nil")
	}
	if len(pending) != 2 {
		t.Errorf("Pending should be 2, got %d", len(pending))
	}
	if len(completed) != 0 {
		t.Error("Completed should be 0")
	}
	if len(failed) != 0 {
		t.Error("Failed should be 0")
	}
}

func TestQueue_Cancel_Pending(t *testing.T) {
	q := NewQueue()
	cfg := config.DefaultConfig()
	j := NewJob(cfg)

	q.Add(j)

	q.CancelJob(j.ID)

	if len(q.Pending) != 0 {
		t.Errorf("Pending should be 0 after cancel, got %d", len(q.Pending))
	}

	if len(q.Failed) != 1 {
		t.Errorf("Failed should be 1 after cancel, got %d", len(q.Failed))
	}

	if q.Failed[0] != j {
		t.Error("Cancelled job missing from Failed list")
	}
}
