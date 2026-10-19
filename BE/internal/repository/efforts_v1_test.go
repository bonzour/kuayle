package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestEffortHistoryScopesAndHistoricalAttribution(t *testing.T) {
	f := newEffortFixture(t)
	description := "Lavoro storico\n漢字 👩🏽‍💻"
	created, err := f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], dto.CreateEffortWorklogRequest{EffortMutationRequest: mutation(0), OwnerUserID: f.member.String(), WorkDate: "2020-01-01", DurationMinutes: 120, Description: &description})
	require.NoError(t, err)
	_, err = f.repo.DeleteWorklog(f.ctx, f.ws, f.issues[0], uuid.MustParse(created.Receipt.ResourceID), dto.DeleteEffortWorklogRequest{EffortMutationRequest: mutation(1)})
	require.NoError(t, err)
	f.exec(t, `UPDATE issues SET assignee_id=NULL WHERE workspace_id=$1`, f.ws)
	f.exec(t, `DELETE FROM users WHERE id=$1`, f.member)
	history := NewIssueHistoryRepository(f.db)
	entries, err := history.ListByIssue(f.ctx, f.issues[0])
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "worklog_deleted", entries[0].Effort.EventType)
	require.Equal(t, description, *entries[0].Effort.Before.Worklog.Description)
	require.Equal(t, f.owner, *entries[0].Effort.Actor.UserID)
	for _, u := range entries[0].Effort.Users {
		if u.ID == f.member {
			require.False(t, u.Available)
		}
	}
	legacy := domain.WithTokenScopes(f.ctx, []string{domain.PermIssuesRead})
	entries, err = history.ListByIssue(legacy, f.issues[0])
	require.NoError(t, err)
	require.Empty(t, entries)
	entries, err = history.ListByIssue(domain.WithTokenScopes(f.ctx, []string{domain.PermIssuesRead, domain.PermEffortsRead}), f.issues[0])
	require.NoError(t, err)
	require.Len(t, entries, 2)
	public, err := history.ListByIssue(context.Background(), f.issues[0])
	require.NoError(t, err)
	encoded, _ := json.Marshal(public)
	require.NotContains(t, string(encoded), "worklog_deleted")
}

func TestEffortDuplicateAndPreservedChildren(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewIssueRepository(f.db)
	estimate := int64(480)
	_, err := f.repo.SetEstimate(f.ctx, f.ws, f.issues[0], dto.SetEffortEstimateRequest{EffortMutationRequest: mutation(0), EstimateMinutes: dto.OptionalEffortMinutes{Set: true, Value: &estimate}})
	require.NoError(t, err)
	_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], dto.CreateEffortWorklogRequest{EffortMutationRequest: mutation(1), OwnerUserID: f.owner.String(), WorkDate: "2020-01-01", DurationMinutes: 120})
	require.NoError(t, err)
	for _, terminal := range []bool{false, true} {
		tx, err := repo.BeginDuplicateTx(f.ctx, f.ws, f.issues[0])
		require.NoError(t, err)
		status := f.status
		if terminal {
			status = f.closed
		}
		n, err := repo.NextNumber(f.ctx, tx, f.team)
		require.NoError(t, err)
		copy := &domain.Issue{ID: uuid.New(), WorkspaceID: f.ws, TeamID: f.team, CreatorID: f.owner, Number: n, Identifier: "COPY-" + uuid.NewString(), Title: "Copy", Status: "open", StatusID: &status}
		require.NoError(t, repo.Create(f.ctx, tx, copy))
		require.NoError(t, repo.CopyEffortPlanning(f.ctx, tx, f.issues[0], copy))
		require.NoError(t, tx.Commit())
		summary, err := f.repo.Summary(f.ctx, f.ws, copy.Identifier)
		require.NoError(t, err)
		require.Equal(t, estimate, *summary.Own.EstimateMinutes)
		require.Zero(t, summary.Own.SpentMinutes)
		if terminal {
			require.Zero(t, *summary.Own.RemainingMinutes)
		} else {
			require.Equal(t, estimate, *summary.Own.RemainingMinutes)
		}
		var count int
		require.NoError(t, f.db.Get(&count, `SELECT count(*) FROM effort_events WHERE issue_id=$1`, copy.ID))
		require.Zero(t, count)
	}
	f.exec(t, `UPDATE issues SET parent_id=$1 WHERE id=$2`, f.issues[0], f.issues[1])
	f.exec(t, `INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes) VALUES($1,$2,90,0)`, f.ws, f.issues[1])
	childBefore, err := repo.GetByID(f.ctx, f.issues[1])
	require.NoError(t, err)
	childBefore.ParentID = &f.issues[2]
	require.NoError(t, repo.Update(f.ctx, childBefore))
	oldRoot, err := f.repo.Summary(f.ctx, f.ws, "T02-"+f.issues[0].String()[:8])
	require.NoError(t, err)
	require.EqualValues(t, 480, *oldRoot.Total.EstimateMinutes)
	newRoot, err := f.repo.Summary(f.ctx, f.ws, "T02-"+f.issues[2].String()[:8])
	require.NoError(t, err)
	require.EqualValues(t, 90, *newRoot.Total.EstimateMinutes)
	require.True(t, newRoot.Total.EstimatePartial)
	require.True(t, newRoot.HasVisibleDescendants)
	childBefore.ParentID = &f.issues[0]
	require.NoError(t, repo.Update(f.ctx, childBefore))
	require.ErrorIs(t, repo.Delete(domain.WithTokenScopes(f.ctx, []string{domain.PermIssueDeleteOwn}), f.issues[0]), domain.ErrEffortForbidden)
	require.NoError(t, repo.Delete(f.ctx, f.issues[0]))
	child, err := repo.GetByID(f.ctx, f.issues[1])
	require.NoError(t, err)
	require.NotNil(t, child)
	require.Nil(t, child.ParentID)
	summary, err := f.repo.Summary(f.ctx, f.ws, child.Identifier)
	require.NoError(t, err)
	require.EqualValues(t, 90, *summary.Own.EstimateMinutes)
	var count int
	require.NoError(t, f.db.Get(&count, `SELECT count(*) FROM effort_worklogs WHERE issue_id=$1`, f.issues[0]))
	require.Zero(t, count)
	require.NoError(t, f.db.Get(&count, `SELECT count(*) FROM effort_events WHERE issue_id=$1`, f.issues[0]))
	require.Zero(t, count)
}

func TestEffortExportSnapshot(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewWorkspaceTransferRepository(f.db)
	guard, err := repo.BeginPublicExport(f.ctx, f.ws)
	require.NoError(t, err)
	defer guard.Rollback()
	reader := repo.ExportReader(guard)
	specs := WorkspaceTransferTableSpecs()
	var planning, worklogs WorkspaceTransferTableSpec
	for _, s := range specs {
		switch s.Name {
		case "issue_efforts":
			planning = s
		case "effort_worklogs":
			worklogs = s
		}
	}
	before, err := reader.TableRows(f.ctx, planning, f.ws)
	require.NoError(t, err)
	require.Empty(t, before)
	// A distinct connection commits between reads; the archive keeps one snapshot.
	_, err = f.repo.CreateWorklog(f.ctx, f.ws, f.issues[0], dto.CreateEffortWorklogRequest{EffortMutationRequest: mutation(0), OwnerUserID: f.owner.String(), WorkDate: "2020-01-01", DurationMinutes: 60})
	require.NoError(t, err)
	rows, err := reader.TableRows(f.ctx, worklogs, f.ws)
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NoError(t, guard.Rollback())
	guard, err = repo.BeginPublicExport(f.ctx, f.ws)
	require.NoError(t, err)
	defer guard.Rollback()
	rows, err = repo.ExportReader(guard).TableRows(f.ctx, worklogs, f.ws)
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

// Record measured volumes and times, without imposing an arbitrary latency SLA.
func TestEffortRepresentativeQueries(t *testing.T) {
	f := newEffortFixture(t)
	f.exec(t, `CREATE TEMP TABLE effort_volume AS SELECT n,gen_random_uuid() id FROM generate_series(1,3000) n`)
	f.exec(t, `INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id,parent_id)
 SELECT v.id,$1,$2,v.n+3,'VOLUME-'||v.n,'Volume',$3,CASE WHEN v.n%2=0 THEN $4::uuid ELSE $5::uuid END,COALESCE(p.id,$6::uuid)
 FROM effort_volume v LEFT JOIN effort_volume p ON p.n=v.n/10`, f.ws, f.team, f.owner, f.closed, f.status, f.issues[0])
	f.exec(t, `INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes) SELECT $1,id,60,30 FROM effort_volume`, f.ws)
	f.exec(t, `INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes)
 SELECT $1,id,$2,$2,'2020-01-01'::date+(n/1000),1 FROM effort_volume`, f.ws, f.owner)
	identifier := "T02-" + f.issues[0].String()[:8]
	start := time.Now()
	summary, err := f.repo.Summary(f.ctx, f.ws, identifier)
	elapsed := time.Since(start)
	require.NoError(t, err)
	require.True(t, summary.HasVisibleDescendants)
	require.EqualValues(t, 180000, *summary.Total.EstimateMinutes)
	require.EqualValues(t, 3000, summary.Total.SpentMinutes)
	t.Logf("3001 visible issues, 3000 worklogs: summary %s", elapsed)
	var plan []string
	require.NoError(t, f.db.Select(&plan, `EXPLAIN (ANALYZE,BUFFERS) WITH RECURSIVE tree(id) AS (
      SELECT $1::uuid UNION SELECT i.id FROM issues i JOIN tree p ON i.parent_id=p.id WHERE i.workspace_id=$2)
      SELECT sum(e.estimate_minutes),sum(e.remaining_minutes) FROM tree JOIN issues i ON i.id=tree.id
      LEFT JOIN issue_efforts e ON e.issue_id=i.id WHERE `+teamVisible(f.ctx, "i.team_id"), f.issues[0], f.ws))
	t.Logf("recursive planning query plan:\n%s", strings.Join(plan, "\n"))
	f.settings(t, false, 0)
	enabled := true
	start = time.Now()
	_, err = f.repo.SetSettings(f.ctx, f.ws, dto.SetEffortSettingsRequest{EffortMutationRequest: mutation(1), Enabled: &enabled, Timezone: "Europe/Rome"})
	require.NoError(t, err)
	t.Logf("reactivation: 1500 terminal issues and events, %s", time.Since(start))
	var count int
	require.NoError(t, f.db.Get(&count, `SELECT count(*) FROM effort_events WHERE workspace_id=$1 AND cause='reactivation'`, f.ws))
	require.Equal(t, 1500, count)
}
