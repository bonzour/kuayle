package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

type effortFixture struct {
	db                                             *sqlx.DB
	repo                                           *EffortRepository
	ctx                                            context.Context
	ws, owner, member, guest, team, status, closed uuid.UUID
	issues                                         []uuid.UUID
	dsn                                            string
}

func newEffortFixture(t *testing.T) *effortFixture {
	t.Helper()
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	admin, err := sql.Open("pgx", raw)
	require.NoError(t, err)
	name := "kuayle_t02_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(`CREATE DATABASE "` + name + `"`)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = admin.Exec(`DROP DATABASE "` + name + `" WITH (FORCE)`); _ = admin.Close() })
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	parsed.Path = "/" + name
	m, err := migrate.New("file://../../migrations", parsed.String())
	require.NoError(t, err)
	require.NoError(t, m.Up())
	_, err = m.Close()
	require.NoError(t, err)
	db, err := Open(parsed.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	f := &effortFixture{db: db, repo: NewEffortRepository(db), ws: uuid.New(), owner: uuid.New(), member: uuid.New(), guest: uuid.New(), team: uuid.New(), status: uuid.New(), closed: uuid.New(), issues: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}}
	f.dsn = parsed.String()
	f.ctx = domain.WithAccess(context.Background(), domain.Access{UserID: f.owner, WorkspaceID: f.ws})
	for _, user := range []uuid.UUID{f.owner, f.member, f.guest} {
		f.exec(t, `INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Effort test','hash')`, user, user.String()+"@example.test")
	}
	f.exec(t, `INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'T02',$2,$3)`, f.ws, f.ws.String(), f.owner)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM workspaces WHERE id=$1`, f.ws)
		for _, user := range []uuid.UUID{f.owner, f.member, f.guest} {
			_, _ = db.Exec(`DELETE FROM users WHERE id=$1`, user)
		}
	})
	for user, role := range map[uuid.UUID]string{f.owner: "owner", f.member: "member", f.guest: "guest"} {
		f.exec(t, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, f.ws, user, role)
	}
	f.exec(t, `INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Effort team','T02')`, f.team, f.ws)
	f.exec(t, `INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started'),($3,$2,'Custom closed','custom-closed','completed')`, f.status, f.team, f.closed)
	for n, id := range f.issues {
		f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id,assignee_id) VALUES($1,$2,$3,$4,$5,'Effort issue',$6,$7,$8)`, id, f.ws, f.team, n+1, "T02-"+id.String()[:8], f.owner, f.status, f.member)
	}
	return f
}
func (f *effortFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	_, err := f.db.Exec(q, args...)
	require.NoError(t, err)
}
func mutation(rev int64) dto.EffortMutationRequest {
	return dto.EffortMutationRequest{RequestID: uuid.NewString(), ExpectedRevision: &rev}
}
func minutes(n int64) *int64 { return &n }
func (f *effortFixture) create(rev int64, owner uuid.UUID, date string, duration int64) dto.CreateEffortWorklogRequest {
	return dto.CreateEffortWorklogRequest{EffortMutationRequest: mutation(rev), OwnerUserID: owner.String(), WorkDate: domain.EffortDate(date), DurationMinutes: duration}
}
func effortCode(t *testing.T, err error, code string) {
	t.Helper()
	var e *EffortError
	require.ErrorAs(t, err, &e)
	require.Equal(t, code, e.Code)
}
func (f *effortFixture) counts(t *testing.T) (worklogs, events, operations int) {
	t.Helper()
	require.NoError(t, f.db.Get(&worklogs, `SELECT COUNT(*) FROM effort_worklogs WHERE workspace_id=$1`, f.ws))
	require.NoError(t, f.db.Get(&events, `SELECT COUNT(*) FROM effort_events WHERE workspace_id=$1`, f.ws))
	require.NoError(t, f.db.Get(&operations, `SELECT COUNT(*) FROM effort_operations WHERE workspace_id=$1`, f.ws))
	return
}
func (f *effortFixture) planning(t *testing.T, issue uuid.UUID) (domain.EffortPlanningSnapshot, int64) {
	t.Helper()
	tx, err := f.db.BeginTxx(f.ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	p, rev, err := effortPlanningTx(f.ctx, tx, f.ws, issue)
	require.NoError(t, err)
	return p, rev
}

func TestEffortCommandsPostgres(t *testing.T) {
	t.Run("planning worklog effects revisions and persistent replay", func(t *testing.T) {
		f := newEffortFixture(t)
		issue := f.issues[0]
		estimate := dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(0), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: minutes(480)}}
		result, err := f.repo.SetEstimate(f.ctx, f.ws, issue, estimate)
		require.NoError(t, err)
		require.EqualValues(t, 1, result.Receipt.Revision)
		initial, _ := f.planning(t, issue)
		require.EqualValues(t, 480, *initial.EstimateMinutes)
		require.EqualValues(t, 480, *initial.RemainingMinutes)
		req := f.create(1, f.owner, "2026-01-01", 120)
		created, err := f.repo.CreateWorklog(f.ctx, f.ws, issue, req)
		require.NoError(t, err)
		worklog := uuid.MustParse(created.Receipt.ResourceID)
		p, rev := f.planning(t, issue)
		require.EqualValues(t, 2, rev)
		require.EqualValues(t, 360, *p.RemainingMinutes)
		replay, err := NewEffortRepository(f.db).CreateWorklog(f.ctx, f.ws, issue, req)
		require.NoError(t, err)
		require.True(t, replay.Replay)
		require.Equal(t, created.Receipt, replay.Receipt)
		altered := req
		altered.DurationMinutes++
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, issue, altered)
		effortCode(t, err, dto.EffortRequestConflict)
		update := dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(2), WorkDate: "2026-01-02", DurationMinutes: 180}
		_, err = f.repo.UpdateWorklog(f.ctx, f.ws, issue, worklog, update)
		require.NoError(t, err)
		p, _ = f.planning(t, issue)
		require.EqualValues(t, 360, *p.RemainingMinutes)
		deletion := dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(3)}
		deleted, err := f.repo.DeleteWorklog(f.ctx, f.ws, issue, worklog, deletion)
		require.NoError(t, err)
		replay, err = f.repo.DeleteWorklog(f.ctx, f.ws, issue, worklog, deletion)
		require.NoError(t, err)
		require.True(t, replay.Replay)
		require.Equal(t, deleted.Receipt, replay.Receipt)
		replay, err = f.repo.UpdateWorklog(f.ctx, f.ws, issue, worklog, update)
		require.NoError(t, err)
		require.True(t, replay.Replay)
		require.EqualValues(t, 3, replay.Receipt.Revision)

		_, err = f.repo.UpdateWorklog(f.ctx, f.ws, issue, worklog, dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(3), WorkDate: "2026-01-02", DurationMinutes: 60})
		effortCode(t, err, dto.EffortRevisionConflict)
		p, rev = f.planning(t, issue)
		require.EqualValues(t, 4, rev)
		require.EqualValues(t, 360, *p.RemainingMinutes)
		w, e, o := f.counts(t)
		require.Equal(t, 0, w)
		require.Equal(t, 6, e)
		require.Equal(t, 4, o)
		f.exec(t, `DELETE FROM issues WHERE id=$1`, issue)
		_, err = f.repo.DeleteWorklog(f.ctx, f.ws, issue, worklog, deletion)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
	t.Run("history and receipt failures roll back every effect", func(t *testing.T) {
		for _, table := range []string{"effort_events", "effort_operations"} {
			t.Run(table, func(t *testing.T) {
				f := newEffortFixture(t)
				issue := f.issues[0]
				_, err := f.repo.SetRemaining(f.ctx, f.ws, issue, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(300)})
				require.NoError(t, err)
				w, e, o := f.counts(t)
				name := "t02_fail_" + strings.ReplaceAll(uuid.NewString(), "-", "")
				f.exec(t, `CREATE FUNCTION `+name+`() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.issue_id='`+issue.String()+`'::uuid THEN RAISE EXCEPTION 'injected failure'; END IF; RETURN NEW; END $$`)
				f.exec(t, `CREATE TRIGGER `+name+` BEFORE INSERT ON `+table+` FOR EACH ROW EXECUTE FUNCTION `+name+`()`)
				t.Cleanup(func() {
					_, _ = f.db.Exec(`DROP TRIGGER IF EXISTS ` + name + ` ON ` + table)
					_, _ = f.db.Exec(`DROP FUNCTION IF EXISTS ` + name + `()`)
				})
				req := f.create(1, f.owner, "2026-01-01", 60)
				_, err = f.repo.CreateWorklog(f.ctx, f.ws, issue, req)
				require.Error(t, err)
				aw, ae, ao := f.counts(t)
				require.Equal(t, []int{w, e, o}, []int{aw, ae, ao})
				p, rev := f.planning(t, issue)
				require.EqualValues(t, 1, rev)
				require.EqualValues(t, 300, *p.RemainingMinutes)
				f.exec(t, `DROP TRIGGER `+name+` ON `+table)
				_, err = f.repo.CreateWorklog(f.ctx, f.ws, issue, req)
				require.NoError(t, err, "same request may retry after rollback")
			})
		}
	})
	t.Run("authorization immutability dates and numeric validation", func(t *testing.T) {
		f := newEffortFixture(t)
		issue := f.issues[0]
		member := domain.WithAccess(f.ctx, domain.Access{UserID: f.member, WorkspaceID: f.ws})
		_, err := f.repo.SetEstimate(member, f.ws, issue, dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(0), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: minutes(0)}})
		require.NoError(t, err)
		f.exec(t, `UPDATE issues SET assignee_id=NULL WHERE id=$1`, issue)
		_, err = f.repo.SetRemaining(member, f.ws, issue, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(1), RemainingMinutes: minutes(0)})
		require.ErrorIs(t, err, domain.ErrEffortForbidden)
		_, err = f.repo.CreateWorklog(member, f.ws, issue, f.create(1, f.owner, "2026-01-01", 60))
		require.ErrorIs(t, err, domain.ErrEffortForbidden)
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, issue, f.create(1, f.guest, "2026-01-01", 60))
		require.ErrorIs(t, err, domain.ErrEffortForbidden)
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, issue, f.create(1, f.owner, "9999-01-01", 60))
		effortCode(t, err, "VALIDATION_ERROR")
		req := f.create(1, f.owner, "2026-01-01", 60)
		description := strings.Repeat("界", 2001)
		req.Description = &description
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, issue, req)
		effortCode(t, err, "VALIDATION_ERROR")
		created, err := f.repo.CreateWorklog(f.ctx, f.ws, issue, f.create(1, f.member, "2026-01-01", 60))
		require.NoError(t, err)
		f.exec(t, `UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
		worklog := uuid.MustParse(created.Receipt.ResourceID)
		_, err = f.repo.DeleteWorklog(member, f.ws, issue, worklog, dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(2)})
		require.ErrorIs(t, err, domain.ErrEffortForbidden)
		_, err = f.repo.DeleteWorklog(f.ctx, f.ws, issue, worklog, dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(2)})
		require.NoError(t, err, "admin may correct historical guest work")
	})
}

// Each worker owns a separate pooled connection and PID, allowing tests to
// observe actual PostgreSQL blocking rather than guess timing with sleeps.
func (f *effortFixture) worker(t *testing.T) (*EffortRepository, int) {
	t.Helper()
	u, err := url.Parse(f.dsn)
	require.NoError(t, err)
	q := u.Query()
	q.Set("application_name", "effort_t02_"+uuid.NewString())
	u.RawQuery = q.Encode()
	db, err := Open(u.String())
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	var pid int
	require.NoError(t, db.Get(&pid, `SELECT pg_backend_pid()`))
	return NewEffortRepository(db), pid
}
func waitEffortBlocked(t *testing.T, db *sqlx.DB, pid, blocker int) {
	t.Helper()
	require.Eventually(t, func() bool {
		var blocked bool
		err := db.Get(&blocked, `SELECT $1=ANY(pg_blocking_pids($2))`, blocker, pid)
		return err == nil && blocked
	}, 5*time.Second, 10*time.Millisecond, "worker must reach the expected lock")
}

type effortOutcome struct {
	result *EffortMutationResult
	err    error
}

func runEffort(fn func() (*EffortMutationResult, error)) <-chan effortOutcome {
	ch := make(chan effortOutcome, 1)
	go func() { r, e := fn(); ch <- effortOutcome{r, e} }()
	return ch
}
func receiveEffort(t *testing.T, ch <-chan effortOutcome) effortOutcome {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(8 * time.Second):
		t.Fatal("effort operation did not complete (possible deadlock)")
		return effortOutcome{}
	}
}
func txPID(t *testing.T, tx *sqlx.Tx) int {
	t.Helper()
	var pid int
	require.NoError(t, tx.Get(&pid, `SELECT pg_backend_pid()`))
	return pid
}

func TestEffortConcurrencyPostgres(t *testing.T) {
	t.Run("empty daily budget serializes different issues", func(t *testing.T) {
		f := newEffortFixture(t)
		a, ap := f.worker(t)
		b, bp := f.worker(t)
		hold, err := beginEffortTx(f.ctx, f.db, f.ws, false)
		require.NoError(t, err)
		defer hold.Rollback()
		require.NoError(t, lockEffortBudgets(f.ctx, hold, f.ws, f.owner, "2026-01-01"))
		pid := txPID(t, hold)
		ca := runEffort(func() (*EffortMutationResult, error) {
			return a.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(0, f.owner, "2026-01-01", 800))
		})
		waitEffortBlocked(t, f.db, ap, pid)
		cb := runEffort(func() (*EffortMutationResult, error) {
			return b.CreateWorklog(f.ctx, f.ws, f.issues[1], f.create(0, f.owner, "2026-01-01", 800))
		})
		// The second writer can queue behind the first advisory waiter.
		require.Eventually(t, func() bool {
			var waiting bool
			_ = f.db.Get(&waiting, `SELECT cardinality(pg_blocking_pids($1))>0`, bp)
			return waiting
		}, 5*time.Second, 10*time.Millisecond)
		require.NoError(t, hold.Commit())
		ra, rb := receiveEffort(t, ca), receiveEffort(t, cb)
		require.True(t, (ra.err == nil) != (rb.err == nil))
		if ra.err != nil {
			effortCode(t, ra.err, dto.EffortDailyLimit)
		} else {
			effortCode(t, rb.err, dto.EffortDailyLimit)
		}
		w, e, o := f.counts(t)
		require.Equal(t, []int{1, 1, 1}, []int{w, e, o})
	})
	t.Run("same attempt persists once across connections", func(t *testing.T) {
		f := newEffortFixture(t)
		a, ap := f.worker(t)
		b, _ := f.worker(t)
		req := f.create(0, f.owner, "2026-01-01", 60)
		hold, err := f.db.BeginTxx(f.ctx, nil)
		require.NoError(t, err)
		defer hold.Rollback()
		_, err = hold.Exec(`SELECT id FROM issues WHERE id=$1 FOR UPDATE`, f.issues[0])
		require.NoError(t, err)
		ca := runEffort(func() (*EffortMutationResult, error) { return a.CreateWorklog(f.ctx, f.ws, f.issues[0], req) })
		waitEffortBlocked(t, f.db, ap, txPID(t, hold))
		cb := runEffort(func() (*EffortMutationResult, error) { return b.CreateWorklog(f.ctx, f.ws, f.issues[0], req) })
		require.NoError(t, hold.Commit())
		ra, rb := receiveEffort(t, ca), receiveEffort(t, cb)
		require.NoError(t, ra.err)
		require.NoError(t, rb.err)
		require.NotEqual(t, ra.result.Replay, rb.result.Replay)
		require.Equal(t, ra.result.Receipt, rb.result.Receipt)
		w, e, o := f.counts(t)
		require.Equal(t, []int{1, 1, 1}, []int{w, e, o})
	})
	t.Run("different attempts on same revision conflict", func(t *testing.T) {
		f := newEffortFixture(t)
		a, _ := f.worker(t)
		b, _ := f.worker(t)
		ca := runEffort(func() (*EffortMutationResult, error) {
			return a.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(0, f.owner, "2026-01-01", 60))
		})
		cb := runEffort(func() (*EffortMutationResult, error) {
			return b.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(0, f.owner, "2026-01-01", 60))
		})
		ra, rb := receiveEffort(t, ca), receiveEffort(t, cb)
		if ra.err == nil {
			effortCode(t, rb.err, dto.EffortRevisionConflict)
		} else {
			require.NoError(t, rb.err)
			effortCode(t, ra.err, dto.EffortRevisionConflict)
		}
	})
	t.Run("opposite date changes use stable budget order", func(t *testing.T) {
		f := newEffortFixture(t)
		one, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(0, f.owner, "2026-01-01", 600))
		require.NoError(t, err)
		two, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[1], f.create(0, f.owner, "2026-01-02", 600))
		require.NoError(t, err)
		a, ap := f.worker(t)
		b, _ := f.worker(t)
		hold, err := beginEffortTx(f.ctx, f.db, f.ws, false)
		require.NoError(t, err)
		defer hold.Rollback()
		require.NoError(t, lockEffortBudgets(f.ctx, hold, f.ws, f.owner, "2026-01-01"))
		ca := runEffort(func() (*EffortMutationResult, error) {
			return a.UpdateWorklog(f.ctx, f.ws, f.issues[0], uuid.MustParse(one.Receipt.ResourceID), dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: "2026-01-02", DurationMinutes: 600})
		})
		waitEffortBlocked(t, f.db, ap, txPID(t, hold))
		cb := runEffort(func() (*EffortMutationResult, error) {
			return b.UpdateWorklog(f.ctx, f.ws, f.issues[1], uuid.MustParse(two.Receipt.ResourceID), dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: "2026-01-01", DurationMinutes: 600})
		})
		require.NoError(t, hold.Commit())
		require.NoError(t, receiveEffort(t, ca).err)
		require.NoError(t, receiveEffort(t, cb).err)
	})
	for _, kind := range []string{"role", "workspace", "assignment", "privacy", "team", "disabled"} {
		t.Run("queued write rechecks "+kind, func(t *testing.T) {
			f := newEffortFixture(t)
			worker, pid := f.worker(t)
			member := domain.WithAccess(f.ctx, domain.Access{UserID: f.member, WorkspaceID: f.ws})
			if kind == "team" {
				f.exec(t, `UPDATE teams SET is_private=TRUE WHERE id=$1`, f.team)
				f.exec(t, `INSERT INTO team_members(team_id,user_id) VALUES($1,$2)`, f.team, f.member)
			}
			hold, err := f.db.BeginTxx(f.ctx, nil)
			require.NoError(t, err)
			defer hold.Rollback()
			switch kind {
			case "role":
				_, err = hold.Exec(`UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
			case "team":
				_, err = hold.Exec(`DELETE FROM team_members WHERE team_id=$1 AND user_id=$2`, f.team, f.member)
			case "workspace":
				_, err = hold.Exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
			case "assignment":
				require.NoError(t, lockEffortAssignmentIssue(f.ctx, hold, f.issues[0]))
				_, err = hold.Exec(`UPDATE issues SET assignee_id=NULL WHERE id=$1`, f.issues[0])
			case "privacy":
				_, err = hold.Exec(`UPDATE teams SET is_private=TRUE WHERE id=$1`, f.team)
			case "disabled":
				_, err = hold.Exec(`SELECT pg_advisory_xact_lock(hashtextextended($1,470))`, f.ws.String())
				require.NoError(t, err)
				_, err = hold.Exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled) VALUES($1,FALSE)`, f.ws)
			}
			require.NoError(t, err)
			ch := runEffort(func() (*EffortMutationResult, error) {
				return worker.SetRemaining(member, f.ws, f.issues[0], dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(60)})
			})
			waitEffortBlocked(t, f.db, pid, txPID(t, hold))
			require.NoError(t, hold.Commit())
			result := receiveEffort(t, ch)
			require.Error(t, result.err)
			if kind == "disabled" {
				effortCode(t, result.err, dto.EffortDisabled)
			} else {
				require.True(t, errors.Is(result.err, domain.ErrEffortForbidden) || errors.Is(result.err, sql.ErrNoRows), fmt.Sprint(result.err))
			}
			w, e, o := f.counts(t)
			require.Equal(t, []int{0, 0, 0}, []int{w, e, o})
		})
	}
}

func TestEffortDailyBudgetPrivacyPostgres(t *testing.T) {
	f := newEffortFixture(t)
	hiddenTeam, hiddenIssue, hiddenStatus := uuid.New(), uuid.New(), uuid.New()
	f.exec(t, `INSERT INTO teams(id,workspace_id,name,key,is_private) VALUES($1,$2,'Hidden','HID',TRUE)`, hiddenTeam, f.ws)
	f.exec(t, `INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Hidden open','open','started')`, hiddenStatus, hiddenTeam)
	f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id) VALUES($1,$2,$3,1,'HID-1','Hidden',$4,$5)`, hiddenIssue, f.ws, hiddenTeam, f.owner, hiddenStatus)
	f.exec(t, `INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes) VALUES($1,$2,$3,$4,'2026-01-01',900)`, f.ws, hiddenIssue, f.member, f.owner)
	member := domain.WithAccess(f.ctx, domain.Access{UserID: f.member, WorkspaceID: f.ws})
	_, err := f.repo.CreateWorklog(member, f.ws, f.issues[0], f.create(0, f.member, "2026-01-01", 541))
	effortCode(t, err, dto.EffortDailyLimit)
	require.NotContains(t, err.Error(), hiddenIssue.String())
	require.NotContains(t, err.Error(), "900")
	req := f.create(0, f.member, "2026-01-01", 540)
	one, err := f.repo.CreateWorklog(member, f.ws, f.issues[0], req)
	require.NoError(t, err)
	_, err = f.repo.UpdateWorklog(member, f.ws, f.issues[0], uuid.MustParse(one.Receipt.ResourceID), dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: "2026-01-01", DurationMinutes: 541})
	effortCode(t, err, dto.EffortDailyLimit)
	// A rejected move must retain the original work date and both daily totals.
	two, err := f.repo.CreateWorklog(member, f.ws, f.issues[1], f.create(0, f.member, "2026-01-02", 60))
	require.NoError(t, err)
	_, err = f.repo.UpdateWorklog(member, f.ws, f.issues[1], uuid.MustParse(two.Receipt.ResourceID), dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: "2026-01-01", DurationMinutes: 60})
	effortCode(t, err, dto.EffortDailyLimit)
	var date domain.EffortDate
	require.NoError(t, f.db.Get(&date, `SELECT work_date FROM effort_worklogs WHERE id=$1`, two.Receipt.ResourceID))
	require.Equal(t, domain.EffortDate("2026-01-02"), date)
	// Historical delete snapshots preserve ownership for member replays too.
	deletion := dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(1)}
	_, err = f.repo.DeleteWorklog(member, f.ws, f.issues[1], uuid.MustParse(two.Receipt.ResourceID), deletion)
	require.NoError(t, err)
	replay, err := f.repo.DeleteWorklog(member, f.ws, f.issues[1], uuid.MustParse(two.Receipt.ResourceID), deletion)
	require.NoError(t, err)
	require.True(t, replay.Replay)
}
