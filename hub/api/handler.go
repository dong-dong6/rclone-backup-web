package api

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rclone-backup-web/hub/models"
	"github.com/rclone-backup-web/hub/services"
)

type Handler struct {
	db               *pgxpool.Pool
	cryptoService    *services.CryptoService
	authService      *services.AuthService
	schedulerService *services.SchedulerService
	sseService       *services.SSEService
	rcloneService    *services.RcloneService
	wsService        *services.AgentWSService
	fsBroker         *services.AgentFSRequestBroker
	oauthFlows       *oauthFlowStore
	logTokens        bool

	agentStatusMu    sync.Mutex
	agentStatusCache map[uuid.UUID]string
}

func NewHandler(
	db *pgxpool.Pool,
	cryptoService *services.CryptoService,
	authService *services.AuthService,
	schedulerService *services.SchedulerService,
	sseService *services.SSEService,
	rcloneService *services.RcloneService,
) *Handler {
	return &Handler{
		db:               db,
		cryptoService:    cryptoService,
		authService:      authService,
		schedulerService: schedulerService,
		sseService:       sseService,
		rcloneService:    rcloneService,
		wsService:        services.NewAgentWSService(),
		fsBroker:         services.NewAgentFSRequestBroker(),
		oauthFlows:       newOAuthFlowStore(),
		logTokens:        strings.EqualFold(os.Getenv("DEBUG_LOG_TOKENS"), "true"),
		agentStatusCache: make(map[uuid.UUID]string),
	}
}

// logAuditEvent logs an audit event
func (h *Handler) logAuditEvent(c *gin.Context, userID uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]interface{}) {
	auditModel := models.NewAuditModel(h.db)

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Add request method and path to details
	if details == nil {
		details = make(map[string]interface{})
	}
	details["method"] = c.Request.Method
	details["path"] = c.Request.URL.Path

	// Log asynchronously to avoid blocking the request
	go func() {
		ctx := context.Background()
		err := auditModel.Create(ctx, &userID, action, resourceType, &resourceID, details, ipAddress, userAgent)
		if err != nil {
			// Log error but don't fail the request
			// In production, use proper logging
			fmt.Printf("Failed to log audit event: %v\n", err)
		}
	}()
}

// checkScheduledTasksForAgent checks if there are any tasks scheduled to run now for this agent
func (h *Handler) checkScheduledTasksForAgent(ctx context.Context, agentID uuid.UUID) ([]*models.BackupTask, error) {
	// Get all active tasks assigned to this agent
	taskModel := models.NewTaskModel(h.db)
	tasks, err := taskModel.GetAgentTasks(ctx, agentID)
	if err != nil {
		return nil, err
	}

	var scheduledTasks []*models.BackupTask
	now := time.Now()

	for _, task := range tasks {
		if !task.IsActive || task.Schedule == "" {
			continue
		}

		// Check if this task should run now based on its schedule
		if h.schedulerService.ShouldTaskRunNow(task.ID, task.Schedule, now) {
			scheduledTasks = append(scheduledTasks, task)

			// Mark that we've dispatched this task
			h.schedulerService.MarkTaskDispatched(task.ID, now)
		}
	}

	return scheduledTasks, nil
}
