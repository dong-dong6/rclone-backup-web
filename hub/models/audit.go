package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLog struct {
	ID           uuid.UUID       `json:"id"`
	UserID       *uuid.UUID      `json:"user_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *uuid.UUID      `json:"resource_id"`
	Details      json.RawMessage `json:"details"`
	IPAddress    string          `json:"ip_address"`
	UserAgent    string          `json:"user_agent"`
	CreatedAt    time.Time       `json:"created_at"`
	
	// Additional fields from joins
	Username     string          `json:"username,omitempty"`
}

type AuditModel struct {
	db *pgxpool.Pool
}

func NewAuditModel(db *pgxpool.Pool) *AuditModel {
	return &AuditModel{db: db}
}

// Create creates a new audit log entry
func (m *AuditModel) Create(ctx context.Context, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details map[string]interface{}, ipAddress, userAgent string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = m.db.Exec(ctx, query,
		uuid.New(),
		userID,
		action,
		resourceType,
		resourceID,
		detailsJSON,
		ipAddress,
		userAgent,
		time.Now(),
	)

	return err
}

// List retrieves audit logs with optional filters
func (m *AuditModel) List(ctx context.Context, limit, offset int, userID *uuid.UUID, action, resourceType string) ([]*AuditLog, error) {
	query := `
		SELECT 
			a.id, a.user_id, a.action, a.resource_type, a.resource_id, 
			a.details, a.ip_address, a.user_agent, a.created_at,
			u.username
		FROM audit_logs a
		LEFT JOIN users u ON a.user_id = u.id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if userID != nil {
		argCount++
		query += fmt.Sprintf(" AND a.user_id = $%d", argCount)
		args = append(args, *userID)
	}

	if action != "" {
		argCount++
		query += fmt.Sprintf(" AND a.action = $%d", argCount)
		args = append(args, action)
	}

	if resourceType != "" {
		argCount++
		query += fmt.Sprintf(" AND a.resource_type = $%d", argCount)
		args = append(args, resourceType)
	}

	query += " ORDER BY a.created_at DESC"

	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}

	if offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	rows, err := m.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []*AuditLog{}
	for rows.Next() {
		log := &AuditLog{}
		var username *string
		
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
			&username,
		)
		if err != nil {
			return nil, err
		}
		
		if username != nil {
			log.Username = *username
		}
		
		logs = append(logs, log)
	}

	return logs, nil
}

// GetByResource retrieves audit logs for a specific resource
func (m *AuditModel) GetByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*AuditLog, error) {
	query := `
		SELECT 
			a.id, a.user_id, a.action, a.resource_type, a.resource_id, 
			a.details, a.ip_address, a.user_agent, a.created_at,
			u.username
		FROM audit_logs a
		LEFT JOIN users u ON a.user_id = u.id
		WHERE a.resource_type = $1 AND a.resource_id = $2
		ORDER BY a.created_at DESC
	`

	rows, err := m.db.Query(ctx, query, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []*AuditLog{}
	for rows.Next() {
		log := &AuditLog{}
		var username *string
		
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
			&username,
		)
		if err != nil {
			return nil, err
		}
		
		if username != nil {
			log.Username = *username
		}
		
		logs = append(logs, log)
	}

	return logs, nil
}

// CleanupOldLogs removes audit logs older than the specified duration
func (m *AuditModel) CleanupOldLogs(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	_, err := m.db.Exec(ctx, query, cutoffTime)
	return err
}