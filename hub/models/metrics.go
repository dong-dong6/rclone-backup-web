package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentMetric struct {
	ID           uuid.UUID `json:"id"`
	AgentID      uuid.UUID `json:"agent_id"`
	Hostname     string    `json:"hostname"`
	Platform     string    `json:"platform"`
	AgentVersion string    `json:"agent_version"`

	CPUUsage float64 `json:"cpu_usage"`

	MemoryTotal int64   `json:"memory_total"`
	MemoryUsed  int64   `json:"memory_used"`
	MemoryUsage float64 `json:"memory_usage"`
	SwapTotal   int64   `json:"swap_total"`
	SwapUsed    int64   `json:"swap_used"`

	DiskTotal int64   `json:"disk_total"`
	DiskUsed  int64   `json:"disk_used"`
	DiskUsage float64 `json:"disk_usage"`

	NetworkRxBytes int64 `json:"network_rx_bytes"`
	NetworkTxBytes int64 `json:"network_tx_bytes"`
	NetworkRxRate  int64 `json:"network_rx_rate"`
	NetworkTxRate  int64 `json:"network_tx_rate"`

	TCPConnections int `json:"tcp_connections"`
	UDPConnections int `json:"udp_connections"`

	ProcessCount int       `json:"process_count"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type MetricsModel struct {
	db *pgxpool.Pool
}

func NewMetricsModel(db *pgxpool.Pool) *MetricsModel {
	return &MetricsModel{db: db}
}

func (m *MetricsModel) Create(ctx context.Context, metric *AgentMetric) error {
	if metric.ID == uuid.Nil {
		metric.ID = uuid.New()
	}

	query := `
		INSERT INTO agent_metrics (
			id, agent_id, hostname, platform, agent_version,
			cpu_usage,
			memory_total, memory_used, memory_usage, swap_total, swap_used,
			disk_total, disk_used, disk_usage,
			network_rx_bytes, network_tx_bytes, network_rx_rate, network_tx_rate,
			tcp_connections, udp_connections, process_count
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21
		)
		RETURNING recorded_at
	`

	if err := m.db.QueryRow(ctx, query,
		metric.ID,
		metric.AgentID,
		metric.Hostname,
		metric.Platform,
		metric.AgentVersion,
		metric.CPUUsage,
		metric.MemoryTotal,
		metric.MemoryUsed,
		metric.MemoryUsage,
		metric.SwapTotal,
		metric.SwapUsed,
		metric.DiskTotal,
		metric.DiskUsed,
		metric.DiskUsage,
		metric.NetworkRxBytes,
		metric.NetworkTxBytes,
		metric.NetworkRxRate,
		metric.NetworkTxRate,
		metric.TCPConnections,
		metric.UDPConnections,
		metric.ProcessCount,
	).Scan(&metric.RecordedAt); err != nil {
		return fmt.Errorf("failed to insert agent metric: %w", err)
	}

	return nil
}

func (m *MetricsModel) GetLatest(ctx context.Context, agentID uuid.UUID) (*AgentMetric, error) {
	query := `
		SELECT id, agent_id, hostname, platform, agent_version,
			cpu_usage,
			memory_total, memory_used, memory_usage, swap_total, swap_used,
			disk_total, disk_used, disk_usage,
			network_rx_bytes, network_tx_bytes, network_rx_rate, network_tx_rate,
			tcp_connections, udp_connections, process_count, recorded_at
		FROM agent_metrics
		WHERE agent_id = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`

	metric := &AgentMetric{}
	if err := m.db.QueryRow(ctx, query, agentID).Scan(
		&metric.ID,
		&metric.AgentID,
		&metric.Hostname,
		&metric.Platform,
		&metric.AgentVersion,
		&metric.CPUUsage,
		&metric.MemoryTotal,
		&metric.MemoryUsed,
		&metric.MemoryUsage,
		&metric.SwapTotal,
		&metric.SwapUsed,
		&metric.DiskTotal,
		&metric.DiskUsed,
		&metric.DiskUsage,
		&metric.NetworkRxBytes,
		&metric.NetworkTxBytes,
		&metric.NetworkRxRate,
		&metric.NetworkTxRate,
		&metric.TCPConnections,
		&metric.UDPConnections,
		&metric.ProcessCount,
		&metric.RecordedAt,
	); err != nil {
		return nil, err
	}

	return metric, nil
}

func (m *MetricsModel) GetHistory(ctx context.Context, agentID uuid.UUID, start, end time.Time) ([]*AgentMetric, error) {
	query := `
		SELECT id, agent_id, hostname, platform, agent_version,
			cpu_usage,
			memory_total, memory_used, memory_usage, swap_total, swap_used,
			disk_total, disk_used, disk_usage,
			network_rx_bytes, network_tx_bytes, network_rx_rate, network_tx_rate,
			tcp_connections, udp_connections, process_count, recorded_at
		FROM agent_metrics
		WHERE agent_id = $1 AND recorded_at BETWEEN $2 AND $3
		ORDER BY recorded_at ASC
	`

	rows, err := m.db.Query(ctx, query, agentID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make([]*AgentMetric, 0)
	for rows.Next() {
		metric := &AgentMetric{}
		if err := rows.Scan(
			&metric.ID,
			&metric.AgentID,
			&metric.Hostname,
			&metric.Platform,
			&metric.AgentVersion,
			&metric.CPUUsage,
			&metric.MemoryTotal,
			&metric.MemoryUsed,
			&metric.MemoryUsage,
			&metric.SwapTotal,
			&metric.SwapUsed,
			&metric.DiskTotal,
			&metric.DiskUsed,
			&metric.DiskUsage,
			&metric.NetworkRxBytes,
			&metric.NetworkTxBytes,
			&metric.NetworkRxRate,
			&metric.NetworkTxRate,
			&metric.TCPConnections,
			&metric.UDPConnections,
			&metric.ProcessCount,
			&metric.RecordedAt,
		); err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (m *MetricsModel) Cleanup(ctx context.Context, retention time.Duration) error {
	if retention <= 0 {
		return nil
	}

	cutoff := time.Now().Add(-retention)
	_, err := m.db.Exec(ctx, `
		DELETE FROM agent_metrics
		WHERE recorded_at < $1
	`, cutoff)
	return err
}
