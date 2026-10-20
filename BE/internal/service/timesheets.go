package service

import (
	"context"

	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/repository"
)

type TimesheetService struct{ repo repository.TimesheetRepo }

func NewTimesheetService(repo repository.TimesheetRepo) *TimesheetService {
	return &TimesheetService{repo: repo}
}

func (s *TimesheetService) Issues(ctx context.Context, params dto.TimesheetIssueParams) (*dto.TimesheetIssuesResponse, error) {
	return s.repo.Issues(ctx, params)
}

func (s *TimesheetService) Options(ctx context.Context, params dto.TimesheetOptionsParams) (*dto.TimesheetOptionsResponse, error) {
	return s.repo.Options(ctx, params)
}

func (s *TimesheetService) Grid(ctx context.Context, params dto.TimesheetReadParams) (*dto.TimesheetGridResponse, error) {
	return s.repo.Grid(ctx, params)
}

func (s *TimesheetService) Worklogs(ctx context.Context, params dto.TimesheetReadParams) (*dto.TimesheetWorklogsResponse, error) {
	return s.repo.Worklogs(ctx, params)
}
