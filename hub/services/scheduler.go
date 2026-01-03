package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"github.com/rclone-backup-web/hub/models"
)

type SchedulerService struct {
	db            *pgxpool.Pool
	cron          *cron.Cron
	taskSchedules map[uuid.UUID]cron.EntryID
	mu            sync.RWMutex
	syncFlags     map[uuid.UUID]bool
	syncMu        sync.RWMutex
	lastDispatch  map[uuid.UUID]time.Time
	dispatchMu    sync.RWMutex
}

func NewSchedulerService(db *pgxpool.Pool) *SchedulerService {
	return &SchedulerService{
		db:            db,
		cron:          cron.New(cron.WithSeconds()),
		taskSchedules: make(map[uuid.UUID]cron.EntryID),
		syncFlags:     make(map[uuid.UUID]bool),
		lastDispatch:  make(map[uuid.UUID]time.Time),
	}
}

// Start starts the scheduler service
func (s *SchedulerService) Start() {
	log.Println("Starting scheduler service...")
	
	// Load all active tasks
	s.reloadTasks()
	
	// Start cron scheduler
	s.cron.Start()
	
	// Periodically check for offline agents
	go s.checkOfflineAgents()
	
	// Periodically reload tasks
	go s.periodicallyReloadTasks()
	
	log.Println("Scheduler service started")
}

// Stop stops the scheduler service
func (s *SchedulerService) Stop() {
	log.Println("Stopping scheduler service...")
	s.cron.Stop()
	log.Println("Scheduler service stopped")
}

// reloadTasks reloads all active tasks from database
func (s *SchedulerService) reloadTasks() {
	ctx := context.Background()
	taskModel := models.NewTaskModel(s.db)
	
	tasks, err := taskModel.List(ctx)
	if err != nil {
		log.Printf("Failed to load tasks: %v", err)
		return
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Remove all existing schedules
	for taskID, entryID := range s.taskSchedules {
		s.cron.Remove(entryID)
		delete(s.taskSchedules, taskID)
	}
	
	// Add new schedules
	for _, task := range tasks {
		if !task.IsActive || task.Schedule == "" {
			continue
		}
		
		taskCopy := *task
		entryID, err := s.cron.AddFunc(task.Schedule, func() {
			s.triggerTask(taskCopy.ID)
		})
		
		if err != nil {
			log.Printf("Failed to schedule task %s: %v", task.ID, err)
			continue
		}
		
		s.taskSchedules[task.ID] = entryID
		log.Printf("Scheduled task %s (%s) with cron: %s", task.ID, task.Name, task.Schedule)
	}
}

// triggerTask triggers a task execution
func (s *SchedulerService) triggerTask(taskID uuid.UUID) {
	ctx := context.Background()
	
	// Get task details
	taskModel := models.NewTaskModel(s.db)
	task, err := taskModel.GetByID(ctx, taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
		return
	}
	
	// Get assigned agents
	agents := task.AssignedAgents
	if len(agents) == 0 {
		log.Printf("No agents assigned to task %s", taskID)
		return
	}
	
	// Create execution for each assigned agent
	executionModel := models.NewExecutionModel(s.db)
	for _, agentID := range agents {
		// Check if agent is online
		agentModel := models.NewAgentModel(s.db)
		agent, err := agentModel.GetByID(ctx, agentID)
		if err != nil || agent.Status == "offline" {
			log.Printf("Agent %s is offline, skipping task %s", agentID, taskID)
			continue
		}
		
		// Create execution record
		_, err = executionModel.Create(ctx, taskID, agentID, "central")
		if err != nil {
			log.Printf("Failed to create execution for task %s, agent %s: %v", taskID, agentID, err)
			continue
		}
		
		log.Printf("Created execution for task %s on agent %s", taskID, agentID)
	}
}

// NeedsConfigSync checks if an agent needs configuration sync
func (s *SchedulerService) NeedsConfigSync(agentID uuid.UUID) bool {
	s.syncMu.RLock()
	defer s.syncMu.RUnlock()
	
	needsSync, exists := s.syncFlags[agentID]
	if exists && needsSync {
		// Clear the flag after reading
		s.syncMu.RUnlock()
		s.syncMu.Lock()
		s.syncFlags[agentID] = false
		s.syncMu.Unlock()
		s.syncMu.RLock()
		return true
	}
	
	return false
}

// MarkAgentForSync marks an agent for configuration sync
func (s *SchedulerService) MarkAgentForSync(agentID uuid.UUID) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	s.syncFlags[agentID] = true
}

// checkOfflineAgents periodically checks for offline agents
func (s *SchedulerService) checkOfflineAgents() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		agentModel := models.NewAgentModel(s.db)
		
		// Mark agents as offline if they haven't sent heartbeat in 2 minutes
		if err := agentModel.CheckOfflineAgents(ctx, 2*time.Minute); err != nil {
			log.Printf("Failed to check offline agents: %v", err)
		}
	}
}

// periodicallyReloadTasks periodically reloads tasks from database
func (s *SchedulerService) periodicallyReloadTasks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.reloadTasks()
	}
}

// ShouldTaskRunNow checks if a task should run based on its cron schedule
func (s *SchedulerService) ShouldTaskRunNow(taskID uuid.UUID, cronExpr string, now time.Time) bool {
	// Parse cron expression
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		log.Printf("Failed to parse cron expression %s: %v", cronExpr, err)
		return false
	}

	// Get last dispatch time
	s.dispatchMu.RLock()
	lastDispatch, exists := s.lastDispatch[taskID]
	s.dispatchMu.RUnlock()

	// If never dispatched, use a time 1 hour ago as reference
	if !exists {
		lastDispatch = now.Add(-1 * time.Hour)
	}

	// Calculate next run time after last dispatch
	nextRun := schedule.Next(lastDispatch)

	// Check if it's time to run (with 1-minute tolerance)
	return now.After(nextRun) || now.Equal(nextRun) || 
		(now.After(nextRun.Add(-1*time.Minute)) && now.Before(nextRun.Add(1*time.Minute)))
}

// MarkTaskDispatched marks that a task has been dispatched
func (s *SchedulerService) MarkTaskDispatched(taskID uuid.UUID, dispatchTime time.Time) {
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	s.lastDispatch[taskID] = dispatchTime
}