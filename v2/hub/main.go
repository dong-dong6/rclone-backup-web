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

	// Setup Gin router
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
		admin.Use(api.AdminAuthMiddleware(authService))
		{
			// Authentication
			admin.POST("/login", apiHandler.AdminLogin)
			
			// Agents management
			admin.GET("/agents", apiHandler.ListAgents)
			admin.DELETE("/agents/:id", apiHandler.DeleteAgent)
			admin.POST("/agents/registration-token", apiHandler.CreateRegistrationToken)
			
			// Tasks management
			admin.GET("/tasks", apiHandler.ListTasks)
			admin.POST("/tasks", apiHandler.CreateTask)
			admin.PUT("/tasks/:id", apiHandler.UpdateTask)
			admin.DELETE("/tasks/:id", apiHandler.DeleteTask)
			
			// Remotes management
			admin.GET("/remotes", apiHandler.ListRemotes)
			admin.POST("/remotes", apiHandler.CreateRemote)
			admin.PUT("/remotes/:id", apiHandler.UpdateRemote)
			admin.DELETE("/remotes/:id", apiHandler.DeleteRemote)
			
			// Executions
			admin.GET("/executions", apiHandler.ListExecutions)
			admin.GET("/executions/:id", apiHandler.GetExecutionDetail)
			admin.POST("/executions/trigger", apiHandler.TriggerExecution)
		}
	}

	// Server-Sent Events endpoint
	router.GET("/events", apiHandler.SSEEndpoint)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "time": time.Now().Unix()})
	})

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