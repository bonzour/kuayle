package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/lib/pq"
)

type TimesheetRepository struct{ db *sqlx.DB }

func NewTimesheetRepository(db *sqlx.DB) *TimesheetRepository { return &TimesheetRepository{db: db} }

// Validation, current authorization, capabilities and all options share one SQL
// statement. No pre-resolved workspace IDs are trusted as authorization.
func (r *TimesheetRepository) Options(ctx context.Context, p dto.TimesheetOptionsParams) (*dto.TimesheetOptionsResponse, error) {
	if !domain.CanReadEfforts(ctx) {
		return nil, domain.ErrEffortForbidden
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	var adminAllowed, selectionAllowed bool
	var raw []byte
	err := r.db.QueryRowContext(ctx, timesheetScopeCTE(ctx)+` SELECT EXISTS(SELECT 1 FROM memberships WHERE can_administer),
 NOT EXISTS(SELECT 1 FROM unnest($7::uuid[]) requested(id) WHERE NOT EXISTS(SELECT 1 FROM admitted a WHERE a.id=requested.id))
 AND NOT EXISTS(SELECT 1 FROM unnest($9::uuid[]) requested(id) WHERE NOT EXISTS(SELECT 1 FROM teams t WHERE t.id=requested.id))
 AND NOT EXISTS(SELECT 1 FROM unnest($11::uuid[]) requested(id) WHERE NOT EXISTS(SELECT 1 FROM projects p WHERE p.id=requested.id)),
 `+timesheetOptionsJSON,
		timesheetScopeArgs(ctx, p)...,
	).Scan(&adminAllowed, &selectionAllowed, &raw)
	if err != nil {
		return nil, err
	}
	if p.Mode != "personal" && !adminAllowed {
		return nil, domain.ErrEffortForbidden
	}
	if !selectionAllowed {
		return nil, sql.ErrNoRows
	}
	var result dto.TimesheetOptionsResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	result.AvailableModes = []string{"personal"}
	if adminAllowed {
		result.AvailableModes = append(result.AvailableModes, "person", "project", "workspace", "team")
	}
	return &result, nil
}

func timesheetScopeCTE(ctx context.Context) string {
	return `WITH memberships AS MATERIALIZED (
 SELECT w.id,w.name,w.slug,COALESCE(s.enabled,TRUE) AS efforts_enabled,
 COALESCE(s.timezone,'Europe/Rome') AS timezone,
 (statement_timestamp() AT TIME ZONE COALESCE(s.timezone,'Europe/Rome'))::date::text AS today,
 ` + workspaceAdmin(ctx, "w.id") + ` AS can_administer,
 (COALESCE(s.enabled,TRUE) AND m.role IN ('owner','admin','member') AND $3) AS can_record_work
 FROM workspaces w JOIN workspace_members m ON m.workspace_id=w.id
 LEFT JOIN workspace_effort_settings s ON s.workspace_id=w.id
 WHERE m.user_id=$1 AND ` + effortWorkspaceVisible(ctx, "w.id") + `
 AND (NOT $2 OR w.slug=ANY($4::text[]))
 ), admitted AS MATERIALIZED (
 SELECT * FROM memberships WHERE $5='personal' OR can_administer
 ), chosen AS MATERIALIZED (
 SELECT * FROM admitted WHERE $6 OR id=ANY($7::uuid[])
 ), teams AS MATERIALIZED (
 SELECT t.id,t.workspace_id,t.name,t.key,t.icon,t.color FROM teams t JOIN chosen w ON w.id=t.workspace_id
 WHERE ` + teamVisible(ctx, "t.id") + `
 ), projects AS MATERIALIZED (
 SELECT p.id,p.workspace_id,p.name FROM projects p JOIN chosen w ON w.id=p.workspace_id
 WHERE ` + projectVisible(ctx, "p.workspace_id", "p.team_id") + `
 )`
}

const timesheetOptionsJSON = ` jsonb_build_object(
 'mode',$5::text,
 'workspaces',COALESCE((SELECT jsonb_agg(to_jsonb(w) ORDER BY w.name,w.id) FROM admitted w),'[]'::jsonb),
 'teams',COALESCE((SELECT jsonb_agg(to_jsonb(t) ORDER BY t.name,t.workspace_id,t.id) FROM teams t),'[]'::jsonb),
 'projects',COALESCE((SELECT jsonb_agg(to_jsonb(p) ORDER BY p.name,p.workspace_id,p.id) FROM projects p),'[]'::jsonb),
 'scope',jsonb_build_object(
 'workspace_ids',COALESCE((SELECT jsonb_agg(id ORDER BY id) FROM chosen),'[]'::jsonb),
 'team_ids',COALESCE((SELECT jsonb_agg(id ORDER BY id) FROM teams WHERE $8 OR id=ANY($9::uuid[])),'[]'::jsonb),
 'project_ids',COALESCE((SELECT jsonb_agg(id ORDER BY id) FROM projects WHERE $10 OR id=ANY($11::uuid[])),'[]'::jsonb),
 'include_no_project',$12::boolean))`

func timesheetScopeArgs(ctx context.Context, p dto.TimesheetOptionsParams) []any {
	slugs := domain.TokenWorkspaces(ctx)
	access, _ := domain.AccessFromContext(ctx)
	return []any{access.UserID, slugs != nil, domain.CheckEffortToken(ctx) == nil, pq.Array(slugs), p.Mode,
		p.Workspaces.All, pq.Array(p.Workspaces.IDs), p.Teams.All, pq.Array(p.Teams.IDs),
		p.Projects.All, pq.Array(p.Projects.IDs), p.Projects.All || p.Projects.IncludeNone}
}
