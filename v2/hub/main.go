package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rclone-backup-web/hub/api"
	"github.com/rclone-backup-web/hub/models"
	"github.com/rclone-backup-web/hub/services"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Set Gin mode based on environment
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "debug" // Enable debug mode by default
	}
	gin.SetMode(ginMode)

	// Initialize database
	db, err := models.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	cryptoService := services.NewCryptoService(os.Getenv("ENCRYPTION_KEY"))
	authService := services.NewAuthService(os.Getenv("JWT_SECRET"))
	schedulerService := services.NewSchedulerService(db)
	sseService := services.NewSSEService()
	executionMonitor := services.NewExecutionMonitor(db)

	// Setup Gin router with logger and recovery middleware
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Initialize API handlers
	apiHandler := api.NewHandler(db, cryptoService, authService, schedulerService, sseService)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Agent facing API
		agent := v1.Group("/agent")
		{
			agent.POST("/register", apiHandler.RegisterAgent)
			agent.GET("/download", apiHandler.DownloadAgent)

			// Agent authenticated routes
			agentAuth := agent.Group("")
			agentAuth.Use(api.AgentAuthMiddleware(authService, db))
			{
				agentAuth.POST("/heartbeat", apiHandler.AgentHeartbeat)
				agentAuth.GET("/tasks", apiHandler.GetAgentTasks)
				agentAuth.PUT("/executions/:executionId", apiHandler.UpdateExecution)
				agentAuth.POST("/executions/:executionId/logs", apiHandler.StreamExecutionLogs)
			}
		}

		// Admin facing API
		admin := v1.Group("/admin")
		{
			// Public endpoints (no auth required)
			admin.POST("/login", apiHandler.AdminLogin)

			// Authenticated endpoints
			adminAuth := admin.Group("")
			adminAuth.Use(api.AdminAuthMiddleware(authService))
			{
				adminAuth.POST("/logout", apiHandler.AdminLogout)

				// Agents management
				adminAuth.GET("/agents", apiHandler.ListAgents)
				adminAuth.DELETE("/agents/:id", apiHandler.DeleteAgent)
				adminAuth.POST("/agents/registration-token", apiHandler.CreateRegistrationToken)

				// Tasks management
				adminAuth.GET("/tasks", apiHandler.ListTasks)
				adminAuth.GET("/tasks/:id", apiHandler.GetTask)
				adminAuth.POST("/tasks", apiHandler.CreateTask)
				adminAuth.PUT("/tasks/:id", apiHandler.UpdateTask)
				adminAuth.DELETE("/tasks/:id", apiHandler.DeleteTask)

				// Remotes management
				adminAuth.GET("/remotes", apiHandler.ListRemotes)
				adminAuth.GET("/remotes/:id", apiHandler.GetRemote)
				adminAuth.POST("/remotes", apiHandler.CreateRemote)
				adminAuth.PUT("/remotes/:id", apiHandler.UpdateRemote)
				adminAuth.DELETE("/remotes/:id", apiHandler.DeleteRemote)
				adminAuth.POST("/remotes/:id/test", apiHandler.TestRemote)

				// Executions
				adminAuth.GET("/executions", apiHandler.ListExecutions)
				adminAuth.GET("/executions/:id", apiHandler.GetExecutionDetail)
				adminAuth.POST("/executions/trigger", apiHandler.TriggerExecution)
				adminAuth.POST("/executions/:id/cancel", apiHandler.CancelExecution)

				// Dashboard & Statistics
				adminAuth.GET("/dashboard/stats", apiHandler.GetDashboardStats)
				adminAuth.GET("/dashboard/recent", apiHandler.GetRecentActivity)
				adminAuth.GET("/dashboard/charts", apiHandler.GetChartData)
				
				// Statistics
				adminAuth.GET("/statistics/overview", apiHandler.GetStatisticsOverview)
				adminAuth.GET("/statistics/agents/:id", apiHandler.GetAgentStatistics)
				adminAuth.GET("/statistics/tasks/:id", apiHandler.GetTaskStatistics)
			}
		}
	}

	// Server-Sent Events endpoint
	router.GET("/events", apiHandler.SSEEndpoint)

	// Health check - support both GET and HEAD methods
	healthHandler := func(c *gin.Context) {
		// 深度健康检查
		health := gin.H{
			"status": "healthy",
			"time":   time.Now().Unix(),
		}
		
		// 检查数据库连接
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{
				"status": "unhealthy",
				"error":  "database connection failed",
				"time":   time.Now().Unix(),
			})
			return
		}
		
		// 返回健康状态
		c.JSON(200, health)
	}
	router.GET("/health", healthHandler)
	router.HEAD("/health", healthHandler)

	// Start scheduler service
	go schedulerService.Start()
	
	// Start SSE service
	go sseService.Start()
	
	// Start execution monitor
	monitorCtx := context.Background()
	go executionMonitor.Start(monitorCtx)

	// Setup server
	srv := &http.Server{
		Addr:    ":" + getPort(),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("Hub API server started on port %s", getPort())

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}