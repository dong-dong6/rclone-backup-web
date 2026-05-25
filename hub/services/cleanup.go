package services

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/rclone-backup-web/hub/models"
)

// StartMetricsCleanup starts a ticker that periodically deletes old metrics.
func StartMetricsCleanup(ctx context.Context, metricsModel *models.MetricsModel, settingsModel *models.SystemSettingsModel) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				retention := getRetentionDuration(ctx, settingsModel)
				if retention <= 0 {
					continue
				}
				if err := metricsModel.Cleanup(ctx, retention); err != nil {
					log.Printf("metrics cleanup failed: %v", err)
				}
			}
		}
	}()
}

func getRetentionDuration(ctx context.Context, settingsModel *models.SystemSettingsModel) time.Duration {
	setting, err := settingsModel.Get(ctx, "metrics.retention_hours")
	if err != nil {
		return 168 * time.Hour
	}

	hours, err := strconv.ParseFloat(setting.Value, 64)
	if err != nil || hours <= 0 {
		return 168 * time.Hour
	}

	return time.Duration(hours * float64(time.Hour))
}
