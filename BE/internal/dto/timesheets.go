package dto

import (
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
)

var ErrTimesheetParams = errors.New("invalid Timesheet parameters")

var timesheetModes = []string{"personal", "person", "project", "workspace", "team"}

// All and an explicit empty selection are different. IncludeNone is only valid
// for projects and denotes SQL NULL, never an inaccessible project reference.
type TimesheetSelection struct {
	All         bool
	IDs         []uuid.UUID
	IncludeNone bool
}

type TimesheetOptionsParams struct {
	Mode       string
	Workspaces TimesheetSelection
	Teams      TimesheetSelection
	Projects   TimesheetSelection
}

func (p TimesheetOptionsParams) Validate() error {
	if !slices.Contains(timesheetModes, p.Mode) || p.Workspaces.IncludeNone || p.Teams.IncludeNone {
		return ErrTimesheetParams
	}
	for _, selection := range []TimesheetSelection{p.Workspaces, p.Teams, p.Projects} {
		if selection.All && (len(selection.IDs) > 0 || selection.IncludeNone) || slices.Contains(selection.IDs, uuid.Nil) {
			return ErrTimesheetParams
		}
	}
	return nil
}

func ParseTimesheetOptions(values url.Values) (TimesheetOptionsParams, error) {
	p := TimesheetOptionsParams{Mode: "personal"}
	for key, values := range values {
		if !slices.Contains([]string{"mode", "workspace_ids", "team_ids", "project_ids"}, key) || len(values) != 1 {
			return p, ErrTimesheetParams
		}
	}
	if values.Has("mode") {
		p.Mode = values.Get("mode")
	}
	for _, field := range []struct {
		key       string
		selection *TimesheetSelection
	}{
		{"workspace_ids", &p.Workspaces}, {"team_ids", &p.Teams}, {"project_ids", &p.Projects},
	} {
		raw := values.Get(field.key)
		field.selection.All = !values.Has(field.key) || raw == "all"
		field.selection.IDs = []uuid.UUID{}
		if field.selection.All || raw == "" {
			continue
		}
		for _, item := range strings.Split(raw, ",") {
			if field.key == "project_ids" && item == "none" {
				field.selection.IncludeNone = true
				continue
			}
			id, err := uuid.Parse(item)
			if err != nil || id == uuid.Nil {
				return p, ErrTimesheetParams
			}
			if !slices.Contains(field.selection.IDs, id) {
				field.selection.IDs = append(field.selection.IDs, id)
			}
		}
	}
	return p, p.Validate()
}

type TimesheetWorkspaceOption struct {
	ID             uuid.UUID         `json:"id"`
	Name           string            `json:"name"`
	Slug           string            `json:"slug"`
	EffortsEnabled bool              `json:"efforts_enabled"`
	Timezone       string            `json:"timezone"`
	Today          domain.EffortDate `json:"today"`
	CanAdminister  bool              `json:"can_administer"`
	CanRecordWork  bool              `json:"can_record_work"`
}

type TimesheetTeamOption struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	Icon        *string   `json:"icon"`
	Color       *string   `json:"color"`
}

type TimesheetProjectOption struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
}

type TimesheetScope struct {
	WorkspaceIDs     []uuid.UUID `json:"workspace_ids"`
	TeamIDs          []uuid.UUID `json:"team_ids"`
	ProjectIDs       []uuid.UUID `json:"project_ids"`
	IncludeNoProject bool        `json:"include_no_project"`
}

type TimesheetOptionsResponse struct {
	Mode           string                     `json:"mode"`
	AvailableModes []string                   `json:"available_modes"`
	Workspaces     []TimesheetWorkspaceOption `json:"workspaces"`
	Teams          []TimesheetTeamOption      `json:"teams"`
	Projects       []TimesheetProjectOption   `json:"projects"`
	Scope          TimesheetScope             `json:"scope"`
}

type TimesheetReadParams struct {
	TimesheetOptionsParams
	Period               string
	From                 domain.EffortDate
	To                   domain.EffortDate
	ReferenceWorkspaceID uuid.UUID
	OwnerIDs             []uuid.UUID // nil means all; a non-nil empty slice selects nobody.
	Page                 int
	IssueID              uuid.UUID
	WorkDate             *domain.EffortDate
}

func ParseTimesheetRead(values url.Values, detail bool) (TimesheetReadParams, error) {
	p := TimesheetReadParams{Page: 1}
	common := url.Values{}
	for key, items := range values {
		if len(items) != 1 {
			return p, ErrTimesheetParams
		}
		switch key {
		case "mode", "workspace_ids", "team_ids", "project_ids":
			common[key] = items
		case "period", "from", "to", "reference_workspace_id", "owner_user_ids", "page", "per_page":
		case "issue_id", "work_date":
			if !detail {
				return p, ErrTimesheetParams
			}
		default:
			return p, ErrTimesheetParams
		}
	}
	var err error
	if p.TimesheetOptionsParams, err = ParseTimesheetOptions(common); err != nil {
		return p, err
	}
	p.Period = values.Get("period")
	if p.From, err = domain.ParseEffortDate(values.Get("from")); err != nil {
		return p, ErrTimesheetParams
	}
	if p.To, err = domain.ParseEffortDate(values.Get("to")); err != nil {
		return p, ErrTimesheetParams
	}
	if p.ReferenceWorkspaceID, err = uuid.Parse(values.Get("reference_workspace_id")); err != nil {
		return p, ErrTimesheetParams
	}
	if values.Has("page") {
		if p.Page, err = strconv.Atoi(values.Get("page")); err != nil {
			return p, ErrTimesheetParams
		}
	}
	if values.Has("per_page") && values.Get("per_page") != "50" {
		return p, ErrTimesheetParams
	}
	if values.Has("owner_user_ids") {
		rawOwners := values.Get("owner_user_ids")
		if p.Mode != "personal" && rawOwners == "all" {
			// Omission and all have the same administrative meaning.
		} else if p.Mode != "personal" && rawOwners == "" {
			p.OwnerIDs = []uuid.UUID{}
		} else {
			for _, raw := range strings.Split(rawOwners, ",") {
				id, err := uuid.Parse(raw)
				if err != nil || id == uuid.Nil {
					return p, ErrTimesheetParams
				}
				if !slices.Contains(p.OwnerIDs, id) {
					p.OwnerIDs = append(p.OwnerIDs, id)
				}
			}
		}
	}
	if values.Has("issue_id") {
		if p.IssueID, err = uuid.Parse(values.Get("issue_id")); err != nil || p.IssueID == uuid.Nil {
			return p, ErrTimesheetParams
		}
	}
	if values.Has("work_date") {
		date, err := domain.ParseEffortDate(values.Get("work_date"))
		if err != nil {
			return p, ErrTimesheetParams
		}
		p.WorkDate = &date
	}
	return p, p.Validate()
}

func (p TimesheetReadParams) Validate() error {
	if err := p.TimesheetOptionsParams.Validate(); err != nil {
		return err
	}
	if p.ReferenceWorkspaceID == uuid.Nil || p.Page < 1 || int64(p.Page) > domain.EffortMaxSafeInteger/domain.EffortPageSize {
		return ErrTimesheetParams
	}
	if _, err := domain.ParseEffortDate(string(p.From)); err != nil {
		return ErrTimesheetParams
	}
	if _, err := domain.ParseEffortDate(string(p.To)); err != nil {
		return ErrTimesheetParams
	}
	from, _ := time.Parse(time.DateOnly, string(p.From))
	to, _ := time.Parse(time.DateOnly, string(p.To))
	switch p.Period {
	case "week":
		if from.Weekday() != time.Monday || !to.Equal(from.AddDate(0, 0, 6)) {
			return ErrTimesheetParams
		}
	case "month":
		if from.Day() != 1 || !to.Equal(from.AddDate(0, 1, -1)) {
			return ErrTimesheetParams
		}
	default:
		return ErrTimesheetParams
	}
	if slices.Contains(p.OwnerIDs, uuid.Nil) {
		return ErrTimesheetParams
	}
	if p.WorkDate != nil {
		if _, err := domain.ParseEffortDate(string(*p.WorkDate)); err != nil || *p.WorkDate < p.From || *p.WorkDate > p.To {
			return ErrTimesheetParams
		}
	}
	return nil
}

type TimesheetDayTotal struct {
	Date    domain.EffortDate `json:"date"`
	Minutes int64             `json:"minutes"`
}

type TimesheetReadResponse struct {
	Mode         string                     `json:"mode"`
	Period       string                     `json:"period"`
	From         domain.EffortDate          `json:"from"`
	To           domain.EffortDate          `json:"to"`
	Days         []domain.EffortDate        `json:"days"`
	Reference    TimesheetWorkspaceOption   `json:"reference"`
	Workspaces   []TimesheetWorkspaceOption `json:"workspaces"`
	Scope        TimesheetScope             `json:"scope"`
	DailyTotals  []TimesheetDayTotal        `json:"daily_totals"`
	TotalMinutes int64                      `json:"total_minutes"`
	TotalCount   int64                      `json:"total_count"`
	Page         int                        `json:"page"`
	PerPage      int                        `json:"per_page"`
	HasMore      bool                       `json:"has_more"`
}

type TimesheetGroup struct {
	Key           string               `json:"key"`
	Type          string               `json:"type"`
	ID            *uuid.UUID           `json:"id"`
	Name          string               `json:"name"`
	WorkspaceID   *uuid.UUID           `json:"workspace_id"`
	WorkspaceSlug *string              `json:"workspace_slug"`
	Identifier    string               `json:"identifier,omitempty"`
	Team          *TimesheetTeamOption `json:"team,omitempty"`
	Person        *domain.EffortUser   `json:"person,omitempty"`
	DailyTotals   []TimesheetDayTotal  `json:"daily_totals"`
	TotalMinutes  int64                `json:"total_minutes"`
	Children      []*TimesheetGroup    `json:"children"`
}

type TimesheetGridResponse struct {
	TimesheetReadResponse
	Groups      []*TimesheetGroup   `json:"groups"`
	People      []domain.EffortUser `json:"people"`
	EmptyReason *string             `json:"empty_reason"`
}

type TimesheetWorklog struct {
	domain.EffortWorklog
	Workspace       TimesheetWorkspaceOption `json:"workspace"`
	Team            TimesheetTeamOption      `json:"team"`
	IssueIdentifier string                   `json:"issue_identifier"`
	IssueTitle      string                   `json:"issue_title"`
}

type TimesheetWorklogsResponse struct {
	TimesheetReadResponse
	Data []TimesheetWorklog `json:"data"`
}
