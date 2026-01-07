package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RcloneRemote struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Type       *string   `json:"type,omitempty"`
	ConfigData string    `json:"-"` // Encrypted, not exposed in JSON
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type RemoteModel struct {
	db *pgxpool.Pool
}

func NewRemoteModel(db *pgxpool.Pool) *RemoteModel {
	return &RemoteModel{db: db}
}

// Create creates a new rclone remote
func (m *RemoteModel) Create(ctx context.Context, name, encryptedConfig string, remoteType *string) (*RcloneRemote, error) {
	remote := &RcloneRemote{
		ID:         uuid.New(),
		Name:       name,
		Type:       remoteType,
		ConfigData: encryptedConfig,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO rclone_remotes (id, name, config_data, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := m.db.QueryRow(ctx, query,
		remote.ID,
		remote.Name,
		remote.ConfigData,
		remote.Type,
		remote.CreatedAt,
		remote.UpdatedAt,
	).Scan(&remote.ID, &remote.CreatedAt, &remote.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return remote, nil
}

// GetByID retrieves a remote by ID
func (m *RemoteModel) GetByID(ctx context.Context, id uuid.UUID) (*RcloneRemote, error) {
	remote := &RcloneRemote{}
	query := `
		SELECT id, name, config_data, type, created_at, updated_at
		FROM rclone_remotes
		WHERE id = $1
	`

	err := m.db.QueryRow(ctx, query, id).Scan(
		&remote.ID,
		&remote.Name,
		&remote.ConfigData,
		&remote.Type,
		&remote.CreatedAt,
		&remote.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return remote, nil
}

// List retrieves all remotes
func (m *RemoteModel) List(ctx context.Context) ([]*RcloneRemote, error) {
	query := `
		SELECT id, name, type, created_at, updated_at
		FROM rclone_remotes
		ORDER BY name ASC
	`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	remotes := []*RcloneRemote{}
	for rows.Next() {
		remote := &RcloneRemote{}
		err := rows.Scan(
			&remote.ID,
			&remote.Name,
			&remote.Type,
			&remote.CreatedAt,
			&remote.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		remotes = append(remotes, remote)
	}

	return remotes, nil
}

// Update updates a remote configuration
func (m *RemoteModel) Update(ctx context.Context, id uuid.UUID, name, encryptedConfig string, remoteType *string) error {
	query := `
		UPDATE rclone_remotes
		SET name = $2, config_data = $3, type = $4, updated_at = NOW()
		WHERE id = $1
	`

	_, err := m.db.Exec(ctx, query, id, name, encryptedConfig, remoteType)
	return err
}

// Delete deletes a remote
func (m *RemoteModel) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM rclone_remotes WHERE id = $1`
	_, err := m.db.Exec(ctx, query, id)
	return err
}
