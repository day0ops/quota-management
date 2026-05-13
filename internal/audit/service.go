package audit

import (
	"context"

	"github.com/agentgateway/quota-management/internal/auth"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/models"
	"github.com/rs/zerolog/log"
)

type Service struct {
	repo *db.Repository
}

func NewService(repo *db.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) LogAction(ctx context.Context, entityType, entityID, action string, identity *auth.Identity, metadata map[string]interface{}) {
	entry := &models.AuditLogEntry{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Metadata:   metadata,
	}

	if identity != nil {
		entry.ActorUserID = identity.Subject
		entry.ActorEmail = identity.Email
		entry.OrgID = identity.OrgID
		entry.TeamID = identity.TeamID
	}

	if err := s.repo.CreateAuditLog(ctx, entry); err != nil {
		log.Error().Err(err).
			Str("entity_type", entityType).
			Str("entity_id", entityID).
			Str("action", action).
			Msg("failed to create audit log entry")
	}
}

func (s *Service) CleanupOldEntries(ctx context.Context, retentionDays int) (int64, error) {
	return s.repo.DeleteOldAuditLogs(ctx, retentionDays)
}
