package job

import (
	"sync"
	"time"
)

// QueueState represents the state of the queue for UI consumption
type QueueState struct {
	Pending   int
	Completed int
	Failed    int
	Total     int
	ActiveID  string
}

type Queue struct {
	Pending   []*Job
	Active    *Job
	Completed []*Job
	Failed    []*Job

	UpdateChan chan QueueState

	mutex sync.Mutex
	quit  chan struct{}
}

func NewQueue() *Queue {
	return &Queue{
		Pending:    make([]*Job, 0),
		Completed:  make([]*Job, 0),
		Failed:     make([]*Job, 0),
		UpdateChan: make(chan QueueState, 10),
		quit:       make(chan struct{}),
	}
}

func (q *Queue) Add(j *Job) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.Pending = append(q.Pending, j)
	q.broadcastState()
}

// Start begins processing the queue.
// jobUpdates is the channel where running jobs will send their progress Msg.
func (q *Queue) Start(jobUpdates chan Msg) {
	go q.process(jobUpdates)
}

func (q *Queue) Stop() {
	close(q.quit)
}

func (q *Queue) process(jobUpdates chan Msg) {
	// Ticker to check for new jobs if nothing pushes (debounce)
	// or just loop?
	// Simple loop with small sleep or cond wait would be better but ticker is easy
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-q.quit:
			return
		case <-ticker.C:
			q.mutex.Lock()
			if q.Active == nil && len(q.Pending) > 0 {
				// Pick next job
				nextJob := q.Pending[0]
				q.Pending = q.Pending[1:]
				q.Active = nextJob

				// Unlock before running to allow Add()
				q.mutex.Unlock()

				// Broadcast start (Active changed)
				q.broadcastState()

				// Run the job (blocking)
				// We pass the updates channel so UI gets messages
				nextJob.Run(jobUpdates)

				// Job finished
				q.mutex.Lock()
				q.Active = nil
				if nextJob.Status == StatusFailed {
					q.Failed = append(q.Failed, nextJob)
				} else {
					q.Completed = append(q.Completed, nextJob)
				}
				q.mutex.Unlock()
				q.broadcastState()
			} else {
				q.mutex.Unlock()
			}
		}
	}
}

func (q *Queue) CancelJob(id string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Check Active
	if q.Active != nil && q.Active.ID == id {
		q.Active.Cancel()
		// Processor loop will handle the completion/failure when Run() returns
		return
	}

	// Check Pending
	for i, j := range q.Pending {
		if j.ID == id {
			j.Cancel()
			// Remove from pending? Or move to failed/cancelled?
			// Let's move to Failed (or Cancelled if we had a list) history immediately
			q.Pending = append(q.Pending[:i], q.Pending[i+1:]...)
			q.Failed = append(q.Failed, j) // Using Failed list for now
			q.broadcastState()
			return
		}
	}
}

// Snapshot returns a safe copy of the current queue state
func (q *Queue) Snapshot() (active *Job, pending, completed, failed []*Job) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Active is a pointer, but Job struct is not thread-safe for reading all fields while running?
	// Job fields like Status/Progress are updated in Run() which is in another goroutine.
	// But mostly we just read ID/Status.
	active = q.Active

	// Copy slices to avoid concurrent modification issues
	if len(q.Pending) > 0 {
		pending = make([]*Job, len(q.Pending))
		copy(pending, q.Pending)
	}
	if len(q.Completed) > 0 {
		completed = make([]*Job, len(q.Completed))
		copy(completed, q.Completed)
	}
	if len(q.Failed) > 0 {
		failed = make([]*Job, len(q.Failed))
		copy(failed, q.Failed)
	}
	return
}

func (q *Queue) broadcastState() {
	// Calculate snapshot
	params := QueueState{
		Pending:   len(q.Pending),
		Completed: len(q.Completed),
		Failed:    len(q.Failed),
	}
	if q.Active != nil {
		params.ActiveID = q.Active.ID
	}
	params.Total = params.Pending + params.Completed + params.Failed
	if q.Active != nil {
		params.Total++
	}

	// Non-blocking send
	select {
	case q.UpdateChan <- params:
	default:
	}
}
