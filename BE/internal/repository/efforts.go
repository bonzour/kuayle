package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
)

type EffortRepository struct{ db *sqlx.DB }

func NewEffortRepository(db *sqlx.DB) *EffortRepository { return &EffortRepository{db: db} }

// Existing public-context predicates permit public teams. Efforts requires an
// authenticated workspace member even when called outside the HTTP middleware.
func effortWorkspaceVisible(ctx context.Context, column string) string {
	access, ok := domain.AccessFromContext(ctx)
	if !ok || access.UserID == uuid.Nil {
		return "FALSE"
	}
	return workspaceVisible(ctx, column)
}

func (r *EffortRepository) Settings(ctx context.Context, workspaceID uuid.UUID) (*domain.EffortSettings, error) {
	var settings domain.EffortSettings
	err := r.db.GetContext(ctx, &settings, `SELECT COALESCE(s.enabled,TRUE) AS enabled,
 COALESCE(s.timezone,'Europe/Rome') AS timezone, COALESCE(s.revision,0) AS revision
 FROM workspaces w LEFT JOIN workspace_effort_settings s ON s.workspace_id=w.id
 WHERE w.id=$1 AND `+effortWorkspaceVisible(ctx, "w.id"), workspaceID)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// Traversal is workspace-scoped; visibility filters contributions rather than
// pruning traversal, so a visible descendant is not lost behind a hidden node.
// UNION deduplicates IDs and also terminates on a corrupt legacy cycle.
func (r *EffortRepository) Summary(ctx context.Context, workspaceID uuid.UUID, identifier string) (*domain.EffortSummary, error) {
	var summary domain.EffortSummary
	err := r.db.GetContext(ctx, &summary, `WITH RECURSIVE root AS (
 SELECT i.id FROM issues i WHERE i.workspace_id=$1 AND i.identifier_text=$2
 AND `+effortWorkspaceVisible(ctx, "i.workspace_id")+` AND `+teamVisible(ctx, "i.team_id")+`
 ), tree(id) AS (
 SELECT id FROM root UNION SELECT i.id FROM issues i JOIN tree p ON i.parent_id=p.id WHERE i.workspace_id=$1
 ), visible AS (
 SELECT i.id FROM tree t JOIN issues i ON i.id=t.id
 WHERE `+effortWorkspaceVisible(ctx, "i.workspace_id")+` AND `+teamVisible(ctx, "i.team_id")+`
 ), spent AS (
 SELECT w.issue_id,SUM(w.duration_minutes)::bigint AS minutes FROM effort_worklogs w
 JOIN visible v ON v.id=w.issue_id WHERE w.workspace_id=$1 GROUP BY w.issue_id
 ), contributions AS (
 SELECT v.id,e.estimate_minutes,e.remaining_minutes,COALESCE(e.revision,0) AS revision,
 COALESCE(s.minutes,0) AS spent_minutes FROM visible v
 LEFT JOIN issue_efforts e ON e.issue_id=v.id AND e.workspace_id=$1
 LEFT JOIN spent s ON s.issue_id=v.id
 ) SELECT own.revision,
 own.estimate_minutes AS "own.estimate_minutes",own.spent_minutes AS "own.spent_minutes",
 own.remaining_minutes AS "own.remaining_minutes",
 totals.estimate_minutes AS "total.estimate_minutes",totals.spent_minutes AS "total.spent_minutes",
 totals.remaining_minutes AS "total.remaining_minutes",totals.estimate_partial AS "total.estimate_partial",
 totals.remaining_partial AS "total.remaining_partial"
 FROM contributions own JOIN root ON root.id=own.id CROSS JOIN (
 SELECT SUM(estimate_minutes)::bigint AS estimate_minutes,SUM(spent_minutes)::bigint AS spent_minutes,
 COALESCE(SUM(remaining_minutes),0)::bigint AS remaining_minutes,
 BOOL_OR(estimate_minutes IS NULL) AS estimate_partial,BOOL_OR(remaining_minutes IS NULL) AS remaining_partial
 FROM contributions) totals`, workspaceID, identifier)
	if err != nil {
		return nil, err
	}
	numbers := []int64{summary.Total.SpentMinutes, summary.Total.RemainingMinutes, summary.Own.SpentMinutes}
	if summary.Total.EstimateMinutes != nil {
		numbers = append(numbers, *summary.Total.EstimateMinutes)
	}
	for _, number := range numbers {
		if number > domain.EffortMaxSafeInteger {
			return nil, fmt.Errorf("Efforts total exceeds the JSON safe integer range")
		}
	}
	return &summary, nil
}

func (r *EffortRepository) Worklogs(ctx context.Context, workspaceID uuid.UUID, identifier string, page int) (*dto.ListResponse[domain.EffortWorklog], error) {
	// The root, count and page share one statement snapshot. The lateral LEFT
	// JOIN retains an authorized empty issue/page, distinct from a missing issue.
	type row struct {
		TotalCount          int           `db:"total_count"`
		ID                  *uuid.UUID    `db:"id"`
		IssueID             uuid.UUID     `db:"issue_id"`
		OwnerID             *uuid.UUID    `db:"owner_id"`
		OwnerName           *string       `db:"owner_name"`
		OwnerDisplayName    *string       `db:"owner_display_name"`
		OwnerAvailable      bool          `db:"owner_available"`
		OperatorID          *uuid.UUID    `db:"operator_id"`
		OperatorName        *string       `db:"operator_name"`
		OperatorDisplayName *string       `db:"operator_display_name"`
		OperatorAvailable   bool          `db:"operator_available"`
		WorkDate            sql.NullTime  `db:"work_date"`
		DurationMinutes     sql.NullInt64 `db:"duration_minutes"`
		Description         *string       `db:"description"`
		CreatedAt           sql.NullTime  `db:"created_at"`
		UpdatedAt           sql.NullTime  `db:"updated_at"`
	}
	if page < 1 || int64(page) > domain.EffortMaxSafeInteger/domain.EffortPageSize {
		return nil, fmt.Errorf("invalid Efforts page")
	}
	rows := []row{}
	err := r.db.SelectContext(ctx, &rows, `WITH root AS MATERIALIZED (
 SELECT i.id,(SELECT COUNT(*) FROM effort_worklogs w WHERE w.workspace_id=$1 AND w.issue_id=i.id) AS total_count
 FROM issues i WHERE i.workspace_id=$1 AND i.identifier_text=$2
 AND `+effortWorkspaceVisible(ctx, "i.workspace_id")+` AND `+teamVisible(ctx, "i.team_id")+`
 ) SELECT r.id AS issue_id,r.total_count,
 w.id,w.owner_user_id AS owner_id,owner.name AS owner_name,owner.display_name AS owner_display_name,
 (owner.id IS NOT NULL) AS owner_available,
 w.created_by_user_id AS operator_id,operator.name AS operator_name,operator.display_name AS operator_display_name,
 (operator.id IS NOT NULL) AS operator_available,w.work_date,w.duration_minutes,w.description,w.created_at,w.updated_at
 FROM root r LEFT JOIN LATERAL (
 SELECT * FROM effort_worklogs w WHERE w.workspace_id=$1 AND w.issue_id=r.id
 ORDER BY w.work_date DESC,w.created_at DESC,w.id DESC LIMIT 50 OFFSET $3
 ) w ON TRUE LEFT JOIN users owner ON owner.id=w.owner_user_id
 LEFT JOIN users operator ON operator.id=w.created_by_user_id
 ORDER BY w.work_date DESC,w.created_at DESC,w.id DESC`, workspaceID, identifier, int64(page-1)*domain.EffortPageSize)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	result := &dto.ListResponse[domain.EffortWorklog]{Data: []domain.EffortWorklog{}, TotalCount: rows[0].TotalCount, Page: page, PerPage: domain.EffortPageSize}
	for _, w := range rows {
		if w.ID == nil {
			continue
		}
		result.Data = append(result.Data, domain.EffortWorklog{
			ID: *w.ID, IssueID: w.IssueID,
			Owner:    domain.EffortUser{ID: *w.OwnerID, Available: w.OwnerAvailable, Name: w.OwnerName, DisplayName: w.OwnerDisplayName},
			Operator: domain.EffortUser{ID: *w.OperatorID, Available: w.OperatorAvailable, Name: w.OperatorName, DisplayName: w.OperatorDisplayName},
			WorkDate: domain.EffortDate(w.WorkDate.Time.Format("2006-01-02")), DurationMinutes: w.DurationMinutes.Int64,
			Description: w.Description, CreatedAt: w.CreatedAt.Time, UpdatedAt: w.UpdatedAt.Time,
		})
	}
	result.HasMore = int64(page)*domain.EffortPageSize < int64(result.TotalCount)
	return result, nil
}
