package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rclone-backup-web/hub/services"
)

type Handler struct {
	db               *pgxpool.Pool
	cryptoService    *services.CryptoService
	authService      *services.AuthService
	schedulerService *services.SchedulerService
	sseService       *services.SSEService
}

func NewHandler(
	db *pgxpool.Pool,
	cryptoService *services.CryptoService,
	authService *services.AuthService,
	schedulerService *services.SchedulerService,
	sseService *services.SSEService,
) *Handler {
	return &Handler{
		db:               db,
		cryptoService:    cryptoService,
		authService:      authService,
		schedulerService: schedulerService,
		sseService:       sseService,
	}
}