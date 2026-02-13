package job

import (
	"context"
	"fmt"
	"time"

	"loot/internal/config"
	"loot/internal/mhl"
	"loot/internal/offload"
	"loot/internal/output"
	"loot/internal/report"
)

type Status string

const (
	StatusPending   Status = "Pending"
	StatusRunning   Status = "Running"
	StatusCopying   Status = "Copying"
	StatusVerifying Status = "Verifying"
	StatusCompleted Status = "Completed"
	StatusFailed    Status = "Failed"
	StatusCancelled Status = "Cancelled"
)

// Msg is sent via channel to UI
type Msg struct {
	Job        *Job
	Progress   offload.ProgressInfo
	Stage      Status
	Status     string // Human readable status
	Err        error
	Finished   bool
	JobChannel chan Msg // Chainable channel
}

type Job struct {
	ID        string
	Config    *config.Config
	Offloader *offload.Offloader
	Status    Status

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	StartTime time.Time
	EndTime   time.Time

	// Stats
	TotalBytes  int64
	CopiedBytes int64
	Speed       float64

	// Result
	Result *output.JobResult
	Err    error
}

func NewJob(cfg *config.Config) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	return &Job{
		ID:        fmt.Sprintf("job-%d", time.Now().UnixNano()), // Use Nano for better uniqueness
		Config:    cfg,
		Offloader: offload.NewOffloaderWithConfig(cfg, cfg.Source, cfg.Destination),
		Status:    StatusPending,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (j *Job) Cancel() {
	if j.cancel != nil {
		j.cancel()
		j.Status = StatusCancelled
	}
}

// Run executes the job blocking.
// In a real scenario, this would likely be run in a goroutine.
// For TUI, we might want a channel to report progress.
func (j *Job) Run(updates chan Msg) {
	j.StartTime = time.Now()
	j.Status = StatusRunning

	// Check cancellation before start
	if j.ctx.Err() != nil {
		j.fail(j.ctx.Err(), updates)
		return
	}

	// 1. COPY
	j.Status = StatusCopying
	updates <- Msg{Job: j, Stage: StatusCopying, Status: "Copying...", JobChannel: updates}

	progressCh := make(chan offload.ProgressInfo, 100)
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic: %v", r)
			}
			close(progressCh)
		}()
		errCh <- j.Offloader.Copy(j.ctx, progressCh)
	}()

	// Consume progress
loop:
	for {
		select {
		case <-j.ctx.Done():
			// Cancelled
			j.fail(j.ctx.Err(), updates)
			return
		case info, ok := <-progressCh:
			if !ok {
				break loop
			}
			j.TotalBytes = info.TotalBytes
			j.CopiedBytes = info.CopiedBytes
			j.Speed = info.Speed
			updates <- Msg{Job: j, Progress: info, Stage: StatusCopying, Status: "Copying...", JobChannel: updates}
		}
	}

	// Check copy error
	if err := <-errCh; err != nil {
		j.fail(err, updates)
		return
	}

	// Check cancellation before verify
	if j.ctx.Err() != nil {
		j.fail(j.ctx.Err(), updates)
		return
	}

	// 2. VERIFY
	if !j.Config.NoVerify {
		j.Status = StatusVerifying
		updates <- Msg{Job: j, Stage: StatusVerifying, Status: "Verifying...", JobChannel: updates}

		// TODO: Pass context to Verify
		success, err := j.Offloader.Verify()
		if err != nil {
			j.fail(fmt.Errorf("verification error: %w", err), updates)
			return
		}
		if !success {
			j.fail(fmt.Errorf("checksum mismatch"), updates)
			return
		}
	}

	// 3. COMPLETE & REPORT
	j.EndTime = time.Now()
	j.Status = StatusCompleted
	j.Err = nil

	// Generate reports
	j.generateReports()

	// Create Result
	j.Result = j.createResult()

	updates <- Msg{Job: j, Stage: StatusCompleted, Status: "Done!", Finished: true, JobChannel: updates}
}

func (j *Job) fail(err error, updates chan Msg) {
	j.EndTime = time.Now()
	j.Status = StatusFailed
	j.Err = err
	j.Result = j.createResult() // Create result even on failure
	updates <- Msg{Job: j, Stage: StatusFailed, Status: fmt.Sprintf("Failed: %v", err), Err: err, Finished: true, JobChannel: updates}
}

func (j *Job) generateReports() {
	for _, dst := range j.Offloader.Destinations {
		// PDF
		reportPath := dst + ".pdf"
		if err := report.GeneratePDF(reportPath, j.Offloader, j.StartTime, j.EndTime); err != nil {
			// Log warning?
		}

		// MHL
		mhlPath := dst + ".mhl"
		if err := mhl.GenerateMHL(mhlPath, j.Offloader.Files); err != nil {
			// Log warning?
		}
	}
}

func (j *Job) createResult() *output.JobResult {
	duration := j.EndTime.Sub(j.StartTime)
	speed := 0.0
	if duration.Seconds() > 0 {
		speed = float64(j.CopiedBytes) / 1024 / 1024 / duration.Seconds()
	}

	statusStr := "success"
	errStr := ""
	if j.Err != nil {
		statusStr = "failed"
		errStr = j.Err.Error()
	}

	return &output.JobResult{
		Timestamp:    time.Now(),
		Source:       j.Offloader.Source,
		Destinations: j.Offloader.Destinations,
		Status:       statusStr,
		TotalFiles:   len(j.Offloader.Files),
		TotalBytes:   j.TotalBytes,
		Duration:     duration.String(),
		DurationMs:   duration.Milliseconds(),
		SpeedMBps:    speed,
		Files:        j.Offloader.Files,
		Error:        errStr,
	}
}
