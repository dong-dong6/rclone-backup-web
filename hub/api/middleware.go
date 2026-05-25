package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rclone-backup-web/hub/models"
	"github.com/rclone-backup-web/hub/services"
)

// AdminAuthMiddleware validates JWT tokens for admin endpoints, enforces admin role and active session
func AdminAuthMiddleware(authService *services.AuthService, db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[AdminAuth] Checking auth for path: %s", c.Request.URL.Path)

		token, err := extractBearerToken(c)
		if err != nil {
			log.Printf("[AdminAuth] Token extraction failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		claims, err := authService.ValidateJWT(token)
		if err != nil {
			log.Printf("[AdminAuth] JWT validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			log.Printf("[AdminAuth] Invalid user ID in claims: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
			c.Abort()
			return
		}

		userModel := models.NewUserModel(db)

		// Validate active session by token hash
		session, err := userModel.GetSessionByToken(c.Request.Context(), authService.HashToken(token))
		if err != nil || session.UserID != userID {
			log.Printf("[AdminAuth] Session not found or mismatched user for token hash")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
			c.Abort()
			return
		}

		user, err := userModel.GetByID(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[AdminAuth] Failed to load user: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if !user.IsActive {
			log.Printf("[AdminAuth] Inactive user %s", userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "User inactive"})
			c.Abort()
			return
		}

		// Enforce admin role from both claims and DB
		if claims.Role != "admin" || user.Role != "admin" {
			log.Printf("[AdminAuth] User %s is not admin (claims: %s, db: %s)", userID, claims.Role, user.Role)
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
			c.Abort()
			return
		}

		log.Printf("[AdminAuth] JWT validated for user: %s, role: %s", claims.UserID, claims.Role)

		// Store claims in context
		c.Set("user_id", userID)
		c.Set("role", claims.Role)
		c.Set("session_id", session.ID)
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" && parts[1] != "" {
			return parts[1], nil
		}
	}

	// Fallback for SSE: allow token in query string
	if token := c.Query("token"); token != "" {
		return token, nil
	}

	return "", fmt.Errorf("authorization token missing")
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

		// Extract credentials from "Bearer <agent-id>:<api-key>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		credentials := parts[1]
		if !strings.Contains(credentials, ":") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization credentials"})
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		agentModel := models.NewAgentModel(db)
		var authenticatedAgent *models.Agent

		// Optimized lookup: "Bearer <agent-id>:<api-key>"
		credParts := strings.SplitN(credentials, ":", 2)
		agentIDStr := strings.TrimSpace(credParts[0])
		apiKey := strings.TrimSpace(credParts[1])

		agentID, err := uuid.Parse(agentIDStr)
		if err == nil && agentID != uuid.Nil && apiKey != "" {
			agent, err := agentModel.GetByID(ctx, agentID)
			if err == nil && authService.ValidateAPIKey(apiKey, agent.APIKeyHash) {
				authenticatedAgent = agent
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
