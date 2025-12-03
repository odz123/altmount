-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- Performance Optimization Indexes
-- ============================================================================

-- Covering index for queue item claim operation (reduces disk I/O)
CREATE INDEX IF NOT EXISTS idx_queue_claim_covering ON import_queue(status, started_at, priority, created_at, id)
WHERE status = 'pending';

-- Index for efficient user API key validation (used in stream authentication)
CREATE INDEX IF NOT EXISTS idx_users_api_key_active ON users(api_key)
WHERE api_key IS NOT NULL AND api_key != '';

-- Composite index for health check scheduling queries
CREATE INDEX IF NOT EXISTS idx_file_health_schedule ON file_health(status, scheduled_check_at)
WHERE status NOT IN ('healthy', 'corrupted');

-- Index for efficient repair notification queries
CREATE INDEX IF NOT EXISTS idx_file_health_repair ON file_health(status, repair_retry_count, max_repair_retries)
WHERE status = 'repair_triggered';

-- Optimize queue stats calculation with covering index
CREATE INDEX IF NOT EXISTS idx_queue_stats_calc ON import_queue(status, started_at, completed_at)
WHERE status = 'completed' AND started_at IS NOT NULL AND completed_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_queue_stats_calc;
DROP INDEX IF EXISTS idx_file_health_repair;
DROP INDEX IF EXISTS idx_file_health_schedule;
DROP INDEX IF EXISTS idx_users_api_key_active;
DROP INDEX IF EXISTS idx_queue_claim_covering;

-- +goose StatementEnd
