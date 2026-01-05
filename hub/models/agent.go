package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Agent struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	APIKeyHash    string     `json:"-"`
	Status        string     `json:"status"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
	CurrentTask   *uuid.UUID `json:"current_task,omitempty"`
	Version       *string    `json:"version,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AgentModel struct {
	db *pgxpool.Pool
}

func NewAgentModel(db *pgxpool.Pool) *AgentModel {
	return &AgentModel{db: db}
}

// Create creates a new agent
func (m *AgentModel) Create(ctx context.Context, name, apiKeyHash string) (*Agent, error) {
	agent := &Agent{
		ID:         uuid.New(),
		Name:       name,
		APIKeyHash: apiKeyHash,
		Status:     "offline",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO agents (id, name, api_key_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, current_task, version, created_at, updated_at
	`

	err := m.db.QueryRow(ctx, query,
		agent.ID,
		agent.Name,
		agent.APIKeyHash,
		agent.Status,
		agent.CreatedAt,
		agent.UpdatedAt,
	).Scan(&agent.ID, &agent.CurrentTask, &agent.Version, &agent.CreatedAt, &agent.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return agent, nil
}

// GetByID retrieves an agent by ID
func (m *AgentModel) GetByID(ctx context.Context, id uuid.UUID) (*Agent, error) {
	agent := &Agent{}
	query := `
		SELECT id, name, api_key_hash, status, last_heartbeat, current_task, version, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	err := m.db.QueryRow(ctx, query, id).Scan(
		&agent.ID,
		&agent.Name,
		&agent.APIKeyHash,
		&agent.Status,
		&agent.LastHeartbeat,
		&agent.CurrentTask,
		&agent.Version,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return agent, nil
}

// GetByAPIKeyHash retrieves an agent by API key hash
func (m *AgentModel) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*Agent, error) {
	agent := &Agent{}
	query := `
		SELECT id, name, api_key_hash, status, last_heartbeat, current_task, version, created_at, updated_at
		FROM agents
		WHERE api_key_hash = $1
	`

	err := m.db.QueryRow(ctx, query, apiKeyHash).Scan(
		&agent.ID,
		&agent.Name,
		&agent.APIKeyHash,
		&agent.Status,
		&agent.LastHeartbeat,
		&agent.CurrentTask,
		&agent.Version,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return agent, nil
}

// List retrieves all agents
func (m *AgentModel) List(ctx context.Context) ([]*Agent, error) {
	query := `
		SELECT id, name, status, last_heartbeat, current_task, version, created_at, updated_at
		FROM agents
		ORDER BY created_at DESC
	`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := []*Agent{}
	for rows.Next() {
		agent := &Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Status,
			&agent.LastHeartbeat,
			&agent.CurrentTask,
			&agent.Version,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// UpdateHeartbeat updates agent's last heartbeat and status
func (m *AgentModel) UpdateHeartbeat(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE agents
		SET last_heartbeat = NOW() - INTERVAL '0 seconds', status = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := m.db.Exec(ctx, query, id, status)
	return err
}

// Delete deletes an agent
func (m *AgentModel) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM agents WHERE id = $1`
	_, err := m.db.Exec(ctx, query, id)
	return err
}

// CheckOfflineAgents marks agents as offline if they haven't sent heartbeat recently
func (m *AgentModel) CheckOfflineAgents(ctx context.Context, timeout time.Duration) error {
	query := `
		UPDATE agents
		SET status = 'offline'
		WHERE status != 'offline' 
		AND last_heartbeat < NOW() - INTERVAL '%d seconds'
	`

	_, err := m.db.Exec(ctx, fmt.Sprintf(query, int(timeout.Seconds())))
	return err
}
