package api

import (
	"context"
	"log"
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
		log.Printf("[AdminAuth] Checking auth for path: %s", c.Request.URL.Path)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("[AdminAuth] No Authorization header found")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("[AdminAuth] Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := authService.ValidateJWT(token)
		if err != nil {
			log.Printf("[AdminAuth] JWT validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		log.Printf("[AdminAuth] JWT validated for user: %s, role: %s", claims.UserID, claims.Role)

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

		// Extract credentials from "Bearer <api-key>" or "Bearer <agent-id>:<api-key>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		credentials := parts[1]
		ctx := c.Request.Context()
		agentModel := models.NewAgentModel(db)
		var authenticatedAgent *models.Agent

		if strings.Contains(credentials, ":") {
			// Optimized lookup: "Bearer <agent-id>:<api-key>"
			credParts := strings.SplitN(credentials, ":", 2)
			agentIDStr, apiKey := credParts[0], credParts[1]

			agentID, err := uuid.Parse(agentIDStr)
			if err == nil {
				agent, err := agentModel.GetByID(ctx, agentID)
				if err == nil && authService.ValidateAPIKey(apiKey, agent.APIKeyHash) {
					authenticatedAgent = agent
				}
			}
		} else {
			// Legacy fallback: "Bearer <api-key>" (O(N) search)
			apiKey := credentials
			agents, err := agentModel.List(ctx)
			if err == nil {
				for _, agent := range agents {
					fullAgent, err := agentModel.GetByID(ctx, agent.ID)
					if err == nil && authService.ValidateAPIKey(apiKey, fullAgent.APIKeyHash) {
						authenticatedAgent = fullAgent
						break
					}
				}
			}
		}

		if authenticatedAgent == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key or ID"})
			c.Abort()
			return
		}

		// Store agent info in context
		c.Set("agent_id", authenticatedAgent.ID)
		c.Set("agent_name", authenticatedAgent.Name)
		c.Next()
	}
}