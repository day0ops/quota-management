package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
)

// AuditFilter holds filter parameters for listing audit log entries.
type AuditFilter struct {
	EntityType string
	EntityID   string
	Action     string
	OrgID      string
	TeamID     string
}

// CreateAuditLog creates a new audit log entry.
func (r *Repository) CreateAuditLog(ctx context.Context, entry *models.AuditLogEntry) error {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()

	var metadataJSON []byte
	var err error
	if entry.Metadata != nil {
		metadataJSON, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal audit metadata: %w", err)
		}
	}

	query := `
		INSERT INTO audit_log (id, entity_type, entity_id, action, actor_user_id, actor_email, org_id, team_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		entry.ID, entry.EntityType, entry.EntityID, entry.Action,
		entry.ActorUserID, entry.ActorEmail, entry.OrgID, entry.TeamID,
		metadataJSON, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// ListAuditLogs returns audit log entries with optional filtering and pagination.
func (r *Repository) ListAuditLogs(ctx context.Context, filter AuditFilter, offset, limit int) ([]models.AuditLogEntry, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.EntityType != "" {
		where += fmt.Sprintf(" AND entity_type = $%d", argIdx)
		args = append(args, filter.EntityType)
		argIdx++
	}

	if filter.EntityID != "" {
		where += fmt.Sprintf(" AND entity_id = $%d", argIdx)
		args = append(args, filter.EntityID)
		argIdx++
	}

	if filter.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, filter.Action)
		argIdx++
	}

	if filter.OrgID != "" {
		where += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, filter.OrgID)
		argIdx++
	}

	if filter.TeamID != "" {
		where += fmt.Sprintf(" AND team_id = $%d", argIdx)
		args = append(args, filter.TeamID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", where)
	var totalCount int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, entity_type, entity_id, action, actor_user_id, actor_email, org_id, team_id, metadata, created_at
		FROM audit_log
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	dataArgs := append(args, limit, offset)
	rows, err := r.db.Pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var entries []models.AuditLogEntry
	for rows.Next() {
		var e models.AuditLogEntry
		var metadataJSON []byte
		err := rows.Scan(
			&e.ID, &e.EntityType, &e.EntityID, &e.Action,
			&e.ActorUserID, &e.ActorEmail, &e.OrgID, &e.TeamID,
			&metadataJSON, &e.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal audit metadata: %w", err)
			}
		}
		entries = append(entries, e)
	}

	return entries, totalCount, nil
}

// DeleteOldAuditLogs deletes audit log entries older than retentionDays days.
func (r *Repository) DeleteOldAuditLogs(ctx context.Context, retentionDays int) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < NOW() - ($1 || ' days')::INTERVAL`

	result, err := r.db.Pool.Exec(ctx, query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	return result.RowsAffected(), nil
}
