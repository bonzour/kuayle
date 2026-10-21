package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestTimesheetAdministrativePostgres(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewTimesheetRepository(f.db)
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: f.member})
	b, bt, bs, bi, project, bp, sharedTeam := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.exec(t, `UPDATE workspace_members SET role='admin' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
	f.exec(t, `INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'W2',$2,$3)`, b, b.String(), f.owner)
	f.exec(t, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'admin')`, b, f.member)
	f.exec(t, `INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Effort team','B'),($3,$4,'Effort team','S')`, bt, b, sharedTeam, f.ws)
	f.exec(t, `INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started')`, bs, bt)
	f.exec(t, `INSERT INTO projects(id,workspace_id,team_id,name) VALUES($1,$2,$3,'Alfa'),($4,$5,$6,'Alfa')`, project, f.ws, f.team, bp, b, bt)
	f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id,project_id) VALUES($1,$2,$3,1,'B-1','Same issue',$4,$5,$6)`, bi, b, bt, f.owner, bs, bp)
	f.exec(t, `UPDATE issues SET project_id=$1 WHERE id=$2`, project, f.issues[0])
	f.exec(t, `UPDATE issues SET project_id=$1,team_id=$2 WHERE id=$3`, project, sharedTeam, f.issues[2])
	f.exec(t, `UPDATE issues SET parent_id=$1 WHERE id=$2`, f.issues[0], f.issues[1])
	f.exec(t, `INSERT INTO issue_assignees(issue_id,user_id) VALUES($1,$2),($1,$3) ON CONFLICT DO NOTHING`, f.issues[0], f.member, f.guest)
	f.exec(t, `UPDATE projects SET lead_id=$1 WHERE id=$2`, f.member, project)
	insert := func(ws, issue, owner uuid.UUID, minutes int) uuid.UUID {
		id := uuid.New()
		f.exec(t, `INSERT INTO effort_worklogs(id,workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes) VALUES($1,$2,$3,$4,$5,'2026-01-05',$6)`, id, ws, issue, owner, f.owner, minutes)
		return id
	}
	insert(f.ws, f.issues[0], f.member, 60)
	insert(f.ws, f.issues[0], f.member, 30)
	insert(f.ws, f.issues[0], f.guest, 20)
	insert(f.ws, f.issues[2], f.guest, 100)
	insert(f.ws, f.issues[1], f.member, 30)
	insert(b, bi, f.member, 60)
	modes := []string{"person", "project", "workspace", "team"}
	for _, mode := range modes {
		p := timesheetReadParams(t, f.ws, "&mode="+mode)
		grid, err := repo.Grid(ctx, p)
		require.NoError(t, err)
		require.EqualValues(t, 300, grid.TotalMinutes)
		require.EqualValues(t, 300, grid.DailyTotals[0].Minutes)
		require.Len(t, grid.People, 2, "same names never merge distinct account IDs")
		month := p
		month.Period, month.From, month.To = "month", "2026-01-01", "2026-01-31"
		monthly, err := repo.Grid(ctx, month)
		require.NoError(t, err)
		require.Len(t, monthly.Days, 31)
		require.EqualValues(t, 300, monthly.TotalMinutes)

		seen := map[string]bool{}
		var verify func([]*dto.TimesheetGroup, []string) int64
		verify = func(groups []*dto.TimesheetGroup, types []string) int64 {
			var sum int64
			for _, g := range groups {
				require.False(t, seen[g.Key])
				seen[g.Key] = true
				expected := types[0]
				if expected == "project" {
					require.Contains(t, []string{"project", "no_project"}, g.Type)
				} else {
					require.Equal(t, expected, g.Type)
				}
				require.EqualValues(t, g.TotalMinutes, g.DailyTotals[0].Minutes)
				if len(types) > 1 {
					require.EqualValues(t, g.TotalMinutes, verify(g.Children, types[1:]))
				}
				sum += g.TotalMinutes
			}
			return sum
		}
		hierarchy := map[string][]string{"person": {"person", "workspace", "team", "project", "issue"}, "project": {"project", "team", "person", "issue"}, "workspace": {"workspace", "team", "project", "person", "issue"}, "team": {"team", "project", "person", "issue"}}
		require.EqualValues(t, 300, verify(grid.Groups, hierarchy[mode]))
		if mode == "person" {
			require.Nil(t, grid.Groups[0].WorkspaceID)
			require.Nil(t, grid.Groups[0].WorkspaceSlug)
			for _, g := range grid.Groups {
				if *g.ID == f.member {
					require.EqualValues(t, 180, g.TotalMinutes)
					require.Len(t, g.Children, 2)
				}
			}
		}
		for suffix, expected := range map[string]int{"&owner_user_ids=all": 300, "&owner_user_ids=": 0, "&owner_user_ids=" + f.member.String(): 180, "&owner_user_ids=" + uuid.NewString(): 0, "&owner_user_ids=" + f.member.String() + "&project_ids=" + project.String(): 90, "&team_ids=" + sharedTeam.String(): 100, "&project_ids=none": 30} {
			filtered, err := repo.Grid(ctx, timesheetReadParams(t, f.ws, "&mode="+mode+suffix))
			require.NoError(t, err, suffix)
			require.EqualValues(t, expected, filtered.TotalMinutes, suffix)
			if suffix == "&owner_user_ids=" || suffix == "&owner_user_ids="+f.member.String() {
				require.Len(t, filtered.People, 2)
			}
			if expected == 0 {
				require.Equal(t, "filters", *filtered.EmptyReason)
			}
		}
	}
	// A leaf for Anna never includes Bob's registration on the same issue/date.
	p := timesheetReadParams(t, f.ws, "&mode=project&owner_user_ids="+f.member.String())
	p.IssueID = f.issues[0]
	date := domain.EffortDate("2026-01-05")
	p.WorkDate = &date
	logs, err := repo.Worklogs(ctx, p)
	require.NoError(t, err)
	require.Len(t, logs.Data, 2)
	require.EqualValues(t, 90, logs.TotalMinutes)
	for _, w := range logs.Data {
		require.Equal(t, f.member, w.Owner.ID)
		require.Equal(t, f.owner, w.Operator.ID)
	}
	// Membership removal does not remove another person's authorized history.
	f.exec(t, `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.guest)
	ghost1, ghost2 := uuid.New(), uuid.New()
	insert(f.ws, f.issues[0], ghost1, 5)
	insert(f.ws, f.issues[0], ghost2, 7)
	grid, err := repo.Grid(ctx, timesheetReadParams(t, f.ws, "&mode=person"))
	require.NoError(t, err)
	require.Len(t, grid.Groups, 4)
	require.EqualValues(t, 312, grid.TotalMinutes)
	for _, person := range grid.People {
		if person.ID == ghost1 || person.ID == ghost2 {
			require.False(t, person.Available)
			require.Nil(t, person.Name)
			require.Nil(t, person.DisplayName)
		}
	}
	// Current project and team classify every historical date immediately.
	f.exec(t, `UPDATE issues SET project_id=NULL WHERE id=$1`, f.issues[0])
	grid, err = repo.Grid(ctx, timesheetReadParams(t, f.ws, "&mode=project&project_ids=none"))
	require.NoError(t, err)
	require.EqualValues(t, 152, grid.TotalMinutes)
	f.exec(t, `UPDATE issues SET team_id=$1,parent_id=NULL WHERE id=$2`, sharedTeam, f.issues[0])
	grid, err = repo.Grid(ctx, timesheetReadParams(t, f.ws, "&mode=team&team_ids="+sharedTeam.String()))
	require.NoError(t, err)
	require.EqualValues(t, 222, grid.TotalMinutes)
	// Admin A/member B sees no administrative contribution or people from B.
	f.exec(t, `UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, b, f.member)
	for _, mode := range modes {
		grid, err = repo.Grid(ctx, timesheetReadParams(t, b, "&mode="+mode))
		require.NoError(t, err)
		require.EqualValues(t, 252, grid.TotalMinutes)
		require.Equal(t, b, grid.Reference.ID)
		_, err = repo.Grid(ctx, timesheetReadParams(t, f.ws, "&mode="+mode+"&workspace_ids="+f.ws.String()+","+b.String()))
		require.ErrorIs(t, err, sql.ErrNoRows)
	}
	// No role or access propagation through known child IDs or a scoped Access.
	_, err = repo.Worklogs(ctx, timesheetReadParams(t, f.ws, "&mode=team&issue_id="+bi.String()))
	require.ErrorIs(t, err, sql.ErrNoRows)
	for _, denied := range []context.Context{domain.WithTokenScopes(ctx, []string{}), domain.WithAccess(ctx, domain.Access{UserID: f.guest}), domain.PublicContext(ctx)} {
		_, err = repo.Grid(denied, timesheetReadParams(t, f.ws, "&mode=person"))
		require.ErrorIs(t, err, domain.ErrEffortForbidden)
	}
	// Deleting the parent leaves surviving children's own work; no event restores it.
	f.exec(t, `DELETE FROM issues WHERE id=$1`, f.issues[0])
	grid, err = repo.Grid(ctx, timesheetReadParams(t, f.ws, "&mode=workspace"))
	require.NoError(t, err)
	require.EqualValues(t, 130, grid.TotalMinutes)
	f.exec(t, `UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
	_, err = repo.Grid(ctx, timesheetReadParams(t, f.ws, "&mode=person"))
	require.ErrorIs(t, err, domain.ErrEffortForbidden)
}

func TestTimesheetAdministrativePaginationPostgres(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewTimesheetRepository(f.db)
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: f.owner})
	// 51 projects, teams and historical users, plus 101 logs on the first leaf.
	for n := 0; n < 51; n++ {
		team, project, issue, owner := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		f.exec(t, `INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Same',$3)`, team, f.ws, fmt.Sprintf("S%d", n))
		f.exec(t, `INSERT INTO projects(id,workspace_id,team_id,name) VALUES($1,$2,$3,'Same')`, project, f.ws, team)
		f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,project_id,status_id) VALUES($1,$2,$3,$4,$5,'Same',$6,$7,$8)`, issue, f.ws, team, n+1, issue.String(), f.owner, project, f.status)
		count := 1
		if n == 0 {
			count = 101
		}
		f.exec(t, `INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes,created_at) SELECT $1,$2,$3,$4,'2026-01-05',1,'2026-01-06T00:00:00Z' FROM generate_series(1,$5::integer)`, f.ws, issue, owner, f.owner, count)
		if n == 0 {
			p := timesheetReadParams(t, f.ws, "&mode=person&owner_user_ids="+owner.String())
			p.IssueID = issue
			seen := map[uuid.UUID]bool{}
			for page := 1; page <= 3; page++ {
				p.Page = page
				logs, err := repo.Worklogs(ctx, p)
				require.NoError(t, err)
				require.EqualValues(t, 101, logs.TotalMinutes)
				for _, w := range logs.Data {
					require.False(t, seen[w.ID])
					seen[w.ID] = true
				}
			}
			require.Len(t, seen, 101)
		}
	}
	for _, mode := range []string{"person", "project", "team"} {
		p := timesheetReadParams(t, f.ws, "&mode="+mode)
		seen := map[string]bool{}
		for page := 1; page <= 3; page++ {
			p.Page = page
			grid, err := repo.Grid(ctx, p)
			require.NoError(t, err)
			require.EqualValues(t, 151, grid.TotalMinutes)
			require.EqualValues(t, 51, grid.TotalCount)
			require.Len(t, grid.People, 51)
			again, err := repo.Grid(ctx, p)
			require.NoError(t, err)
			require.Equal(t, grid.Groups, again.Groups)
			for _, g := range grid.Groups {
				require.False(t, seen[g.Key])
				seen[g.Key] = true
			}
			if page == 3 {
				require.Empty(t, grid.Groups)
			}
		}
		require.Len(t, seen, 51)
	}
}
