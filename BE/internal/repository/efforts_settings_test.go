package repository

import (
	"context"
	"net/url"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestEffortSettingsPostgres(t *testing.T) {
	t.Run("defaults and saved configuration survive reopening and migrations", func(t *testing.T) {
		f := newEffortFixture(t)
		initial, err := f.repo.Settings(f.ctx, f.ws)
		require.NoError(t, err)
		require.Equal(t, domain.EffortSettings{Enabled: true, Timezone: "Europe/Rome"}, initial.EffortSettings)
		enabled := false
		req := dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(0), Enabled: &enabled, Timezone: "UTC"}
		first, err := f.repo.SetSettings(f.ctx, f.ws, req)
		require.NoError(t, err)
		u, err := url.Parse(f.dsn)
		require.NoError(t, err)
		m, err := migrate.New("file://../../migrations", u.String())
		require.NoError(t, err)
		require.ErrorIs(t, m.Up(), migrate.ErrNoChange)
		_, err = m.Close()
		require.NoError(t, err)
		worker, _ := f.worker(t)
		saved, err := worker.Settings(f.ctx, f.ws)
		require.NoError(t, err)
		require.Equal(t, domain.EffortSettings{Enabled: false, Timezone: "UTC", Revision: 1}, saved.EffortSettings)
		replay, err := worker.SetSettings(f.ctx, f.ws, req)
		require.NoError(t, err)
		require.True(t, replay.Replay)
		require.Equal(t, first.Receipt, replay.Receipt)
		newWorkspace := uuid.New()
		f.exec(t, `INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'After EP01',$2,$3)`, newWorkspace, newWorkspace.String(), f.owner)
		f.exec(t, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, newWorkspace, f.owner)
		ctx := domain.WithAccess(context.Background(), domain.Access{UserID: f.owner, WorkspaceID: newWorkspace})
		fresh, err := worker.Settings(ctx, newWorkspace)
		require.NoError(t, err)
		require.Equal(t, initial.EffortSettings, fresh.EffortSettings)
	})

	t.Run("workspace today ignores server zone and changing zone preserves future history", func(t *testing.T) {
		f := newEffortFixture(t)
		worker, _ := f.worker(t)
		_, err := worker.db.Exec(`SET TIME ZONE 'Etc/GMT+12'`)
		require.NoError(t, err)
		enabled := true
		_, err = worker.SetSettings(f.ctx, f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(0), Enabled: &enabled, Timezone: "Pacific/Kiritimati"})
		require.NoError(t, err)
		ahead, err := worker.Settings(f.ctx, f.ws)
		require.NoError(t, err)
		var expected domain.EffortDate
		require.NoError(t, worker.db.Get(&expected, `SELECT (clock_timestamp() AT TIME ZONE 'Pacific/Kiritimati')::date`))
		require.Equal(t, expected, ahead.Today)
		created, err := worker.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(0, f.owner, string(ahead.Today), 90))
		require.NoError(t, err)
		_, err = worker.SetSettings(f.ctx, f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(1), Enabled: &enabled, Timezone: "Etc/GMT+12"})
		require.NoError(t, err)
		behind, err := worker.Settings(f.ctx, f.ws)
		require.NoError(t, err)
		require.Less(t, string(behind.Today), string(ahead.Today))
		logs, err := worker.Worklogs(f.ctx, f.ws, "T02-"+f.issues[0].String()[:8], 1)
		require.NoError(t, err)
		require.Len(t, logs.Data, 1)
		require.Equal(t, ahead.Today, logs.Data[0].WorkDate)
		require.EqualValues(t, 90, logs.Data[0].DurationMinutes)
		_, err = worker.UpdateWorklog(f.ctx, f.ws, f.issues[0], uuid.MustParse(created.Receipt.ResourceID), dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: ahead.Today, DurationMinutes: 90})
		effortCode(t, err, "VALIDATION_ERROR")
		_, err = worker.UpdateWorklog(f.ctx, f.ws, f.issues[0], uuid.MustParse(created.Receipt.ResourceID), dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: behind.Today, DurationMinutes: 90})
		require.NoError(t, err)
	})

	t.Run("disabled data stays readable and immutable until atomic reactivation", func(t *testing.T) {
		f := newEffortFixture(t)
		id := f.issues[0]
		_, err := f.repo.SetEstimate(f.ctx, f.ws, id, dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(0), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: minutes(480)}})
		require.NoError(t, err)
		created, err := f.repo.CreateWorklog(f.ctx, f.ws, id, f.create(1, f.owner, "2026-01-01", 90))
		require.NoError(t, err)
		worklog := uuid.MustParse(created.Receipt.ResourceID)
		f.remaining(t, f.issues[1], 240)
		f.settings(t, false, 0)
		for _, command := range []func() (*EffortMutationResult, error){
			func() (*EffortMutationResult, error) {
				return f.repo.SetEstimate(f.ctx, f.ws, id, dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(2), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: minutes(120)}})
			},
			func() (*EffortMutationResult, error) {
				return f.repo.SetRemaining(f.ctx, f.ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(2), RemainingMinutes: minutes(10)})
			},
			func() (*EffortMutationResult, error) {
				return f.repo.CreateWorklog(f.ctx, f.ws, id, f.create(2, f.owner, "2026-01-01", 60))
			},
			func() (*EffortMutationResult, error) {
				return f.repo.UpdateWorklog(f.ctx, f.ws, id, worklog, dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(2), WorkDate: "2026-01-02", DurationMinutes: 60})
			},
			func() (*EffortMutationResult, error) {
				return f.repo.DeleteWorklog(f.ctx, f.ws, id, worklog, dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(2)})
			},
		} {
			_, err := command()
			effortCode(t, err, dto.EffortDisabled)
		}
		r := NewIssueRepository(f.db)
		issue, err := r.GetByID(f.ctx, id)
		require.NoError(t, err)
		issue.Status, issue.StatusID = "custom-closed", &f.closed
		require.NoError(t, r.Update(f.ctx, issue))
		frozen, revision := f.planning(t, id)
		require.EqualValues(t, 480, *frozen.EstimateMinutes)
		require.EqualValues(t, 390, *frozen.RemainingMinutes)
		require.EqualValues(t, 2, revision)
		logs, err := f.repo.Worklogs(f.ctx, f.ws, "T02-"+id.String()[:8], 1)
		require.NoError(t, err)
		require.Len(t, logs.Data, 1)
		require.EqualValues(t, 90, logs.Data[0].DurationMinutes)
		f.settings(t, true, 1)
		reconciled, revision := f.planning(t, id)
		require.EqualValues(t, 480, *reconciled.EstimateMinutes)
		require.Zero(t, *reconciled.RemainingMinutes)
		require.EqualValues(t, 3, revision)
		open, _ := f.planning(t, f.issues[1])
		require.EqualValues(t, 240, *open.RemainingMinutes)
		var actor uuid.UUID
		require.NoError(t, f.db.Get(&actor, `SELECT actor_user_id FROM effort_events WHERE issue_id=$1 AND cause='reactivation' AND before_snapshot->'planning'->>'remaining_minutes'='390' AND after_snapshot->'planning'->>'remaining_minutes'='0'`, id))
		require.Equal(t, f.owner, actor)
		logsAfter, err := f.repo.Worklogs(f.ctx, f.ws, "T02-"+id.String()[:8], 1)
		require.NoError(t, err)
		require.Equal(t, logs.ListResponse, logsAfter.ListResponse)
		require.Greater(t, logsAfter.Revision, logs.Revision)
	})

	t.Run("calendar boundaries follow the configured IANA daylight saving rules", func(t *testing.T) {
		f := newEffortFixture(t)
		for _, tc := range []struct{ instant, date string }{
			{"2026-03-28T23:30:00Z", "2026-03-29"},
			{"2026-03-29T01:30:00Z", "2026-03-29"},
			{"2026-03-29T22:30:00Z", "2026-03-30"},
			{"2026-10-25T00:30:00Z", "2026-10-25"},
			{"2026-10-25T01:30:00Z", "2026-10-25"},
			{"2026-10-25T23:30:00Z", "2026-10-26"},
		} {
			var date domain.EffortDate
			require.NoError(t, f.db.Get(&date, `SELECT ($1::timestamptz AT TIME ZONE COALESCE(s.timezone,'Europe/Rome'))::date FROM workspaces w LEFT JOIN workspace_effort_settings s ON s.workspace_id=w.id WHERE w.id=$2`, tc.instant, f.ws))
			require.Equal(t, tc.date, string(date), tc.instant)
		}
	})

	t.Run("queued settings write rechecks admin role after waiting", func(t *testing.T) {
		f := newEffortFixture(t)
		f.exec(t, `UPDATE workspace_members SET role='admin' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
		worker, pid := f.worker(t)
		hold, err := f.db.BeginTxx(f.ctx, nil)
		require.NoError(t, err)
		defer hold.Rollback()
		_, err = hold.Exec(`UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
		require.NoError(t, err)
		ctx := domain.WithAccess(f.ctx, domain.Access{UserID: f.member, WorkspaceID: f.ws})
		enabled := false
		ch := runEffort(func() (*EffortMutationResult, error) {
			return worker.SetSettings(ctx, f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(0), Enabled: &enabled, Timezone: "UTC"})
		})
		waitEffortBlocked(t, f.db, pid, txPID(t, hold))
		require.NoError(t, hold.Commit())
		require.ErrorIs(t, receiveEffort(t, ch).err, domain.ErrEffortForbidden)
		settings, err := f.repo.Settings(f.ctx, f.ws)
		require.NoError(t, err)
		require.True(t, settings.Enabled)
		require.Zero(t, settings.Revision)
	})

	t.Run("concurrent settings submissions cannot overwrite one another", func(t *testing.T) {
		f := newEffortFixture(t)
		a, _ := f.worker(t)
		b, _ := f.worker(t)
		enabled := false
		change := func(repo *EffortRepository, zone string) (*EffortMutationResult, error) {
			return repo.SetSettings(f.ctx, f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(0), Enabled: &enabled, Timezone: zone})
		}
		ca := runEffort(func() (*EffortMutationResult, error) { return change(a, "UTC") })
		cb := runEffort(func() (*EffortMutationResult, error) { return change(b, "Europe/Rome") })
		ra, rb := receiveEffort(t, ca), receiveEffort(t, cb)
		if ra.err == nil {
			effortCode(t, rb.err, dto.EffortRevisionConflict)
		} else {
			require.NoError(t, rb.err)
			effortCode(t, ra.err, dto.EffortRevisionConflict)
		}
	})
}
