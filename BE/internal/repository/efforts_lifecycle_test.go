package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func (f *effortFixture) remaining(t *testing.T, issue uuid.UUID, n int64) {
	t.Helper()
	_, err := f.repo.SetRemaining(f.ctx, f.ws, issue, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(n)})
	require.NoError(t, err)
}
func (f *effortFixture) settings(t *testing.T, enabled bool, rev int64) *EffortMutationResult {
	t.Helper()
	result, err := f.repo.SetSettings(f.ctx, f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(rev), Enabled: &enabled, Timezone: "Europe/Rome"})
	require.NoError(t, err)
	return result
}
func TestEffortLifecyclePostgres(t *testing.T) {
	for _, kind := range []string{"single", "bulk", "github"} {
		t.Run(kind+" close and reopen", func(t *testing.T) {
			f := newEffortFixture(t)
			issueRepo := NewIssueRepository(f.db)
			for _, id := range f.issues {
				f.remaining(t, id, 300)
			}
			if kind == "bulk" {
				slug := "custom-closed"
				n, err := issueRepo.BulkUpdate(f.ctx, f.ws, f.issues, &slug, nil, nil, &f.closed, nil, false)
				require.NoError(t, err)
				require.Equal(t, 3, n)
			} else {
				ctx := f.ctx
				if kind == "github" {
					ctx = domain.WithGitHubEffortActor(context.Background())
				}
				issue, err := issueRepo.GetByID(f.ctx, f.issues[0])
				require.NoError(t, err)
				issue.Status = "custom-closed"
				issue.StatusID = &f.closed
				require.NoError(t, issueRepo.Update(ctx, issue))
			}
			p, rev := f.planning(t, f.issues[0])
			require.EqualValues(t, 0, *p.RemainingMinutes)
			require.EqualValues(t, 2, rev)
			var actorKind string
			require.NoError(t, f.db.Get(&actorKind, `SELECT actor_kind FROM effort_events WHERE issue_id=$1 AND cause='closure'`, f.issues[0]))
			if kind == "github" {
				require.Equal(t, "system", actorKind)
				var source string
				require.NoError(t, f.db.Get(&source, `SELECT system_source FROM effort_events WHERE issue_id=$1 AND cause='closure'`, f.issues[0]))
				require.Equal(t, "github", source)
			} else {
				require.Equal(t, "user", actorKind)
			}
			issue, err := issueRepo.GetByID(f.ctx, f.issues[0])
			require.NoError(t, err)
			issue.Status = "open"
			issue.StatusID = &f.status
			require.NoError(t, issueRepo.Update(f.ctx, issue))
			p, rev = f.planning(t, issue.ID)
			require.EqualValues(t, 0, *p.RemainingMinutes)
			require.EqualValues(t, 2, rev)
		})
	}
	t.Run("existing planning at revision zero is preserved and closed", func(t *testing.T) {
		f := newEffortFixture(t)
		f.exec(t, `INSERT INTO issue_efforts(workspace_id,issue_id,remaining_minutes) VALUES($1,$2,300)`, f.ws, f.issues[0])
		r := NewIssueRepository(f.db)
		i, err := r.GetByID(f.ctx, f.issues[0])
		require.NoError(t, err)
		i.Status = "custom-closed"
		i.StatusID = &f.closed
		require.NoError(t, r.Update(f.ctx, i))
		p, rev := f.planning(t, i.ID)
		require.EqualValues(t, 0, *p.RemainingMinutes)
		require.EqualValues(t, 1, rev)
	})
	t.Run("closed issue worklogs and estimate cannot restore remaining", func(t *testing.T) {
		f := newEffortFixture(t)
		r := NewIssueRepository(f.db)
		f.remaining(t, f.issues[0], 300)
		i, err := r.GetByID(f.ctx, f.issues[0])
		require.NoError(t, err)
		i.StatusID = &f.closed
		i.Status = "custom-closed"
		require.NoError(t, r.Update(f.ctx, i))
		_, err = f.repo.SetRemaining(f.ctx, f.ws, i.ID, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(2), RemainingMinutes: minutes(60)})
		effortCode(t, err, "VALIDATION_ERROR")
		_, err = f.repo.SetEstimate(f.ctx, f.ws, i.ID, dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(2), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: minutes(480)}})
		require.NoError(t, err)
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, i.ID, f.create(3, f.owner, "2026-01-01", 60))
		require.NoError(t, err)
		p, rev := f.planning(t, i.ID)
		require.EqualValues(t, 0, *p.RemainingMinutes)
		require.EqualValues(t, 4, rev)
	})
	t.Run("disable freezes and reactivation reconciles only closed issues", func(t *testing.T) {
		f := newEffortFixture(t)
		r := NewIssueRepository(f.db)
		f.remaining(t, f.issues[0], 300)
		f.remaining(t, f.issues[1], 240)
		f.settings(t, false, 0)
		issue, err := r.GetByID(f.ctx, f.issues[0])
		require.NoError(t, err)
		issue.Status = "custom-closed"
		issue.StatusID = &f.closed
		require.NoError(t, r.Update(f.ctx, issue))
		p, rev := f.planning(t, issue.ID)
		require.EqualValues(t, 300, *p.RemainingMinutes)
		require.EqualValues(t, 1, rev)
		f.settings(t, true, 1)
		p, rev = f.planning(t, issue.ID)
		require.EqualValues(t, 0, *p.RemainingMinutes)
		require.EqualValues(t, 2, rev)
		p, rev = f.planning(t, f.issues[1])
		require.EqualValues(t, 240, *p.RemainingMinutes)
		require.EqualValues(t, 1, rev)
		var cause string
		require.NoError(t, f.db.Get(&cause, `SELECT cause FROM effort_events WHERE issue_id=$1 AND revision=2`, issue.ID))
		require.Equal(t, "reactivation", cause)
	})
	t.Run("closure history failure rolls back issue and lifecycle", func(t *testing.T) {
		f := newEffortFixture(t)
		f.remaining(t, f.issues[0], 300)
		r := NewIssueRepository(f.db)
		i, err := r.GetByID(f.ctx, f.issues[0])
		require.NoError(t, err)
		// Overflow makes the event/revision write fail inside the closure transaction.
		f.exec(t, `UPDATE issue_efforts SET revision=$1 WHERE issue_id=$2`, domain.EffortMaxSafeInteger, i.ID)
		i.StatusID = &f.closed
		i.Status = "custom-closed"
		require.Error(t, r.Update(f.ctx, i))
		saved, err := r.GetByID(f.ctx, i.ID)
		require.NoError(t, err)
		require.Equal(t, f.status, *saved.StatusID)
		var count int
		require.NoError(t, f.db.Get(&count, `SELECT COUNT(*) FROM issue_lifecycle_events WHERE issue_id=$1 AND event_type='status_changed'`, i.ID))
		require.Equal(t, 0, count)
		p, _ := f.planning(t, i.ID)
		require.EqualValues(t, 300, *p.RemainingMinutes)
	})
	t.Run("reactivation failure rolls back settings and all issues", func(t *testing.T) {
		f := newEffortFixture(t)
		for _, id := range f.issues {
			f.remaining(t, id, 300)
		}
		f.settings(t, false, 0)
		r := NewIssueRepository(f.db)
		slug := "custom-closed"
		_, err := r.BulkUpdate(f.ctx, f.ws, f.issues, &slug, nil, nil, &f.closed, nil, false)
		require.NoError(t, err)
		// Fail the last issue in lock order, proving earlier changes also roll back.
		f.exec(t, `UPDATE issue_efforts SET revision=$1 WHERE issue_id=(SELECT MAX(issue_id::text)::uuid FROM issue_efforts WHERE workspace_id=$2)`, domain.EffortMaxSafeInteger, f.ws)
		enabled := true
		_, err = f.repo.SetSettings(f.ctx, f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(1), Enabled: &enabled, Timezone: "UTC"})
		require.Error(t, err)
		settings, err := f.repo.Settings(f.ctx, f.ws)
		require.NoError(t, err)
		require.False(t, settings.Enabled)
		require.Equal(t, "Europe/Rome", settings.Timezone)
		require.EqualValues(t, 1, settings.Revision)
		for _, id := range f.issues {
			p, _ := f.planning(t, id)
			require.EqualValues(t, 300, *p.RemainingMinutes)
		}
	})
	t.Run("queued closure uses settings committed before lock acquisition", func(t *testing.T) {
		f := newEffortFixture(t)
		f.remaining(t, f.issues[0], 300)
		worker, pid := f.worker(t)
		r := NewIssueRepository(worker.db)
		issue, err := r.GetByID(f.ctx, f.issues[0])
		require.NoError(t, err)
		issue.StatusID = &f.closed
		issue.Status = "custom-closed"
		hold, err := beginEffortTx(f.ctx, f.db, f.ws, true)
		require.NoError(t, err)
		defer hold.Rollback()
		_, err = hold.Exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled) VALUES($1,FALSE)`, f.ws)
		require.NoError(t, err)
		ch := make(chan error, 1)
		go func() { ch <- r.Update(f.ctx, issue) }()
		waitEffortBlocked(t, f.db, pid, txPID(t, hold))
		require.NoError(t, hold.Commit())
		select {
		case err := <-ch:
			require.NoError(t, err)
		case <-time.After(8 * time.Second):
			t.Fatal("closure deadlocked")
		}
		p, _ := f.planning(t, issue.ID)
		require.EqualValues(t, 300, *p.RemainingMinutes)
	})
	t.Run("PAT state changes require both scopes independently of data", func(t *testing.T) {
		f := newEffortFixture(t)
		r := NewIssueRepository(f.db)
		for _, id := range f.issues {
			issue, err := r.GetByID(f.ctx, id)
			require.NoError(t, err)
			issue.StatusID = &f.closed
			issue.Status = "custom-closed"
			for _, scopes := range [][]string{{domain.PermIssueUpdate}, {domain.PermEffortsRead}, {domain.PermEffortsWrite}} {
				require.ErrorIs(t, r.Update(domain.WithTokenScopes(f.ctx, scopes), issue), domain.ErrEffortForbidden)
			}
			require.NoError(t, r.Update(domain.WithTokenScopes(f.ctx, []string{domain.PermEffortsRead, domain.PermEffortsWrite}), issue))
		}
		ctx := domain.WithTokenScopes(f.ctx, []string{domain.PermIssueUpdate})
		issue, err := r.GetByID(f.ctx, f.issues[0])
		require.NoError(t, err)
		issue.Title = "Ordinary title"
		require.NoError(t, r.Update(ctx, issue))
		enabled := false
		_, err = f.repo.SetSettings(domain.WithTokenScopes(f.ctx, []string{domain.PermEffortsRead, domain.PermEffortsWrite}), f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(0), Enabled: &enabled, Timezone: "UTC"})
		require.ErrorIs(t, err, domain.ErrEffortForbidden)
	})
}
