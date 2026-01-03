package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegistrationToken struct {
	ID            uuid.UUID  `json:"id"`
	Token         string     `json:"token"`
	Used          bool       `json:"used"`
	UsedByAgentID *uuid.UUID `json:"used_by_agent_id,omitempty"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type RegistrationTokenModel struct {
	db *pgxpool.Pool
}

func NewRegistrationTokenModel(db *pgxpool.Pool) *RegistrationTokenModel {
	return &RegistrationTokenModel{db: db}
}

// Create creates a new registration token
func (m *RegistrationTokenModel) Create(ctx context.Context, token string, expiresIn time.Duration) (*RegistrationToken, error) {
	regToken := &RegistrationToken{
		ID:        uuid.New(),
		Token:     token,
		Used:      false,
		ExpiresAt: time.Now().Add(expiresIn),
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO registration_tokens (id, token, used, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := m.db.QueryRow(ctx, query,
		regToken.ID,
		regToken.Token,
		regToken.Used,
		regToken.ExpiresAt,
		regToken.CreatedAt,
	).Scan(&regToken.ID, &regToken.CreatedAt)

	if err != nil {
		return nil, err
	}

	return regToken, nil
}

// GetByToken retrieves a token by its value
func (m *RegistrationTokenModel) GetByToken(ctx context.Context, token string) (*RegistrationToken, error) {
	regToken := &RegistrationToken{}
	query := `
		SELECT id, token, used, used_by_agent_id, expires_at, created_at
		FROM registration_tokens
		WHERE token = $1 AND used = false AND expires_at > NOW()
	`

	err := m.db.QueryRow(ctx, query, token).Scan(
		&regToken.ID,
		&regToken.Token,
		&regToken.Used,
		&regToken.UsedByAgentID,
		&regToken.ExpiresAt,
		&regToken.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return regToken, nil
}

// MarkUsed marks a token as used by an agent
func (m *RegistrationTokenModel) MarkUsed(ctx context.Context, tokenID, agentID uuid.UUID) error {
	query := `
		UPDATE registration_tokens
		SET used = true, used_by_agent_id = $2, used_at = NOW()
		WHERE id = $1
	`

	_, err := m.db.Exec(ctx, query, tokenID, agentID)
	return err
}

// CleanupExpired removes expired tokens
func (m *RegistrationTokenModel) CleanupExpired(ctx context.Context) error {
	query := `
		DELETE FROM registration_tokens
		WHERE expires_at < NOW() AND used = false
	`

	_, err := m.db.Exec(ctx, query)
	return err
}