package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/lib/pq"
)

// Prepare existing issue writes with the same settings/membership/issue order as
// Efforts commands. This is also used by bulk and GitHub; no process mutexes.
func beginEffortIssueWrite(ctx context.Context, db *sqlx.DB, ws uuid.UUID, ids []uuid.UUID, status *uuid.UUID, legacy *string) (*sqlx.Tx, []domain.Issue, error) {
	tx, err := beginEffortTx(ctx, db, ws, false)
	if err != nil {
		return nil, nil, err
	}
	fail := func(err error) (*sqlx.Tx, []domain.Issue, error) { _ = tx.Rollback(); return nil, nil, err }
	var teams []uuid.UUID
	if err = tx.SelectContext(ctx, &teams, `SELECT DISTINCT team_id FROM issues WHERE workspace_id=$1 AND id=ANY($2) ORDER BY team_id`, ws, pq.Array(ids)); err != nil {
		return fail(err)
	}
	if access, ok := domain.AccessFromContext(ctx); ok {
		if err = lockEffortMembers(ctx, tx, ws, teams, []uuid.UUID{access.UserID}); err != nil {
			return fail(err)
		}
		role, e := effortRole(ctx, tx, ws, access.UserID)
		if e != nil {
			return fail(e)
		}
		if role != "owner" && role != "admin" && role != "member" {
			return fail(domain.ErrEffortForbidden)
		}
	}
	issues, err := lockEffortIssues(ctx, tx, ws, ids, status, legacy)
	if err != nil {
		return fail(err)
	}
	return tx, issues, nil
}

func lockEffortAssignmentIssue(ctx context.Context, tx *sqlx.Tx, issueID uuid.UUID) error {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock_shared(610046,0)`); err != nil {
		return err
	}
	var id uuid.UUID
	return tx.GetContext(ctx, &id, `SELECT id FROM issues WHERE id=$1 AND `+issueVisible(ctx, "issues.id")+` FOR UPDATE`, issueID)
}

func sameStatus(a, b *domain.Issue) bool {
	return a.Status == b.Status && (a.StatusID == nil && b.StatusID == nil || a.StatusID != nil && b.StatusID != nil && *a.StatusID == *b.StatusID)
}

// Hold settings, current membership and source planning until the copy commits.
func (r *IssueRepository) BeginDuplicateTx(ctx context.Context, ws, source uuid.UUID) (*sqlx.Tx, error) {
	if err := domain.CheckEffortToken(ctx); err != nil {
		return nil, err
	}
	tx, issues, err := beginEffortIssueWrite(ctx, r.db, ws, []uuid.UUID{source}, nil, nil)
	if err != nil {
		return nil, err
	}
	if len(issues) != 1 {
		_ = tx.Rollback()
		return nil, sql.ErrNoRows
	}
	if err = requireVisible(ctx, tx, effortWorkspaceVisible(ctx, uuidSQL(ws))+" AND "+issueVisible(ctx, uuidSQL(source))); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r *IssueRepository) CopyEffortPlanning(ctx context.Context, tx *sqlx.Tx, source uuid.UUID, issue *domain.Issue) error {
	settings, err := effortSettingsTx(ctx, tx, issue.WorkspaceID)
	if err != nil || !settings.Enabled {
		return err
	}
	planning, _, err := effortPlanningTx(ctx, tx, issue.WorkspaceID, source)
	if err != nil {
		return err
	}
	planning.RemainingMinutes = planning.EstimateMinutes
	terminal, err := effortTerminal(ctx, tx, issue)
	if err != nil {
		return err
	}
	if terminal {
		zero := int64(0)
		planning.RemainingMinutes = &zero
	}
	if planning.EstimateMinutes == nil && planning.RemainingMinutes == nil {
		return nil
	}
	return saveEffortPlanning(ctx, tx, issue.WorkspaceID, issue.ID, planning, 1)
}
