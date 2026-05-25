package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description *string   `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SystemSettingsModel struct {
	db *pgxpool.Pool
}

func NewSystemSettingsModel(db *pgxpool.Pool) *SystemSettingsModel {
	return &SystemSettingsModel{db: db}
}

func (m *SystemSettingsModel) Get(ctx context.Context, key string) (*SystemSetting, error) {
	setting := &SystemSetting{}
	query := `
		SELECT key, value, description, updated_at
		FROM system_settings
		WHERE key = $1
	`

	if err := m.db.QueryRow(ctx, query, key).Scan(
		&setting.Key,
		&setting.Value,
		&setting.Description,
		&setting.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return setting, nil
}

func (m *SystemSettingsModel) Upsert(ctx context.Context, key, value, description string) error {
	query := `
		INSERT INTO system_settings (key, value, description, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			description = COALESCE(system_settings.description, EXCLUDED.description),
			updated_at = NOW()
	`

	_, err := m.db.Exec(ctx, query, key, value, description)
	return err
}

