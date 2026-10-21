package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/realtime"
	"github.com/kuayle/kuayle-backend/internal/repository"
	"github.com/kuayle/kuayle-backend/pkg/sanitize"
)

type ProjectService struct {
	projectRepo repository.ProjectRepo
	hub         *realtime.Hub
}

func NewProjectService(projectRepo repository.ProjectRepo, hubs ...*realtime.Hub) *ProjectService {
	s := &ProjectService{projectRepo: projectRepo}
	if len(hubs) > 0 {
		s.hub = hubs[0]
	}
	return s
}

func (s *ProjectService) refresh(workspaceID uuid.UUID) {
	if s.hub != nil {
		s.hub.Broadcast(workspaceID, realtime.Event{Type: "app.refresh", Payload: map[string]any{"resources": []string{"projects", "issues"}}})
	}
}

func (s *ProjectService) Create(ctx context.Context, workspaceID uuid.UUID, req dto.CreateProjectRequest) (*domain.Project, error) {
	req.Name = sanitize.StripHTML(req.Name)
	if req.Description != nil {
		clean := sanitize.SanitizeHTML(*req.Description)
		req.Description = &clean
	}

	project := &domain.Project{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Description: req.Description,
		Status:      domain.ProjectStatusPlanned,
	}
	if req.TeamID != nil {
		tid, _ := uuid.Parse(*req.TeamID)
		project.TeamID = &tid
	}
	if req.LeadID != nil {
		lid, _ := uuid.Parse(*req.LeadID)
		project.LeadID = &lid
	}
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}
	s.refresh(workspaceID)
	return project, nil
}

func (s *ProjectService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return s.projectRepo.GetByID(ctx, id)
}

func (s *ProjectService) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Project, error) {
	return s.projectRepo.ListByWorkspace(ctx, workspaceID)
}

func (s *ProjectService) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]domain.Project, error) {
	return s.projectRepo.ListByTeam(ctx, teamID)
}

func (s *ProjectService) Update(ctx context.Context, workspaceID, id uuid.UUID, req dto.UpdateProjectRequest) (*domain.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil || project == nil {
		return nil, fmt.Errorf("project not found")
	}
	if project.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("project not found")
	}

	if req.Name != nil {
		clean := sanitize.StripHTML(*req.Name)
		req.Name = &clean
		project.Name = clean
	}
	if req.Description != nil {
		clean := sanitize.SanitizeHTML(*req.Description)
		req.Description = &clean
		project.Description = req.Description
	}
	if req.Status != nil {
		project.Status = domain.ProjectStatus(*req.Status)
	}
	if req.TeamID != nil {
		tid, _ := uuid.Parse(*req.TeamID)
		project.TeamID = &tid
	}
	if req.LeadID != nil {
		lid, _ := uuid.Parse(*req.LeadID)
		project.LeadID = &lid
	}
	if req.SortOrder != nil {
		project.SortOrder = *req.SortOrder
	}

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}
	s.refresh(workspaceID)
	return project, nil
}

func (s *ProjectService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil || project == nil {
		return fmt.Errorf("project not found")
	}
	if project.WorkspaceID != workspaceID {
		return fmt.Errorf("project not found")
	}
	if err := s.projectRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.refresh(workspaceID)
	return nil
}

func (s *ProjectService) GetStats(ctx context.Context, projectID uuid.UUID) (*dto.ProjectProgressResponse, error) {
	total, completed, cancelled, err := s.projectRepo.IssueStats(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &dto.ProjectProgressResponse{
		Total:     total,
		Completed: completed,
		Cancelled: cancelled,
	}, nil
}
