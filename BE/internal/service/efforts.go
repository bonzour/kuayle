package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/repository"
)

type EffortService struct{ repo repository.EffortRepo }

func NewEffortService(repo repository.EffortRepo) *EffortService { return &EffortService{repo: repo} }

func (s *EffortService) Settings(ctx context.Context, workspaceID uuid.UUID) (*dto.EffortSettingsResponse, error) {
	return s.repo.Settings(ctx, workspaceID)
}

func (s *EffortService) Summary(ctx context.Context, workspaceID uuid.UUID, identifier string) (*dto.EffortSummaryResponse, error) {
	return s.repo.Summary(ctx, workspaceID, identifier)
}

func (s *EffortService) Worklogs(ctx context.Context, workspaceID uuid.UUID, identifier string, page int) (*dto.ListResponse[domain.EffortWorklog], error) {
	return s.repo.Worklogs(ctx, workspaceID, identifier, page)
}
