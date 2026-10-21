package repository

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

// Explicitly opt in: this creates its own migrated database, then loads 100k
// deterministic records. It has no latency threshold and adds no schema indexes.
func TestTimesheetQueryMeasurements(t *testing.T) {
	if os.Getenv("TIMESHEET_QUERY_MEASUREMENTS") != "1" {
		t.Skip("set TIMESHEET_QUERY_MEASUREMENTS=1 for the 100k dataset")
	}
	f := newEffortFixture(t)
	f.exec(t, `DELETE FROM workspaces`)
	f.exec(t, `DELETE FROM users`)
	f.exec(t, `INSERT INTO users(id,email,name,password_hash) SELECT md5('person'||n)::uuid,'person'||n||'@example.test','Person '||n,'hash' FROM generate_series(1,100) n`)
	f.exec(t, `INSERT INTO workspaces(id,name,slug,owner_id) SELECT md5('workspace'||n)::uuid,'Workspace '||n,'measure-'||n,md5('person1')::uuid FROM generate_series(1,10) n`)
	f.exec(t, `INSERT INTO workspace_members(workspace_id,user_id,role) SELECT w.id,u.id,CASE WHEN u.id=md5('person1')::uuid THEN 'owner' ELSE 'member' END FROM workspaces w CROSS JOIN users u`)
	f.exec(t, `INSERT INTO workspace_effort_settings(workspace_id,timezone) SELECT id,CASE WHEN slug='measure-1' THEN 'Pacific/Kiritimati' ELSE 'America/Adak' END FROM workspaces`)
	f.exec(t, `INSERT INTO teams(id,workspace_id,name,key) SELECT md5('team'||n)::uuid,md5('workspace'||((n-1)/10+1))::uuid,'Team '||n,'T'||n FROM generate_series(1,100) n`)
	f.exec(t, `INSERT INTO team_statuses(id,team_id,name,slug,category) SELECT md5('status'||n)::uuid,md5('team'||n)::uuid,'Open','open','started' FROM generate_series(1,100) n`)
	f.exec(t, `INSERT INTO projects(id,workspace_id,team_id,name) SELECT md5('project'||n)::uuid,md5('workspace'||((n-1)/10+1))::uuid,md5('team'||n)::uuid,'Project '||n FROM generate_series(1,100) n`)
	f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id,project_id)
 SELECT md5('issue'||n)::uuid,md5('workspace'||((n-1)/100+1))::uuid,md5('team'||((n-1)/10+1))::uuid,(n-1)%10+1,'M-'||n,'Measured issue '||n,md5('person1')::uuid,md5('status'||((n-1)/10+1))::uuid,md5('project'||((n-1)/100*10+(n-1)%10+1))::uuid FROM generate_series(1,1000) n`)
	f.exec(t, `INSERT INTO effort_worklogs(id,workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes)
 SELECT md5('worklog'||n)::uuid,md5('workspace'||((n-1)%1000/100+1))::uuid,md5('issue'||((n-1)%1000+1))::uuid,md5('person'||((n-1)/1000+1))::uuid,md5('person1')::uuid,'2025-01-01'::date+((n*37)%365),15*(1+n%8) FROM generate_series(1,100000) n`)
	f.exec(t, `ANALYZE`)
	var version string
	require.NoError(t, f.db.Get(&version, `SELECT version()`))
	t.Logf("Environment: %s; dataset: 10 workspaces, 100 people, 100 teams, 100 projects, 1000 issues, 100000 worklogs, dates 2025; one warm-up, five measured calls", version)
	var actor, reference uuid.UUID
	require.NoError(t, f.db.Get(&actor, `SELECT md5('person1')::uuid`))
	require.NoError(t, f.db.Get(&reference, `SELECT md5('workspace1')::uuid`))
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: actor})
	repo := NewTimesheetRepository(f.db)
	for _, period := range []string{"week", "month"} {
		for _, mode := range []string{"personal", "person", "project", "workspace", "team"} {
			t.Run(period+"/"+mode, func(t *testing.T) {
				p := dto.TimesheetReadParams{TimesheetOptionsParams: dto.TimesheetOptionsParams{Mode: mode, Workspaces: dto.TimesheetSelection{All: true}, Teams: dto.TimesheetSelection{All: true}, Projects: dto.TimesheetSelection{All: true}}, Period: period, From: "2025-10-06", To: "2025-10-12", ReferenceWorkspaceID: reference, Page: 1}
				if period == "month" {
					p.From = "2025-10-01"
					p.To = "2025-10-31"
				}
				_, err := repo.Grid(ctx, p)
				require.NoError(t, err)
				timings := make([]time.Duration, 5)
				for i := range timings {
					start := time.Now()
					grid, err := repo.Grid(ctx, p)
					timings[i] = time.Since(start)
					require.NoError(t, err)
					require.Positive(t, grid.TotalMinutes)
				}
				sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })
				t.Logf("%s/%s end-to-end repository: min=%s median=%s max=%s", period, mode, timings[0], timings[2], timings[4])
				query, args := timesheetReadQuery(ctx, p, false)
				var plan []string
				require.NoError(t, f.db.SelectContext(ctx, &plan, "EXPLAIN (ANALYZE, BUFFERS) "+query, args...))
				t.Logf("%s/%s SQL plan:\n%s", period, mode, strings.Join(plan, "\n"))
			})
		}
	}
}
