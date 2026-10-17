package repository

import (
	"context"

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
