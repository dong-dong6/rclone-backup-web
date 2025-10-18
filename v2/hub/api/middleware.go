package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rclone-backup-web/hub/models"
	"github.com/rclone-backup-web/hub/services"
)

// AdminAuthMiddleware validates JWT tokens for admin endpoints
func AdminAuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := authService.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AgentAuthMiddleware validates API keys for agent endpoints
func AgentAuthMiddleware(authService *services.AuthService, db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		// Extract API key from "Bearer <api-key>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		apiKey := parts[1]
		
		// Find agent by API key
		ctx := context.Background()
		agentModel := models.NewAgentModel(db)
		
		// Get all agents and check API key (in production, use a more efficient method)
		agents, err := agentModel.List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate credentials"})
			c.Abort()
			return
		}

		var authenticatedAgent *models.Agent
		for _, agent := range agents {
			// Get agent with API key hash
			fullAgent, err := agentModel.GetByID(ctx, agent.ID)
			if err != nil {
				continue
			}
			
			if authService.ValidateAPIKey(apiKey, fullAgent.APIKeyHash) {
				authenticatedAgent = fullAgent
				break
			}
		}

		if authenticatedAgent == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// Store agent info in context
		c.Set("agent_id", authenticatedAgent.ID)
		c.Set("agent_name", authenticatedAgent.Name)
		c.Next()
	}
}