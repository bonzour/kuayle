package repository

import (
	"github.com/google/uuid"
	"testing"

	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestEffortPlanningPostgres(t *testing.T) {
	t.Run("estimate initializes only missing remaining and preserves work", func(t *testing.T) {
		f := newEffortFixture(t)
		id := f.issues[0]
		_, err := f.repo.CreateWorklog(f.ctx, f.ws, id, f.create(0, f.owner, "2026-01-01", 120))
		require.NoError(t, err)
		estimate := func(rev int64, value *int64) {
			t.Helper()
			_, err := f.repo.SetEstimate(f.ctx, f.ws, id, dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(rev), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: value}})
			require.NoError(t, err)
		}
		estimate(1, minutes(480))
		p, rev := f.planning(t, id)
		require.EqualValues(t, 360, *p.RemainingMinutes)
		require.EqualValues(t, 2, rev)
		estimate(2, minutes(60))
		estimate(3, nil)
		p, _ = f.planning(t, id)
		require.Nil(t, p.EstimateMinutes)
		require.EqualValues(t, 360, *p.RemainingMinutes)
		_, err = f.repo.SetRemaining(f.ctx, f.ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(4), RemainingMinutes: minutes(0)})
		require.NoError(t, err)
		estimate(5, minutes(600))
		p, _ = f.planning(t, id)
		require.Zero(t, *p.RemainingMinutes)
		var spent int
		require.NoError(t, f.db.Get(&spent, `SELECT SUM(duration_minutes) FROM effort_worklogs WHERE issue_id=$1`, id))
		require.Equal(t, 120, spent)
		var recorded int
		require.NoError(t, f.db.Get(&recorded, `SELECT COUNT(*) FROM effort_events WHERE issue_id=$1 AND event_type='estimate_set' AND before_snapshot->'planning'->>'estimate_minutes'='60' AND after_snapshot->'planning'->'estimate_minutes'='null'::jsonb`, id))
		require.Equal(t, 1, recorded)
		other := f.issues[1]
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, other, f.create(0, f.owner, "2026-01-02", 120))
		require.NoError(t, err)
		_, err = f.repo.SetEstimate(f.ctx, f.ws, other, dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(1), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: minutes(60)}})
		require.NoError(t, err)
		p, _ = f.planning(t, other)
		require.Zero(t, *p.RemainingMinutes)
	})
	t.Run("closed remaining rejects even zero but replays do not mutate", func(t *testing.T) {
		f := newEffortFixture(t)
		id := f.issues[0]
		req := dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(300)}
		first, err := f.repo.SetRemaining(f.ctx, f.ws, id, req)
		require.NoError(t, err)
		issues := NewIssueRepository(f.db)
		issue, err := issues.GetByID(f.ctx, id)
		require.NoError(t, err)
		issue.StatusID, issue.Status = &f.closed, "custom-closed"
		require.NoError(t, issues.Update(f.ctx, issue))
		for _, value := range []int64{0, 60} {
			_, err = f.repo.SetRemaining(f.ctx, f.ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(2), RemainingMinutes: minutes(value)})
			effortCode(t, err, "VALIDATION_ERROR")
		}
		replay, err := f.repo.SetRemaining(f.ctx, f.ws, id, req)
		require.NoError(t, err)
		require.True(t, replay.Replay)
		require.Equal(t, first.Receipt, replay.Receipt)
		issue.StatusID, issue.Status = &f.status, "open"
		require.NoError(t, issues.Update(f.ctx, issue))
		p, rev := f.planning(t, id)
		require.Zero(t, *p.RemainingMinutes)
		require.EqualValues(t, 2, rev)
		_, err = f.repo.SetRemaining(f.ctx, f.ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(2), RemainingMinutes: minutes(60)})
		require.NoError(t, err)
	})
	for _, table := range []string{"effort_events", "effort_operations"} {
		t.Run("planning rollback on "+table+" failure", func(t *testing.T) {
			f := newEffortFixture(t)
			id := f.issues[0]
			f.exec(t, `CREATE FUNCTION ep02_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$`)
			f.exec(t, `CREATE TRIGGER ep02_fail BEFORE INSERT ON `+table+` FOR EACH ROW EXECUTE FUNCTION ep02_fail()`)
			for _, command := range []func() (*EffortMutationResult, error){
				func() (*EffortMutationResult, error) {
					return f.repo.SetEstimate(f.ctx, f.ws, id, dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(0), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: minutes(480)}})
				},
				func() (*EffortMutationResult, error) {
					return f.repo.SetRemaining(f.ctx, f.ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(60)})
				},
			} {
				_, err := command()
				require.Error(t, err)
				p, rev := f.planning(t, id)
				require.Nil(t, p.EstimateMinutes)
				require.Nil(t, p.RemainingMinutes)
				require.Zero(t, rev)
				w, e, o := f.counts(t)
				require.Equal(t, []int{0, 0, 0}, []int{w, e, o})
			}
		})
	}

	t.Run("queued remaining observes a concurrent closure", func(t *testing.T) {
		f := newEffortFixture(t)
		worker, pid := f.worker(t)
		hold, err := beginEffortTx(f.ctx, f.db, f.ws, false)
		require.NoError(t, err)
		defer hold.Rollback()
		_, err = lockEffortIssues(f.ctx, hold, f.ws, []uuid.UUID{f.issues[0]}, nil, nil)
		require.NoError(t, err)
		_, err = hold.Exec(`UPDATE issues SET status_id=$1 WHERE id=$2`, f.closed, f.issues[0])
		require.NoError(t, err)
		ch := runEffort(func() (*EffortMutationResult, error) {
			return worker.SetRemaining(f.ctx, f.ws, f.issues[0], dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(0)})
		})
		waitEffortBlocked(t, f.db, pid, txPID(t, hold))
		require.NoError(t, hold.Commit())
		effortCode(t, receiveEffort(t, ch).err, "VALIDATION_ERROR")
		_, revision := f.planning(t, f.issues[0])
		require.Zero(t, revision)
	})
}
