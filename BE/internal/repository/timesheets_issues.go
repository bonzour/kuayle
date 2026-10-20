package repository

import (
	"context"
	"encoding/json"

	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/lib/pq"
)

// The workspace, visible suggestions, count and capabilities share one snapshot.
// Suggestions need no worklog and do not inherit the grid's reporting filters.
func (r *TimesheetRepository) Issues(ctx context.Context, p dto.TimesheetIssueParams) (*dto.TimesheetIssuesResponse, error) {
	if !domain.CanReadEfforts(ctx) {
		return nil, domain.ErrEffortForbidden
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	access, _ := domain.AccessFromContext(ctx)
	slugs := domain.TokenWorkspaces(ctx)
	var raw []byte
	err := r.db.QueryRowContext(ctx, `WITH authorized AS MATERIALIZED (
 SELECT w.id,w.name,w.slug,COALESCE(s.enabled,TRUE) AS efforts_enabled,
 COALESCE(s.timezone,'Europe/Rome') AS timezone,
 (statement_timestamp() AT TIME ZONE COALESCE(s.timezone,'Europe/Rome'))::date::text AS today,
 `+workspaceAdmin(ctx, "w.id")+` AS can_administer,
 (COALESCE(s.enabled,TRUE) AND m.role IN ('owner','admin','member') AND $5) AS can_record_work
 FROM workspaces w JOIN workspace_members m ON m.workspace_id=w.id AND m.user_id=$1
 LEFT JOIN workspace_effort_settings s ON s.workspace_id=w.id
 WHERE w.id=$2 AND `+effortWorkspaceVisible(ctx, "w.id")+`
 AND (NOT $3 OR w.slug=ANY($4::text[]))
 ), visible AS MATERIALIZED (
 SELECT i.id,i.identifier_text AS identifier,i.title,i.updated_at,
 jsonb_build_object('id',t.id,'workspace_id',t.workspace_id,'name',t.name,'key',t.key,'icon',t.icon,'color',t.color) AS team
 FROM issues i JOIN authorized w ON w.id=i.workspace_id JOIN teams t ON t.id=i.team_id
 WHERE `+teamVisible(ctx, "i.team_id")+`
 AND ($6::text='' OR strpos(lower(i.identifier_text),lower($6))>0 OR strpos(lower(i.title),lower($6))>0)
 ), page AS (
 SELECT * FROM visible ORDER BY updated_at DESC,id DESC LIMIT 50 OFFSET $7
 ) SELECT jsonb_build_object('workspace',to_jsonb(w),
 'total_count',(SELECT COUNT(*) FROM visible),
 'data',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',id,'identifier',identifier,'title',title,'team',team)
 ORDER BY updated_at DESC,id DESC) FROM page),'[]'::jsonb)) FROM authorized w`,
		access.UserID, p.WorkspaceID, slugs != nil, pq.Array(slugs), domain.CheckEffortToken(ctx) == nil, p.Search, int64(p.Page-1)*domain.EffortPageSize).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var result dto.TimesheetIssuesResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	result.Page = p.Page
	result.PerPage = domain.EffortPageSize
	result.HasMore = int64(p.Page)*domain.EffortPageSize < int64(result.TotalCount)
	return &result, nil
}
