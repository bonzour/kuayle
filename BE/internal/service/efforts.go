package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/realtime"
	"github.com/kuayle/kuayle-backend/internal/repository"
)

type EffortService struct {
	repo repository.EffortRepo
	hub  *realtime.Hub
}

func NewEffortService(repo repository.EffortRepo, hubs ...*realtime.Hub) *EffortService {
	s := &EffortService{repo: repo}
	if len(hubs) > 0 {
		s.hub = hubs[0]
	}
	return s
}

func (s *EffortService) Settings(ctx context.Context, workspaceID uuid.UUID) (*dto.EffortSettingsResponse, error) {
	return s.repo.Settings(ctx, workspaceID)
}

func (s *EffortService) Summary(ctx context.Context, workspaceID uuid.UUID, identifier string) (*dto.EffortSummaryResponse, error) {
	return s.repo.Summary(ctx, workspaceID, identifier)
}

func (s *EffortService) Worklogs(ctx context.Context, workspaceID uuid.UUID, identifier string, page int) (*dto.ListResponse[domain.EffortWorklog], error) {
	return s.repo.Worklogs(ctx, workspaceID, identifier, page)
}

// Replays return the previous receipt without publishing another mutation.
func (s *EffortService) committed(ws uuid.UUID, result *repository.EffortMutationResult, err error, resources ...string) (*dto.EffortMutationReceipt, error) {
	if err != nil {
		return nil, err
	}
	if !result.Replay && s.hub != nil {
		if len(resources) == 0 {
			resources = []string{"issues"}
		}
		s.hub.Broadcast(ws, realtime.Event{Type: "app.refresh", Payload: map[string]any{"resources": resources, "source": "efforts"}})
	}
	return &result.Receipt, nil
}

func (s *EffortService) SetEstimate(ctx context.Context, ws, issue uuid.UUID, req dto.SetEffortEstimateRequest) (*dto.EffortMutationReceipt, error) {
	result, err := s.repo.SetEstimate(ctx, ws, issue, req)
	return s.committed(ws, result, err)
}

func (s *EffortService) SetRemaining(ctx context.Context, ws, issue uuid.UUID, req dto.SetEffortRemainingRequest) (*dto.EffortMutationReceipt, error) {
	result, err := s.repo.SetRemaining(ctx, ws, issue, req)
	return s.committed(ws, result, err)
}

func (s *EffortService) CreateWorklog(ctx context.Context, ws, issue uuid.UUID, req dto.CreateEffortWorklogRequest) (*dto.EffortMutationReceipt, error) {
	result, err := s.repo.CreateWorklog(ctx, ws, issue, req)
	return s.committed(ws, result, err)
}

func (s *EffortService) UpdateWorklog(ctx context.Context, ws, issue, worklog uuid.UUID, req dto.UpdateEffortWorklogRequest) (*dto.EffortMutationReceipt, error) {
	result, err := s.repo.UpdateWorklog(ctx, ws, issue, worklog, req)
	return s.committed(ws, result, err)
}

func (s *EffortService) DeleteWorklog(ctx context.Context, ws, issue, worklog uuid.UUID, req dto.DeleteEffortWorklogRequest) (*dto.EffortMutationReceipt, error) {
	result, err := s.repo.DeleteWorklog(ctx, ws, issue, worklog, req)
	return s.committed(ws, result, err)
}

func (s *EffortService) SetSettings(ctx context.Context, ws uuid.UUID, req dto.SetEffortSettingsRequest) (*dto.EffortMutationReceipt, error) {
	result, err := s.repo.SetSettings(ctx, ws, req)
	return s.committed(ws, result, err, "workspace", "issues")
}
