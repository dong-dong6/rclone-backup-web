package services

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler handles local task scheduling
type Scheduler struct {
	workDir    string
	cron       *cron.Cron
	tasks      map[string]*ScheduledTask
	tasksMutex sync.RWMutex
	lastRuns   map[string]time.Time
	runsMutex  sync.RWMutex
}

// ScheduledTask represents a locally cached task
type ScheduledTask struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Schedule            string    `json:"schedule"`
	RemoteConfig        string    `json:"remote_config"`
	SourcePath          string    `json:"source_path"`
	DestPath            string    `json:"dest_path"`
	RcloneArgs          []string  `json:"rclone_args"`
	Enabled             bool      `json:"enabled"`
	BackupMode          string    `json:"backup_mode,omitempty"`
	ArchiveFormat       string    `json:"archive_format,omitempty"`
	EncryptionEnabled   bool      `json:"encryption_enabled"`
	EncryptionPassword  string    `json:"encryption_password,omitempty"`
	EncryptionPassword2 string    `json:"encryption_password2,omitempty"`
	MaxRetention        int       `json:"max_retention,omitempty"`
	LastRun             time.Time `json:"last_run,omitempty"`
	NextRun             time.Time `json:"next_run,omitempty"`
}

// NewScheduler creates a new scheduler
func NewScheduler(workDir string) *Scheduler {
	return &Scheduler{
		workDir:  workDir,
		cron:     cron.New(),
		tasks:    make(map[string]*ScheduledTask),
		lastRuns: make(map[string]time.Time),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	log.Println("[Scheduler] Starting local scheduler")

	// Load cached tasks
	if err := s.loadTasks(); err != nil {
		log.Printf("[Scheduler] Failed to load cached tasks: %v", err)
	}

	// Start cron
	s.cron.Start()

	// Wait for context cancellation
	<-ctx.Done()

	// Stop cron
	s.cron.Stop()

	// Save tasks before exit
	if err := s.saveTasks(); err != nil {
		log.Printf("[Scheduler] Failed to save tasks: %v", err)
	}

	log.Println("[Scheduler] Stopped")
}

// UpdateTasks updates the scheduled tasks
func (s *Scheduler) UpdateTasks(tasks []Task) {
	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	// Clear existing cron entries
	entries := s.cron.Entries()
	for _, entry := range entries {
		s.cron.Remove(entry.ID)
	}

	// Clear tasks map
	s.tasks = make(map[string]*ScheduledTask)

	// Add new tasks
	for _, task := range tasks {
		if !task.Enabled {
			continue
		}

		scheduledTask := &ScheduledTask{
			ID:                  task.ID,
			Name:                task.Name,
			Schedule:            task.Schedule,
			RemoteConfig:        task.RemoteConfig,
			SourcePath:          task.SourcePath,
			DestPath:            task.DestPath,
			RcloneArgs:          task.RcloneArgs,
			Enabled:             task.Enabled,
			BackupMode:          task.BackupMode,
			ArchiveFormat:       task.ArchiveFormat,
			EncryptionEnabled:   task.EncryptionEnabled,
			EncryptionPassword:  task.EncryptionPassword,
			EncryptionPassword2: task.EncryptionPassword2,
			MaxRetention:        task.MaxRetention,
		}

		// Add to cron
		taskID := task.ID
		entryID, err := s.cron.AddFunc(task.Schedule, func() {
			log.Printf("[Scheduler] Task %s triggered by schedule", taskID)
			// Task execution is handled by the main agent loop
		})

		if err != nil {
			log.Printf("[Scheduler] Failed to schedule task %s: %v", task.ID, err)
			continue
		}

		// Get next run time
		entry := s.cron.Entry(entryID)
		scheduledTask.NextRun = entry.Next

		s.tasks[task.ID] = scheduledTask
	}

	// Save to cache
	if err := s.saveTasks(); err != nil {
		log.Printf("[Scheduler] Failed to save tasks: %v", err)
	}

	log.Printf("[Scheduler] Updated %d tasks", len(s.tasks))
}

// GetDueTasks returns tasks that are due to run
func (s *Scheduler) GetDueTasks() []*ScheduledTask {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	s.runsMutex.RLock()
	defer s.runsMutex.RUnlock()

	var dueTasks []*ScheduledTask
	now := time.Now()

	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}

		// Check if task is due
		lastRun, exists := s.lastRuns[task.ID]
		if !exists {
			// Never run, check if it's time
			if task.NextRun.Before(now) || task.NextRun.Equal(now) {
				dueTasks = append(dueTasks, task)
			}
		} else {
			// Check based on schedule
			schedule, err := cron.ParseStandard(task.Schedule)
			if err != nil {
				continue
			}

			nextRun := schedule.Next(lastRun)
			if nextRun.Before(now) || nextRun.Equal(now) {
				dueTasks = append(dueTasks, task)
			}
		}
	}

	return dueTasks
}

// UpdateLastRun updates the last run time for a task
func (s *Scheduler) UpdateLastRun(taskID string) {
	s.runsMutex.Lock()
	defer s.runsMutex.Unlock()

	s.lastRuns[taskID] = time.Now()

	// Update in tasks map
	s.tasksMutex.Lock()
	if task, exists := s.tasks[taskID]; exists {
		task.LastRun = time.Now()
	}
	s.tasksMutex.Unlock()

	// Save to cache
	s.saveTasks()
}

// loadTasks loads tasks from cache file
func (s *Scheduler) loadTasks() error {
	cachePath := filepath.Join(s.workDir, "tasks.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet
		}
		return err
	}

	var tasks []*ScheduledTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}

	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	for _, task := range tasks {
		s.tasks[task.ID] = task
		if !task.LastRun.IsZero() {
			s.lastRuns[task.ID] = task.LastRun
		}
	}

	log.Printf("[Scheduler] Loaded %d cached tasks", len(tasks))
	return nil
}

// saveTasks saves tasks to cache file
func (s *Scheduler) saveTasks() error {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	// Convert map to slice
	var tasks []*ScheduledTask
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(s.workDir, "tasks.json")
	return os.WriteFile(cachePath, data, 0600)
}
