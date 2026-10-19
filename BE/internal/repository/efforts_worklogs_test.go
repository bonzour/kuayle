package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestEffortWorklogCorrectionsPostgres(t *testing.T) {
	t.Run("creation preserves absent remaining and records duration beyond the prediction", func(t *testing.T) {
		f := newEffortFixture(t)
		_, err := f.repo.SetRemaining(f.ctx, f.ws, f.issues[0], dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(30)})
		require.NoError(t, err)
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(1, f.member, "2026-01-01", 60))
		require.NoError(t, err)
		planning, _ := f.planning(t, f.issues[0])
		require.Nil(t, planning.EstimateMinutes)
		require.Zero(t, *planning.RemainingMinutes)
		logs, err := f.repo.Worklogs(f.ctx, f.ws, "T02-"+f.issues[0].String()[:8], 1)
		require.NoError(t, err)
		require.EqualValues(t, 60, logs.Data[0].DurationMinutes)
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[1], f.create(0, f.member, "2026-01-02", 1440))
		require.NoError(t, err)
		planning, _ = f.planning(t, f.issues[1])
		require.Nil(t, planning.EstimateMinutes)
		require.Nil(t, planning.RemainingMinutes)
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[2], f.create(0, f.member, "2026-01-02", 1))
		effortCode(t, err, dto.EffortDailyLimit)
	})
	t.Run("daily corrections exclude the previous duration and rejected moves preserve both dates", func(t *testing.T) {
		f := newEffortFixture(t)
		first, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(0, f.member, "2026-01-01", 1320))
		require.NoError(t, err)
		second, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[1], f.create(0, f.member, "2026-01-01", 120))
		require.NoError(t, err)
		update := dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: "2026-01-01", DurationMinutes: 180}
		_, err = f.repo.UpdateWorklog(f.ctx, f.ws, f.issues[1], uuid.MustParse(second.Receipt.ResourceID), update)
		effortCode(t, err, dto.EffortDailyLimit)
		update.DurationMinutes = 60
		_, err = f.repo.UpdateWorklog(f.ctx, f.ws, f.issues[1], uuid.MustParse(second.Receipt.ResourceID), update)
		require.NoError(t, err)
		// Distinct intentional attempts with the same content remain possible.
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[1], f.create(2, f.member, "2026-01-01", 60))
		require.NoError(t, err)
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[1], f.create(3, f.member, "2026-01-01", 1))
		effortCode(t, err, dto.EffortDailyLimit)
		moving, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[2], f.create(0, f.member, "2026-01-02", 120))
		require.NoError(t, err)
		update = dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: "2026-01-01", DurationMinutes: 120}
		_, err = f.repo.UpdateWorklog(f.ctx, f.ws, f.issues[2], uuid.MustParse(moving.Receipt.ResourceID), update)
		effortCode(t, err, dto.EffortDailyLimit)
		var date string
		require.NoError(t, f.db.Get(&date, `SELECT work_date::text FROM effort_worklogs WHERE id=$1`, moving.Receipt.ResourceID))
		require.Equal(t, "2026-01-02", date)
		_, err = f.repo.DeleteWorklog(f.ctx, f.ws, f.issues[0], uuid.MustParse(first.Receipt.ResourceID), dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(1)})
		require.NoError(t, err)
		_, err = f.repo.UpdateWorklog(f.ctx, f.ws, f.issues[2], uuid.MustParse(moving.Receipt.ResourceID), update)
		require.NoError(t, err, "a successful deletion frees the budget")
	})
	for _, table := range []string{"effort_events", "effort_operations"} {
		t.Run("update and deletion roll back on "+table+" failure", func(t *testing.T) {
			f := newEffortFixture(t)
			issue := f.issues[0]
			created, err := f.repo.CreateWorklog(f.ctx, f.ws, issue, f.create(0, f.member, "2026-01-01", 120))
			require.NoError(t, err)
			id := uuid.MustParse(created.Receipt.ResourceID)
			before, err := f.repo.Worklogs(f.ctx, f.ws, "T02-"+issue.String()[:8], 1)
			require.NoError(t, err)
			w, e, o := f.counts(t)
			f.exec(t, `CREATE FUNCTION ep03_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$`)
			f.exec(t, `CREATE TRIGGER ep03_fail BEFORE INSERT ON `+table+` FOR EACH ROW EXECUTE FUNCTION ep03_fail()`)
			for _, command := range []func() (*EffortMutationResult, error){
				func() (*EffortMutationResult, error) {
					return f.repo.UpdateWorklog(f.ctx, f.ws, issue, id, dto.UpdateEffortWorklogRequest{EffortMutationRequest: mutation(1), WorkDate: "2026-01-02", DurationMinutes: 180})
				},
				func() (*EffortMutationResult, error) {
					return f.repo.DeleteWorklog(f.ctx, f.ws, issue, id, dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(1)})
				},
			} {
				_, err = command()
				require.Error(t, err)
				after, err := f.repo.Worklogs(f.ctx, f.ws, "T02-"+issue.String()[:8], 1)
				require.NoError(t, err)
				require.Equal(t, before, after)
				aw, ae, ao := f.counts(t)
				require.Equal(t, []int{w, e, o}, []int{aw, ae, ao})
			}
		})
	}
}

func TestEffortDelegationRevocationPostgres(t *testing.T) {
	for _, kind := range []string{"owner_role", "owner_membership", "owner_team", "operator_role", "disabled"} {
		t.Run(kind, func(t *testing.T) {
			f := newEffortFixture(t)
			if kind == "owner_team" {
				f.exec(t, `UPDATE teams SET is_private=TRUE WHERE id=$1`, f.team)
				f.exec(t, `INSERT INTO team_members(team_id,user_id) VALUES($1,$2)`, f.team, f.member)
			}
			worker, pid := f.worker(t)
			hold, err := f.db.BeginTxx(f.ctx, nil)
			require.NoError(t, err)
			defer hold.Rollback()
			switch kind {
			case "owner_role":
				_, err = hold.Exec(`UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
			case "owner_membership":
				_, err = hold.Exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
			case "owner_team":
				_, err = hold.Exec(`DELETE FROM team_members WHERE team_id=$1 AND user_id=$2`, f.team, f.member)
			case "operator_role":
				_, err = hold.Exec(`UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.owner)
			case "disabled":
				_, err = hold.Exec(`SELECT pg_advisory_xact_lock(hashtextextended($1,470))`, f.ws.String())
				require.NoError(t, err)
				_, err = hold.Exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled) VALUES($1,FALSE)`, f.ws)
			}
			require.NoError(t, err)
			command := runEffort(func() (*EffortMutationResult, error) {
				return worker.CreateWorklog(f.ctx, f.ws, f.issues[0], f.create(0, f.member, "2026-01-01", 60))
			})
			waitEffortBlocked(t, f.db, pid, txPID(t, hold))
			require.NoError(t, hold.Commit())
			result := receiveEffort(t, command)
			if kind == "disabled" {
				effortCode(t, result.err, dto.EffortDisabled)
			} else {
				require.ErrorIs(t, result.err, domain.ErrEffortForbidden)
			}
			w, e, o := f.counts(t)
			require.Equal(t, []int{0, 0, 0}, []int{w, e, o})
		})
	}
}
