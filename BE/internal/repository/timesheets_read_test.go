package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func timesheetReadParams(t *testing.T, reference uuid.UUID, extra string) dto.TimesheetReadParams {
	t.Helper()
	values, err := url.ParseQuery("period=week&from=2026-01-05&to=2026-01-11&reference_workspace_id=" + reference.String() + extra)
	require.NoError(t, err)
	p, err := dto.ParseTimesheetRead(values, true)
	require.NoError(t, err)
	return p
}

func TestTimesheetPersonalPostgres(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewTimesheetRepository(f.db)
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: f.member})
	project, hiddenTeam, hiddenProject := uuid.New(), uuid.New(), uuid.New()
	f.exec(t, `INSERT INTO teams(id,workspace_id,name,key,is_private) VALUES($1,$2,'Hidden','H',TRUE)`, hiddenTeam, f.ws)
	f.exec(t, `INSERT INTO projects(id,workspace_id,team_id,name) VALUES($1,$2,$3,'Visible project'),($4,$2,$5,'SECRET project')`, project, f.ws, f.team, hiddenProject, hiddenTeam)
	f.exec(t, `UPDATE issues SET project_id=$1 WHERE id=$2`, project, f.issues[0])
	f.exec(t, `UPDATE issues SET parent_id=$1,status_id=$2,assignee_id=NULL WHERE id=$3`, f.issues[0], f.closed, f.issues[1])
	// Simulate a legacy reference that predates private-boundary enforcement.
	f.exec(t, `ALTER TABLE issues DISABLE TRIGGER private_boundary_check`)
	f.exec(t, `UPDATE issues SET project_id=$1 WHERE id=$2`, hiddenProject, f.issues[2])
	f.exec(t, `ALTER TABLE issues ENABLE TRIGGER private_boundary_check`)
	insert := func(issue, owner, operator uuid.UUID, date string, minutes int) uuid.UUID {
		id := uuid.New()
		f.exec(t, `INSERT INTO effort_worklogs(id,workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, f.ws, issue, owner, operator, date, minutes)
		return id
	}
	first := insert(f.issues[0], f.member, f.member, "2026-01-05", 60)
	second := insert(f.issues[0], f.member, f.owner, "2026-01-05", 30)
	insert(f.issues[1], f.member, f.member, "2026-01-05", 30)
	insert(f.issues[0], f.owner, f.member, "2026-01-05", 120)
	insert(f.issues[2], f.member, f.owner, "2026-01-11", 15)
	p := timesheetReadParams(t, f.ws, "")
	grid, err := repo.Grid(ctx, p)
	require.NoError(t, err)
	require.EqualValues(t, 135, grid.TotalMinutes)
	require.Len(t, grid.Days, 7)
	require.EqualValues(t, 120, grid.DailyTotals[0].Minutes)
	require.EqualValues(t, 15, grid.DailyTotals[6].Minutes)
	require.Nil(t, grid.EmptyReason)
	require.Len(t, grid.Groups, 1)
	projects := grid.Groups[0].Children[0].Children
	require.Len(t, projects, 3)
	for _, p := range projects {
		if p.Type == "unavailable_project" {
			require.Nil(t, p.ID)
			require.Empty(t, p.Name)
		}
		if p.Type == "no_project" {
			require.EqualValues(t, 30, p.TotalMinutes)
		}
	}
	detail := p
	detail.IssueID = f.issues[0]
	day := domain.EffortDate("2026-01-05")
	detail.WorkDate = &day
	logs, err := repo.Worklogs(ctx, detail)
	require.NoError(t, err)
	require.EqualValues(t, 90, logs.TotalMinutes)
	require.Len(t, logs.Data, 2)
	require.ElementsMatch(t, []uuid.UUID{first, second}, []uuid.UUID{logs.Data[0].ID, logs.Data[1].ID})
	for _, w := range logs.Data {
		require.Equal(t, f.member, w.Owner.ID)
		require.Equal(t, f.ws, w.Workspace.ID)
	}
	// Month length and leap-day work stay identical across reference timezones.
	monthLog := insert(f.issues[0], f.member, f.owner, "2024-02-29", 40)
	month := p
	month.Period, month.From, month.To = "month", "2024-02-01", "2024-02-29"
	monthly, err := repo.Grid(ctx, month)
	require.NoError(t, err)
	require.Len(t, monthly.Days, 29)
	require.EqualValues(t, 40, monthly.DailyTotals[28].Minutes)
	f.exec(t, `DELETE FROM effort_worklogs WHERE id=$1`, monthLog)
	filtered, err := repo.Grid(ctx, timesheetReadParams(t, f.ws, "&project_ids=none"))
	require.NoError(t, err)
	require.EqualValues(t, 30, filtered.TotalMinutes)
	filtered, err = repo.Grid(ctx, timesheetReadParams(t, f.ws, "&project_ids="))
	require.NoError(t, err)
	require.Equal(t, "filters", *filtered.EmptyReason)
	empty := p
	empty.From, empty.To = "2026-02-02", "2026-02-08"
	filtered, err = repo.Grid(ctx, empty)
	require.NoError(t, err)
	require.Equal(t, "period", *filtered.EmptyReason)
	f.exec(t, `INSERT INTO workspace_effort_settings(workspace_id,enabled,timezone) VALUES($1,FALSE,'Pacific/Kiritimati')`, f.ws)
	f.exec(t, `UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
	grid, err = repo.Grid(ctx, p)
	require.NoError(t, err)
	require.EqualValues(t, 135, grid.TotalMinutes)
	require.False(t, grid.Workspaces[0].CanRecordWork)
	require.Equal(t, "Pacific/Kiritimati", grid.Reference.Timezone)
	privateIssue, privateStatus := uuid.New(), uuid.New()
	f.exec(t, `INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started')`, privateStatus, hiddenTeam)
	f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id) VALUES($1,$2,$3,1,'H-1','SECRET issue',$4,$5)`, privateIssue, f.ws, hiddenTeam, f.owner, privateStatus)
	insert(privateIssue, f.member, f.member, "2026-01-05", 60)
	grid, err = repo.Grid(ctx, p)
	require.NoError(t, err)
	require.EqualValues(t, 135, grid.TotalMinutes)
	detail.IssueID = privateIssue
	_, err = repo.Worklogs(ctx, detail)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = repo.Grid(ctx, timesheetReadParams(t, f.ws, "&owner_user_ids="+f.owner.String()))
	require.ErrorIs(t, err, domain.ErrEffortForbidden)
	_, err = repo.Grid(domain.WithTokenScopes(ctx, []string{}), p)
	require.ErrorIs(t, err, domain.ErrEffortForbidden)
	_, err = repo.Grid(domain.WithTokenWorkspaces(ctx, []string{}), p)
	require.ErrorIs(t, err, sql.ErrNoRows)
	ghost := uuid.New()
	ghostLog := insert(f.issues[0], f.member, ghost, "2026-01-05", 1)
	detail.IssueID = f.issues[0]
	logs, err = repo.Worklogs(ctx, detail)
	require.NoError(t, err)
	for _, log := range logs.Data {
		if log.ID == ghostLog {
			require.Equal(t, ghost, log.Operator.ID)
			require.False(t, log.Operator.Available)
			require.Nil(t, log.Operator.Name)
		}
	}
	// Deleting current records removes their minutes; events never recreate them.
	f.exec(t, `DELETE FROM effort_worklogs WHERE id IN ($1,$2)`, ghostLog, second)
	grid, err = repo.Grid(ctx, p)
	require.NoError(t, err)
	require.EqualValues(t, 105, grid.TotalMinutes)
	f.exec(t, `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
	_, err = repo.Grid(ctx, p)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestTimesheetPaginationPostgres(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewTimesheetRepository(f.db)
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: f.member})
	// Identical timestamps force the UUID tie breaker; all 51 groups have the
	// same label so their IDs must also determine a stable group order.
	f.exec(t, `INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes,created_at)
 SELECT $1,$2,$3,$3,'2026-01-05',1,'2026-01-06T00:00:00Z' FROM generate_series(1,101)`, f.ws, f.issues[0], f.member)
	for n := 0; n < 50; n++ {
		ws, team, status, issue := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		f.exec(t, `INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'T02',$2,$3)`, ws, ws.String(), f.owner)
		f.exec(t, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'member')`, ws, f.member)
		f.exec(t, `INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Same team','S')`, team, ws)
		f.exec(t, `INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started')`, status, team)
		f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id) VALUES($1,$2,$3,1,'S-1','Same issue',$4,$5)`, issue, ws, team, f.owner, status)
		f.exec(t, `INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes) VALUES($1,$2,$3,$3,'2026-01-05',1)`, ws, issue, f.member)
	}
	p := timesheetReadParams(t, f.ws, "")
	seenGroups := map[string]bool{}
	for page := 1; page <= 2; page++ {
		p.Page = page
		grid, err := repo.Grid(ctx, p)
		require.NoError(t, err)
		require.EqualValues(t, 151, grid.TotalMinutes)
		require.EqualValues(t, 51, grid.TotalCount)
		require.Equal(t, page == 1, grid.HasMore)
		for _, g := range grid.Groups {
			require.False(t, seenGroups[g.Key])
			seenGroups[g.Key] = true
		}
	}
	require.Len(t, seenGroups, 51)
	p.IssueID = f.issues[0]
	seen := map[uuid.UUID]bool{}
	for page := 1; page <= 3; page++ {
		p.Page = page
		logs, err := repo.Worklogs(ctx, p)
		require.NoError(t, err)
		require.EqualValues(t, 101, logs.TotalCount)
		require.EqualValues(t, 101, logs.TotalMinutes)
		require.Equal(t, page < 3, logs.HasMore)
		repeat, err := repo.Worklogs(ctx, p)
		require.NoError(t, err)
		require.Equal(t, logs.Data, repeat.Data)
		for _, w := range logs.Data {
			require.False(t, seen[w.ID])
			seen[w.ID] = true
		}
	}
	require.Len(t, seen, 101)
	p.Page = 4
	logs, err := repo.Worklogs(ctx, p)
	require.NoError(t, err)
	require.Empty(t, logs.Data)
	require.EqualValues(t, 101, logs.TotalMinutes)
	// Administrative workspace roots keep the same full totals across pages.
	f.exec(t, `UPDATE workspace_members SET role='admin' WHERE user_id=$1`, f.member)
	p.Mode, p.IssueID = "workspace", uuid.Nil
	adminSeen := map[string]bool{}
	for page := 1; page <= 2; page++ {
		p.Page = page
		grid, err := repo.Grid(ctx, p)
		require.NoError(t, err)
		require.EqualValues(t, 151, grid.TotalMinutes)
		require.EqualValues(t, 51, grid.TotalCount)
		for _, g := range grid.Groups {
			require.False(t, adminSeen[g.Key])
			adminSeen[g.Key] = true
		}
	}
	require.Len(t, adminSeen, 51)
	p.Mode = "personal"
	// A reference outside the selected data remains valid; the PAT ceiling
	// applies independently to it, even when it would contribute no minutes.
	p.IssueID = uuid.Nil
	p.Page = 1
	p.Workspaces = dto.TimesheetSelection{}
	grid, err := repo.Grid(ctx, p)
	require.NoError(t, err)
	require.Empty(t, grid.Groups)
	require.Equal(t, f.ws, grid.Reference.ID)
	_, err = repo.Grid(domain.WithTokenWorkspaces(ctx, []string{"other"}), p)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
