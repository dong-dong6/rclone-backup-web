-- ============================================
-- Add remote connection test status columns
-- ============================================

ALTER TABLE rclone_remotes
  ADD COLUMN IF NOT EXISTS last_test_at TIMESTAMP WITH TIME ZONE,
  ADD COLUMN IF NOT EXISTS last_test_success BOOLEAN,
  ADD COLUMN IF NOT EXISTS last_test_message TEXT,
  ADD COLUMN IF NOT EXISTS last_test_error TEXT,
  ADD COLUMN IF NOT EXISTS last_test_duration_ms BIGINT;

CREATE INDEX IF NOT EXISTS idx_rclone_remotes_last_test_at ON rclone_remotes(last_test_at DESC);
