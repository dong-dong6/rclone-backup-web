package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	encryptionKey := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY"))
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if err := validateSecrets(encryptionKey, jwtSecret); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize services
	cryptoService := services.NewCryptoService(encryptionKey)
	authService := services.NewAuthService(jwtSecret)
	schedulerService := services.NewSchedulerService(db)
	sseService := services.NewSSEService()
	executionMonitor := services.NewExecutionMonitor(db)
	rcloneService := services.NewRcloneService()
	systemSettingsModel := models.NewSystemSettingsModel(db)
	metricsModel := models.NewMetricsModel(db)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	services.StartMetricsCleanup(cleanupCtx, metricsModel, systemSettingsModel)

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
	apiHandler := api.NewHandler(db, cryptoService, authService, schedulerService, sseService, rcloneService)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Agent facing API
		agent := v1.Group("/agent")
		{
			agent.POST("/register", apiHandler.RegisterAgent)
			agent.GET("/download", apiHandler.DownloadAgent)
			agent.GET("/install.sh", apiHandler.DownloadAgentScript)

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
			adminAuth.Use(api.AdminAuthMiddleware(authService, db))
			{
				adminAuth.POST("/logout", apiHandler.AdminLogout)

				// Agents management
				adminAuth.GET("/agents", apiHandler.ListAgents)
				adminAuth.GET("/agents/:id/metrics/latest", apiHandler.GetAgentMetricsLatest)
				adminAuth.GET("/agents/:id/metrics/history", apiHandler.GetAgentMetricsHistory)
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

	// Server-Sent Events endpoint (admin auth required)
	router.GET("/events", api.AdminAuthMiddleware(authService, db), apiHandler.SSEEndpoint)

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

	// ============================================
	// Static file serving (React SPA)
	// ============================================
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		staticPath = "./static/web"
	}

	// Check if static directory exists
	if _, err := os.Stat(staticPath); err == nil {
		log.Printf("Serving static files from: %s", staticPath)

		// Serve static assets (JS, CSS, images)
		router.Static("/assets", staticPath+"/assets")

		// Serve favicon.ico if exists
		faviconPath := staticPath + "/favicon.ico"
		if _, err := os.Stat(faviconPath); err == nil {
			router.GET("/favicon.ico", func(c *gin.Context) {
				c.File(faviconPath)
			})
		}

		// Serve vite.svg if exists
		viteSvgPath := staticPath + "/vite.svg"
		if _, err := os.Stat(viteSvgPath); err == nil {
			router.GET("/vite.svg", func(c *gin.Context) {
				c.File(viteSvgPath)
			})
		}

		// Serve index.html for SPA routing
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			// Skip API requests
			if strings.HasPrefix(path, "/api/") {
				c.JSON(404, gin.H{"error": "API endpoint not found"})
				return
			}

			// Skip /events (SSE endpoint)
			if path == "/events" {
				c.JSON(404, gin.H{"error": "SSE endpoint not configured"})
				return
			}

			// Skip /health
			if path == "/health" {
				c.JSON(404, gin.H{"error": "Health endpoint"})
				return
			}

			// For all other routes, serve index.html (SPA routing)
			c.File(staticPath + "/index.html")
		})

		// Serve root
		router.GET("/", func(c *gin.Context) {
			c.File(staticPath + "/index.html")
		})
	} else {
		log.Printf("Static directory not found at %s, skipping frontend serving", staticPath)
	}

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

	cleanupCancel()

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

func validateSecrets(encryptionKey, jwtSecret string) error {
	if len(encryptionKey) < 16 {
		return fmt.Errorf("ENCRYPTION_KEY must be at least 16 characters long")
	}

	if len(jwtSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters long")
	}

	return nil
}
