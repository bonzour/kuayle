package repository

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kuayle/kuayle-backend/internal/domain"
)

type IssueHistoryRepository struct {
	db *sqlx.DB
}

func NewIssueHistoryRepository(db *sqlx.DB) *IssueHistoryRepository {
	return &IssueHistoryRepository{db: db}
}

func (r *IssueHistoryRepository) Create(ctx context.Context, issueID, userID uuid.UUID, field string, oldValue, newValue *string) error {
	id := uuid.New()
	_, err := r.db.ExecContext(ctx, `INSERT INTO issue_history (id, issue_id, user_id, field, old_value, new_value) VALUES ($1, $2, $3, $4, $5, $6)`, id, issueID, userID, field, oldValue, newValue)
	return err
}

func (r *IssueHistoryRepository) ListByIssue(ctx context.Context, issueID uuid.UUID) ([]domain.IssueHistory, error) {
	var entries []domain.IssueHistory
	err := r.db.SelectContext(ctx, &entries, `SELECT h.id,h.issue_id,h.user_id,h.field,`+historyReference(ctx, "h.old_value")+` AS old_value,`+historyReference(ctx, "h.new_value")+` AS new_value,h.created_at FROM issue_history h WHERE h.issue_id = $1 AND `+issueVisible(ctx, "h.issue_id")+` ORDER BY h.created_at DESC`, issueID)
	if err != nil {
		return nil, err
	}
	if domain.CanReadEfforts(ctx) {
		var raw []string
		err = r.db.SelectContext(ctx, &raw, `SELECT jsonb_build_object(
   'id',e.id,'issue_id',e.issue_id,'event_type',e.event_type,'cause',e.cause,
   'actor',jsonb_strip_nulls(jsonb_build_object('kind',e.actor_kind,'user_id',e.actor_user_id,'system_source',e.system_source)),
   'snapshot_version',e.snapshot_version,'before',e.before_snapshot,'after',e.after_snapshot,
   'revision',e.revision,'created_at',e.created_at,'users',COALESCE((
    SELECT jsonb_agg(jsonb_build_object('id',refs.id,'available',u.id IS NOT NULL,'name',u.name,'display_name',u.display_name))
    FROM (SELECT DISTINCT id FROM unnest(ARRAY[e.actor_user_id,
     (e.before_snapshot->'worklog'->>'owner_user_id')::uuid,(e.after_snapshot->'worklog'->>'owner_user_id')::uuid,
     (e.before_snapshot->'worklog'->>'created_by_user_id')::uuid,(e.after_snapshot->'worklog'->>'created_by_user_id')::uuid]) id WHERE id IS NOT NULL) refs
    LEFT JOIN users u ON u.id=refs.id),'[]'::jsonb))::text
   FROM effort_events e WHERE e.issue_id=$1 AND `+effortWorkspaceVisible(ctx, "e.workspace_id")+` AND `+issueVisible(ctx, "e.issue_id"), issueID)
		if err != nil {
			return nil, err
		}
		for _, value := range raw {
			var event domain.EffortEvent
			if err := json.Unmarshal([]byte(value), &event); err != nil {
				return nil, err
			}
			entry := domain.IssueHistory{ID: event.ID, IssueID: event.IssueID, Field: "efforts", CreatedAt: event.CreatedAt, Effort: &event}
			if event.Actor.UserID != nil {
				entry.UserID = *event.Actor.UserID
			}
			entries = append(entries, entry)
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
				return entries[i].ID.String() > entries[j].ID.String()
			}
			return entries[i].CreatedAt.After(entries[j].CreatedAt)
		})
	}
	return entries, nil
}

// History may predate a team's privacy change and outlive the relationship.
// Redact the raw reference as well as preventing display-name enrichment.
func historyReference(ctx context.Context, value string) string {
	return "CASE WHEN " + value + " IS NULL OR " + value + "='' THEN " + value +
		" WHEN h.field IN ('project','project_id') THEN CASE WHEN EXISTS (SELECT 1 FROM projects p WHERE p.id::text=LOWER(BTRIM(" + value + ")) AND " + projectVisible(ctx, "p.workspace_id", "p.team_id") + ") THEN " + value + " ELSE NULL END" +
		" WHEN h.field IN ('cycle','cycle_id') THEN CASE WHEN EXISTS (SELECT 1 FROM cycles c WHERE c.id::text=LOWER(BTRIM(" + value + ")) AND " + teamVisible(ctx, "c.team_id") + ") THEN " + value + " ELSE NULL END" +
		" WHEN h.field IN ('parent','parent_id') THEN CASE WHEN EXISTS (SELECT 1 FROM issues parent WHERE parent.id::text=LOWER(BTRIM(" + value + ")) AND " + teamVisible(ctx, "parent.team_id") + ") THEN " + value + " ELSE NULL END" +
		" WHEN h.field='status_id' THEN CASE WHEN EXISTS (SELECT 1 FROM team_statuses s WHERE s.id::text=LOWER(BTRIM(" + value + ")) AND " + teamVisible(ctx, "s.team_id") + ") THEN " + value + " ELSE NULL END" +
		" ELSE " + value + " END"
}
