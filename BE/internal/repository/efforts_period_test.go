package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestEffortPeriodDates(t *testing.T) {
	for _, tc := range []struct {
		start, end string
		skip       bool
		want       []domain.EffortDate
	}{
		{"2026-01-02", "2026-01-05", true, []domain.EffortDate{"2026-01-02", "2026-01-05"}},
		{"2026-01-02", "2026-01-05", false, []domain.EffortDate{"2026-01-02", "2026-01-03", "2026-01-04", "2026-01-05"}},
		{"2024-02-28", "2024-03-01", true, []domain.EffortDate{"2024-02-28", "2024-02-29", "2024-03-01"}},
		{"2026-03-27", "2026-03-30", true, []domain.EffortDate{"2026-03-27", "2026-03-30"}},
		{"0001-01-01", "0001-01-01", false, []domain.EffortDate{"0001-01-01"}},
		{"9999-12-31", "9999-12-31", false, []domain.EffortDate{"9999-12-31"}},
	} {
		got, err := effortPeriodDates(context.Background(), domain.EffortDate(tc.start), tc.end, tc.skip)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	for _, end := range []string{"2026-02-30", "2026-01-01", ""} {
		_, err := effortPeriodDates(context.Background(), "2026-01-02", end, true)
		effortCode(t, err, "VALIDATION_ERROR")
	}
	_, err := effortPeriodDates(context.Background(), "2026-01-03", "2026-01-04", true)
	effortCode(t, err, "VALIDATION_ERROR")
}

func TestEffortPeriodPostgres(t *testing.T) {
	t.Run("one revision and persistent retry cover the entire inclusive period", func(t *testing.T) {
		f := newEffortFixture(t)
		_, err := f.repo.SetRemaining(f.ctx, f.ws, f.issues[0], dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(1440)})
		require.NoError(t, err)
		req := f.create(1, f.member, "2026-01-02", 480)
		end := "2026-01-05"
		req.EndDate, req.SkipWeekends = &end, true
		result, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], req)
		require.NoError(t, err)
		require.EqualValues(t, 2, result.Receipt.Revision)
		require.Equal(t, f.issues[0].String(), result.Receipt.ResourceID)
		var dates []domain.EffortDate
		require.NoError(t, f.db.Select(&dates, `SELECT work_date FROM effort_worklogs ORDER BY work_date`))
		require.Equal(t, []domain.EffortDate{"2026-01-02", "2026-01-05"}, dates)
		p, revision := f.planning(t, f.issues[0])
		require.EqualValues(t, 480, *p.RemainingMinutes)
		require.EqualValues(t, 2, revision)
		counts := []int{}
		w, e, o := f.counts(t)
		counts = append(counts, w, e, o)
		replay, err := NewEffortRepository(f.db).CreateWorklog(f.ctx, f.ws, f.issues[0], req)
		require.NoError(t, err)
		require.True(t, replay.Replay)
		require.Equal(t, result.Receipt, replay.Receipt)
		w, e, o = f.counts(t)
		require.Equal(t, counts, []int{w, e, o})
		req.SkipWeekends = false
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], req)
		effortCode(t, err, dto.EffortRequestConflict)
	})
	t.Run("daily limit on a later date rolls back earlier rows history remaining and receipt", func(t *testing.T) {
		f := newEffortFixture(t)
		_, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[1], f.create(0, f.owner, "2026-01-05", 1380))
		require.NoError(t, err)
		_, err = f.repo.SetRemaining(f.ctx, f.ws, f.issues[0], dto.SetEffortRemainingRequest{EffortMutationRequest: mutation(0), RemainingMinutes: minutes(480)})
		require.NoError(t, err)
		w, e, o := f.counts(t)
		req := f.create(1, f.owner, "2026-01-02", 120)
		end := "2026-01-05"
		req.EndDate, req.SkipWeekends = &end, true
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], req)
		effortCode(t, err, dto.EffortDailyLimit)
		aw, ae, ao := f.counts(t)
		require.Equal(t, []int{w, e, o}, []int{aw, ae, ao})
		p, revision := f.planning(t, f.issues[0])
		require.EqualValues(t, 480, *p.RemainingMinutes)
		require.EqualValues(t, 1, revision)
	})
	t.Run("weekends may be included and invalid future or empty periods never write", func(t *testing.T) {
		f := newEffortFixture(t)
		req := f.create(0, f.owner, "2026-01-03", 240)
		end := "2026-01-04"
		req.EndDate = &end
		req.SkipWeekends = true
		_, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], req)
		effortCode(t, err, "VALIDATION_ERROR")
		req.SkipWeekends = false
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], req)
		require.NoError(t, err)
		req = f.create(1, f.owner, "2026-01-03", 240)
		end = "9999-12-31"
		req.EndDate = &end
		// A future bound is rejected even if some days would be skipped.
		// Keep the requested range short by moving its start into the future too.
		req.WorkDate = "9999-12-30"
		_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], req)
		effortCode(t, err, "VALIDATION_ERROR")
		w, _, _ := f.counts(t)
		require.Equal(t, 2, w)
	})
	t.Run("overlapping periods on separate issues cannot exceed a daily budget", func(t *testing.T) {
		f := newEffortFixture(t)
		req := f.create(0, f.owner, "2026-01-02", 960)
		end := "2026-01-05"
		req.EndDate, req.SkipWeekends = &end, true
		out := make(chan error, 2)
		for _, issue := range f.issues[:2] {
			go func(id uuid.UUID, request dto.CreateEffortWorklogRequest) {
				_, err := f.repo.CreateWorklog(f.ctx, f.ws, id, request)
				out <- err
			}(issue, req)
			req.EffortMutationRequest = mutation(0)
		}
		first, second := <-out, <-out
		if first == nil {
			effortCode(t, second, dto.EffortDailyLimit)
		} else {
			effortCode(t, first, dto.EffortDailyLimit)
			require.NoError(t, second)
		}
		w, _, o := f.counts(t)
		require.Equal(t, 2, w)
		require.Equal(t, 1, o)
	})
}
