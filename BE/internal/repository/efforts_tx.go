package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/lib/pq"
)

var ErrEffortPersistence = errors.New("Efforts persistence failed")

// EffortError uses the T-01 error codes without exposing SQL or hidden budgets.
type EffortError struct {
	Code    string
	Field   string
	Message string
}

func (e *EffortError) Error() string                { return e.Message }
func effortError(code, field, message string) error { return &EffortError{code, field, message} }

// EffortMutationResult is internal: HTTP write routes are intentionally absent.
type EffortMutationResult struct {
	Receipt dto.EffortMutationReceipt
	Replay  bool
}

// Lock order: graph -> settings -> request -> memberships -> statuses/issues ->
// daily budgets -> worklog. Namespaces 470/471/472 are separate from privacy.
func beginEffortTx(ctx context.Context, db *sqlx.DB, ws uuid.UUID, exclusive bool) (*sqlx.Tx, error) {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock_shared(610046,0)`); err == nil {
		lock := "pg_advisory_xact_lock_shared"
		if exclusive {
			lock = "pg_advisory_xact_lock"
		}
		_, err = tx.ExecContext(ctx, `SELECT `+lock+`(hashtextextended($1,470))`, ws.String())
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func effortSettingsTx(ctx context.Context, tx *sqlx.Tx, ws uuid.UUID) (domain.EffortSettings, error) {
	var settings domain.EffortSettings
	err := tx.GetContext(ctx, &settings, `SELECT COALESCE(s.enabled,TRUE) enabled,COALESCE(s.timezone,'Europe/Rome') timezone,
 COALESCE(s.revision,0) revision FROM workspaces w LEFT JOIN workspace_effort_settings s ON s.workspace_id=w.id WHERE w.id=$1`, ws)
	return settings, err
}

func lockEffortMembers(ctx context.Context, tx *sqlx.Tx, ws uuid.UUID, teams, users []uuid.UUID) error {
	// Match RemoveMember: team memberships precede workspace memberships. SHARE,
	// rather than KEY SHARE, also excludes a role UPDATE until this commit.
	var ids []uuid.UUID
	if err := tx.SelectContext(ctx, &ids, `SELECT user_id FROM team_members WHERE team_id=ANY($1) AND user_id=ANY($2)
 ORDER BY team_id,user_id FOR SHARE`, pq.Array(teams), pq.Array(users)); err != nil {
		return err
	}
	return tx.SelectContext(ctx, &ids, `SELECT user_id FROM workspace_members WHERE workspace_id=$1 AND user_id=ANY($2)
 ORDER BY user_id FOR SHARE`, ws, pq.Array(users))
}

func effortRole(ctx context.Context, tx *sqlx.Tx, ws, user uuid.UUID) (string, error) {
	var role string
	err := tx.GetContext(ctx, &role, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, ws, user)
	return role, err
}

func lockEffortIssues(ctx context.Context, tx *sqlx.Tx, ws uuid.UUID, ids []uuid.UUID, extraStatus *uuid.UUID, legacy *string) ([]domain.Issue, error) {
	var statuses []uuid.UUID
	if err := tx.SelectContext(ctx, &statuses, `SELECT id FROM team_statuses WHERE id IN (
 SELECT status_id FROM issues WHERE workspace_id=$1 AND id=ANY($2) UNION SELECT $3::uuid)
 OR (slug=$4 AND team_id IN (SELECT team_id FROM issues WHERE workspace_id=$1 AND id=ANY($2)))
 ORDER BY id FOR SHARE`, ws, pq.Array(ids), extraStatus, legacy); err != nil {
		return nil, err
	}
	issues := []domain.Issue{}
	err := tx.SelectContext(ctx, &issues, `SELECT * FROM issues WHERE workspace_id=$1 AND id=ANY($2) ORDER BY id FOR UPDATE`, ws, pq.Array(ids))
	return issues, err
}

func effortTerminal(ctx context.Context, tx *sqlx.Tx, issue *domain.Issue) (bool, error) {
	var terminal bool
	err := tx.GetContext(ctx, &terminal, `SELECT COALESCE((SELECT category IN ('completed','cancelled') FROM team_statuses WHERE id=$1),
 $2 IN ('done','completed','cancelled'))`, issue.StatusID, string(issue.Status))
	return terminal, err
}

func effortPlanningTx(ctx context.Context, tx *sqlx.Tx, ws, issue uuid.UUID) (domain.EffortPlanningSnapshot, int64, error) {
	var row struct {
		domain.EffortPlanningSnapshot
		Revision int64 `db:"revision"`
	}
	err := tx.GetContext(ctx, &row, `SELECT estimate_minutes,remaining_minutes,revision FROM issue_efforts WHERE workspace_id=$1 AND issue_id=$2`, ws, issue)
	if err == sql.ErrNoRows {
		// sqlx allocates pointer fields before Scan discovers an absent row.
		return domain.EffortPlanningSnapshot{}, 0, nil
	}
	return row.EffortPlanningSnapshot, row.Revision, err
}

func saveEffortPlanning(ctx context.Context, tx *sqlx.Tx, ws, issue uuid.UUID, p domain.EffortPlanningSnapshot, rev int64) error {
	if rev < 1 || rev > domain.EffortMaxSafeInteger {
		return fmt.Errorf("Efforts revision overflow")
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes,revision) VALUES($1,$2,$3,$4,$5)
 ON CONFLICT(issue_id) DO UPDATE SET estimate_minutes=EXCLUDED.estimate_minutes,remaining_minutes=EXCLUDED.remaining_minutes,revision=EXCLUDED.revision`, ws, issue, p.EstimateMinutes, p.RemainingMinutes, rev)
	return err
}

func writeEffortEvent(ctx context.Context, tx *sqlx.Tx, ws, issue uuid.UUID, actor domain.EffortActor, event, cause string, before, after *domain.EffortSnapshot, rev int64) error {
	encode := func(v *domain.EffortSnapshot) (any, error) {
		if v == nil {
			return nil, nil
		}
		b, e := json.Marshal(v)
		return string(b), e
	}
	b, err := encode(before)
	if err != nil {
		return err
	}
	a, err := encode(after)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO effort_events(workspace_id,issue_id,event_type,actor_kind,actor_user_id,system_source,cause,before_snapshot,after_snapshot,revision)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, ws, issue, event, actor.Kind, actor.UserID, actor.SystemSource, cause, b, a, rev)
	return err
}

func planningSnapshot(p domain.EffortPlanningSnapshot) *domain.EffortSnapshot {
	return &domain.EffortSnapshot{Planning: &p}
}
func sameMinutes(a, b *int64) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// Called inside the issue mutation transaction, after the settings gate and issue
// locks. No separate history call or broadcast may make closure partially commit.
func closeEffortTx(ctx context.Context, tx *sqlx.Tx, issue *domain.Issue, cause string) error {
	settings, err := effortSettingsTx(ctx, tx, issue.WorkspaceID)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return nil
	}
	terminal, err := effortTerminal(ctx, tx, issue)
	if err != nil {
		return err
	}
	if !terminal {
		return nil
	}
	before, rev, err := effortPlanningTx(ctx, tx, issue.WorkspaceID, issue.ID)
	if err != nil {
		return err
	}
	// Avoid creating effort rows for issues that have never contained efforts.
	if rev == 0 && before.EstimateMinutes == nil && before.RemainingMinutes == nil || before.RemainingMinutes != nil && *before.RemainingMinutes == 0 {
		return nil
	}
	actor, err := domain.EffortActorFromContext(ctx)
	if err != nil {
		return err
	}
	after := before
	zero := int64(0)
	after.RemainingMinutes = &zero
	if err = saveEffortPlanning(ctx, tx, issue.WorkspaceID, issue.ID, after, rev+1); err != nil {
		return err
	}
	return writeEffortEvent(ctx, tx, issue.WorkspaceID, issue.ID, actor, "remaining_set", cause, planningSnapshot(before), planningSnapshot(after), rev+1)
}

func lockEffortBudgets(ctx context.Context, tx *sqlx.Tx, ws, owner uuid.UUID, dates ...domain.EffortDate) error {
	keys := make([]string, 0, len(dates))
	for _, date := range dates {
		keys = append(keys, ws.String()+"/"+owner.String()+"/"+string(date))
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i > 0 && keys[i-1] == key {
			continue
		}
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,472))`, key); err != nil {
			return err
		}
	}
	return nil
}
