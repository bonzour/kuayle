package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/lib/pq"
)

type timesheetCell struct {
	Owner       domain.EffortUser            `json:"owner"`
	Workspace   dto.TimesheetWorkspaceOption `json:"workspace"`
	Team        dto.TimesheetTeamOption      `json:"team"`
	ProjectID   *uuid.UUID                   `json:"project_id"`
	ProjectName string                       `json:"project_name"`
	ProjectType string                       `json:"project_type"`
	IssueID     uuid.UUID                    `json:"issue_id"`
	Identifier  string                       `json:"identifier"`
	Title       string                       `json:"title"`
	Date        domain.EffortDate            `json:"date"`
	Minutes     int64                        `json:"minutes"`
}

type timesheetSnapshot struct {
	dto.TimesheetReadResponse
	Options       dto.TimesheetOptionsResponse `json:"options"`
	PeriodHasWork bool                         `json:"period_has_work"`
	People        []domain.EffortUser          `json:"people"`
	Cells         []timesheetCell              `json:"cells"`
	Data          []dto.TimesheetWorklog       `json:"data"`
}

// Both projections use the same authorized worklog relation. Each request is a
// single statement: validation, reference, totals and the page share a snapshot.
func (r *TimesheetRepository) read(ctx context.Context, p dto.TimesheetReadParams, detail bool) (*timesheetSnapshot, error) {
	if !domain.CanReadEfforts(ctx) {
		return nil, domain.ErrEffortForbidden
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	access, _ := domain.AccessFromContext(ctx)
	for _, id := range p.OwnerIDs {
		if p.Mode == "personal" && id != access.UserID {
			return nil, domain.ErrEffortForbidden
		}
	}
	query, args := timesheetReadQuery(ctx, p, detail)
	var adminAllowed, allowed bool
	var raw []byte
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&adminAllowed, &allowed, &raw); err != nil {
		return nil, err
	}
	if p.Mode != "personal" && !adminAllowed {
		return nil, domain.ErrEffortForbidden
	}
	if !allowed {
		return nil, sql.ErrNoRows
	}
	result, err := decodeTimesheetSnapshot(raw)
	if err != nil {
		return nil, err
	}
	result.Mode, result.Period, result.From, result.To = p.Mode, p.Period, p.From, p.To
	result.Scope = result.Options.Scope
	result.Page, result.PerPage = p.Page, domain.EffortPageSize
	result.HasMore = int64(p.Page)*50 < result.TotalCount
	from, _ := time.Parse(time.DateOnly, string(p.From))
	to, _ := time.Parse(time.DateOnly, string(p.To))
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		result.Days = append(result.Days, domain.EffortDate(date.Format(time.DateOnly)))
	}
	result.DailyTotals = completeTimesheetDays(result.Days, result.DailyTotals)
	return result, nil
}

func decodeTimesheetSnapshot(raw []byte) (*timesheetSnapshot, error) {
	var result timesheetSnapshot
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.TotalMinutes > domain.EffortMaxSafeInteger || result.TotalCount > domain.EffortMaxSafeInteger {
		return nil, fmt.Errorf("Timesheet total exceeds the JSON safe integer range")
	}
	return &result, nil
}

// Shared by production reads and the opt-in EXPLAIN measurements.
func timesheetReadQuery(ctx context.Context, p dto.TimesheetReadParams, detail bool) (string, []any) {
	var issue, date any
	if p.IssueID != uuid.Nil {
		issue = p.IssueID
	}
	if p.WorkDate != nil {
		date = string(*p.WorkDate)
	}
	args := append(timesheetScopeArgs(ctx, p.TimesheetOptionsParams), string(p.From), string(p.To), p.ReferenceWorkspaceID, int64(p.Page-1)*50, issue, date, p.OwnerIDs == nil, pq.Array(p.OwnerIDs))
	query := timesheetScopeCTE(ctx) + `, baseline AS MATERIALIZED (
 SELECT w.*, jsonb_build_object('id',w.owner_user_id,'available',owner.id IS NOT NULL,'name',owner.name,'display_name',owner.display_name) AS owner, i.team_id, i.project_id AS raw_project_id, i.identifier_text, i.title,
 to_jsonb(ws) AS workspace, jsonb_build_object('id',t.id,'workspace_id',t.workspace_id,'name',t.name,'key',t.key,'icon',t.icon,'color',t.color) AS team,
 pr.id AS project_id, COALESCE(pr.name,'') AS project_name,
 CASE WHEN i.project_id IS NULL THEN 'no_project' WHEN pr.id IS NULL THEN 'unavailable_project' ELSE 'project' END AS project_type
 FROM effort_worklogs w JOIN admitted ws ON ws.id=w.workspace_id
 LEFT JOIN users owner ON owner.id=w.owner_user_id
 JOIN issues i ON i.id=w.issue_id AND i.workspace_id=w.workspace_id
 JOIN public.teams t ON t.id=i.team_id AND t.workspace_id=i.workspace_id
 LEFT JOIN public.projects pr ON pr.id=i.project_id AND pr.workspace_id=i.workspace_id AND ` + projectVisible(ctx, "pr.workspace_id", "pr.team_id") + `
 WHERE ($5<>'personal' OR w.owner_user_id=$1) AND w.work_date BETWEEN $13::date AND $14::date
 AND ` + teamVisible(ctx, "i.team_id") + `
 ), candidates AS MATERIALIZED (
 SELECT b.* FROM baseline b JOIN chosen c ON c.id=b.workspace_id
 WHERE ($8 OR b.team_id=ANY($9::uuid[]))
 AND ($10 OR (b.raw_project_id IS NULL AND $12) OR b.project_id=ANY($11::uuid[]))
 ), filtered AS MATERIALIZED (
 SELECT b.* FROM candidates b WHERE ($19 OR b.owner_user_id=ANY($20::uuid[]))
 AND ($17::uuid IS NULL OR b.issue_id=$17) AND ($18::date IS NULL OR b.work_date=$18)
 ), daily AS (
 SELECT work_date::text AS date,SUM(duration_minutes)::bigint AS minutes FROM filtered GROUP BY work_date
 )`
	// The root identity determines the page, while each selected root retains
	// all of its cells. These expressions are developer-owned, never request SQL.
	rootID, rootName, rootType, rootWorkspace := "workspace_id::text", "workspace->>'name'", "'workspace'", "workspace_id::text"
	switch p.Mode {
	case "person":
		rootID, rootName, rootType, rootWorkspace = "owner_user_id::text", "COALESCE(NULLIF(owner->>'display_name',''),owner->>'name','')", "'person'", "''"
	case "project":
		rootID, rootName, rootType = "workspace_id::text||':'||project_type||':'||COALESCE(project_id::text,'')", "project_name", "project_type"
	case "team":
		rootID, rootName, rootType = "team_id::text", "team->>'name'", "'team'"
	}
	projection := `, rooted AS MATERIALIZED (
 SELECT b.*,` + rootID + ` AS root_id,` + rootName + ` AS root_name,` + rootType + ` AS root_type,` + rootWorkspace + ` AS root_workspace FROM filtered b
 ), roots AS (
 SELECT DISTINCT root_id,root_name,root_type,root_workspace FROM rooted
 ), root_page AS (
 SELECT root_id FROM roots ORDER BY root_name COLLATE "C",root_type,root_workspace,root_id LIMIT 50 OFFSET $16
 ), cells AS (
 SELECT b.workspace,b.team,b.owner,b.project_id,b.project_name,b.project_type,b.issue_id,b.identifier_text AS identifier,b.title,
 b.work_date::text AS date,SUM(b.duration_minutes)::bigint AS minutes
 FROM rooted b JOIN root_page p ON p.root_id=b.root_id
 GROUP BY b.workspace,b.team,b.owner,b.project_id,b.project_name,b.project_type,b.issue_id,b.identifier_text,b.title,b.work_date
 )`
	count := `(SELECT COUNT(*) FROM roots)`
	page := `'people',COALESCE((SELECT jsonb_agg(owner ORDER BY COALESCE(NULLIF(owner->>'display_name',''),owner->>'name','') COLLATE "C",owner->>'id') FROM (SELECT DISTINCT owner FROM candidates) people),'[]'::jsonb),
 'cells',COALESCE((SELECT jsonb_agg(to_jsonb(c)) FROM cells c),'[]'::jsonb)`
	if detail {
		projection = `, worklog_page AS (
 SELECT * FROM filtered ORDER BY work_date DESC,created_at DESC,id DESC LIMIT 50 OFFSET $16
 )`
		count = `(SELECT COUNT(*) FROM filtered)`
		page = `'data',COALESCE((SELECT jsonb_agg(jsonb_build_object(
 'id',w.id,'issue_id',w.issue_id,'owner',w.owner,
 'operator',jsonb_build_object('id',w.created_by_user_id,'available',operator.id IS NOT NULL,'name',operator.name,'display_name',operator.display_name),
 'work_date',w.work_date::text,'duration_minutes',w.duration_minutes,'description',w.description,'created_at',w.created_at,'updated_at',w.updated_at,
 'workspace',w.workspace,'team',w.team,'issue_identifier',w.identifier_text,'issue_title',w.title
 ) ORDER BY w.work_date DESC,w.created_at DESC,w.id DESC)
 FROM worklog_page w LEFT JOIN users operator ON operator.id=w.created_by_user_id),'[]'::jsonb)`
	}
	query += projection + ` SELECT EXISTS(SELECT 1 FROM memberships WHERE can_administer),
 NOT EXISTS(SELECT 1 FROM unnest($7::uuid[]) requested(id) WHERE NOT EXISTS(SELECT 1 FROM admitted a WHERE a.id=requested.id))
 AND NOT EXISTS(SELECT 1 FROM unnest($9::uuid[]) requested(id) WHERE NOT EXISTS(SELECT 1 FROM teams t WHERE t.id=requested.id))
 AND NOT EXISTS(SELECT 1 FROM unnest($11::uuid[]) requested(id) WHERE NOT EXISTS(SELECT 1 FROM projects p WHERE p.id=requested.id))
 AND EXISTS(SELECT 1 FROM memberships WHERE id=$15)
 AND ($17::uuid IS NULL OR EXISTS(SELECT 1 FROM issues i JOIN chosen c ON c.id=i.workspace_id WHERE i.id=$17 AND ` + teamVisible(ctx, "i.team_id") + `)),
 jsonb_build_object('options',` + timesheetOptionsJSON + `,
 'reference',(SELECT to_jsonb(w) FROM memberships w WHERE id=$15),
 'workspaces',COALESCE((SELECT jsonb_agg(to_jsonb(w) ORDER BY w.name,w.id) FROM chosen w),'[]'::jsonb),
 'total_minutes',COALESCE((SELECT SUM(duration_minutes)::bigint FROM filtered),0),
 'daily_totals',COALESCE((SELECT jsonb_agg(to_jsonb(d) ORDER BY d.date) FROM daily d),'[]'::jsonb),
 'total_count',` + count + `,'period_has_work',EXISTS(SELECT 1 FROM baseline),` + page + `)`
	return query, args
}

func completeTimesheetDays(days []domain.EffortDate, totals []dto.TimesheetDayTotal) []dto.TimesheetDayTotal {
	minutes := make(map[domain.EffortDate]int64, len(totals))
	for _, total := range totals {
		minutes[total.Date] = total.Minutes
	}
	result := make([]dto.TimesheetDayTotal, 0, len(days))
	for _, day := range days {
		result = append(result, dto.TimesheetDayTotal{Date: day, Minutes: minutes[day]})
	}
	return result
}

func (r *TimesheetRepository) Grid(ctx context.Context, p dto.TimesheetReadParams) (*dto.TimesheetGridResponse, error) {
	if p.IssueID != uuid.Nil || p.WorkDate != nil {
		return nil, dto.ErrTimesheetParams
	}
	snapshot, err := r.read(ctx, p, false)
	if err != nil {
		return nil, err
	}
	result := &dto.TimesheetGridResponse{TimesheetReadResponse: snapshot.TimesheetReadResponse, Groups: []*dto.TimesheetGroup{}, People: snapshot.People}
	if snapshot.TotalCount == 0 {
		reason := "period"
		if snapshot.PeriodHasWork {
			reason = "filters"
		}
		result.EmptyReason = &reason
	}
	nodes := map[string]*dto.TimesheetGroup{}
	for _, cell := range snapshot.Cells {
		workspaceID, teamID, issueID := cell.Workspace.ID, cell.Team.ID, cell.IssueID
		projectKey := cell.ProjectType
		if cell.ProjectID != nil {
			projectKey += ":" + cell.ProjectID.String()
		}
		workspace := dto.TimesheetGroup{Key: "workspace:" + workspaceID.String(), Type: "workspace", ID: &workspaceID, Name: cell.Workspace.Name}
		team := dto.TimesheetGroup{Key: "team:" + teamID.String(), Type: "team", ID: &teamID, Name: cell.Team.Name, Team: &cell.Team}
		project := dto.TimesheetGroup{Key: "workspace:" + workspaceID.String() + ":" + projectKey, Type: cell.ProjectType, ID: cell.ProjectID, Name: cell.ProjectName}
		person := dto.TimesheetGroup{Key: "person:" + cell.Owner.ID.String(), Type: "person", ID: &cell.Owner.ID, Name: timesheetPersonName(cell.Owner), Person: &cell.Owner}
		issue := dto.TimesheetGroup{Key: "issue:" + issueID.String(), Type: "issue", ID: &issueID, Name: cell.Title, Identifier: cell.Identifier}
		path := []dto.TimesheetGroup{workspace, team, project, issue}
		switch p.Mode {
		case "person":
			path = []dto.TimesheetGroup{person, workspace, team, project, issue}
		case "project":
			path = []dto.TimesheetGroup{project, team, person, issue}
		case "workspace":
			path = []dto.TimesheetGroup{workspace, team, project, person, issue}
		case "team":
			path = []dto.TimesheetGroup{team, project, person, issue}
		}
		var parent *dto.TimesheetGroup
		for _, item := range path {
			if parent != nil {
				item.Key = parent.Key + "/" + item.Key
			}
			node := nodes[item.Key]
			if node == nil {
				node = &item
				if parent != nil || p.Mode != "person" {
					node.WorkspaceID, node.WorkspaceSlug = &workspaceID, &cell.Workspace.Slug
				}
				node.Children = []*dto.TimesheetGroup{}
				node.DailyTotals = completeTimesheetDays(snapshot.Days, nil)
				nodes[item.Key] = node
				if parent == nil {
					result.Groups = append(result.Groups, node)
				} else {
					parent.Children = append(parent.Children, node)
				}
			}
			node.TotalMinutes += cell.Minutes
			for i := range node.DailyTotals {
				if node.DailyTotals[i].Date == cell.Date {
					node.DailyTotals[i].Minutes += cell.Minutes
					break
				}
			}
			parent = node
		}
	}
	sortTimesheetGroups(result.Groups)
	return result, nil
}

func timesheetPersonName(person domain.EffortUser) string {
	if person.DisplayName != nil && *person.DisplayName != "" {
		return *person.DisplayName
	}
	if person.Name != nil {
		return *person.Name
	}
	return ""
}

func sortTimesheetGroups(groups []*dto.TimesheetGroup) {
	slices.SortFunc(groups, func(a, b *dto.TimesheetGroup) int {
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		if c := strings.Compare(a.Type, b.Type); c != 0 {
			return c
		}
		if a.WorkspaceID != nil && b.WorkspaceID != nil {
			if c := strings.Compare(a.WorkspaceID.String(), b.WorkspaceID.String()); c != 0 {
				return c
			}
		}
		return strings.Compare(a.Key, b.Key)
	})
	for _, group := range groups {
		sortTimesheetGroups(group.Children)
	}
}

func (r *TimesheetRepository) Worklogs(ctx context.Context, p dto.TimesheetReadParams) (*dto.TimesheetWorklogsResponse, error) {
	snapshot, err := r.read(ctx, p, true)
	if err != nil {
		return nil, err
	}
	return &dto.TimesheetWorklogsResponse{TimesheetReadResponse: snapshot.TimesheetReadResponse, Data: snapshot.Data}, nil
}
