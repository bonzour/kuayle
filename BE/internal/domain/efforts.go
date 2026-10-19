package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	EffortMaxSafeInteger  int64 = 9007199254740991
	EffortMaxMinutes      int64 = 2147483647
	EffortPageSize              = 50
	EffortDefaultTimezone       = "Europe/Rome"
	EffortSnapshotVersion       = 1
)

// EffortDate is a calendar date, never a timezone-dependent instant.
type EffortDate string

func ParseEffortDate(value string) (EffortDate, error) {
	date, err := time.Parse(time.DateOnly, value)
	if err != nil || date.Year() < 1 || date.Format(time.DateOnly) != value {
		return "", fmt.Errorf("work_date must be a calendar date in YYYY-MM-DD format")
	}
	return EffortDate(value), nil
}

func (d *EffortDate) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := ParseEffortDate(value)
	if err == nil {
		*d = parsed
	}
	return err
}

func (d *EffortDate) Scan(value any) error {
	var text string
	switch value := value.(type) {
	case time.Time:
		text = value.Format(time.DateOnly)
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return fmt.Errorf("invalid Efforts date type %T", value)
	}
	parsed, err := ParseEffortDate(text)
	if err == nil {
		*d = parsed
	}
	return err
}

func (d EffortDate) Value() (driver.Value, error) {
	parsed, err := ParseEffortDate(string(d))
	if err != nil {
		return nil, err
	}
	return string(parsed), nil
}

type EffortSettings struct {
	Enabled  bool   `json:"enabled" db:"enabled"`
	Timezone string `json:"timezone" db:"timezone"`
	Revision int64  `json:"revision" db:"revision"`
}

type EffortValues struct {
	EstimateMinutes  *int64 `json:"estimate_minutes" db:"estimate_minutes"`
	SpentMinutes     int64  `json:"spent_minutes" db:"spent_minutes"`
	RemainingMinutes *int64 `json:"remaining_minutes" db:"remaining_minutes"`
}

type EffortTotals struct {
	EstimateMinutes  *int64 `json:"estimate_minutes" db:"estimate_minutes"`
	SpentMinutes     int64  `json:"spent_minutes" db:"spent_minutes"`
	RemainingMinutes int64  `json:"remaining_minutes" db:"remaining_minutes"`
	EstimatePartial  bool   `json:"estimate_partial" db:"estimate_partial"`
	RemainingPartial bool   `json:"remaining_partial" db:"remaining_partial"`
}

type EffortSummary struct {
	HasData               bool         `json:"has_data" db:"has_data"`
	HasVisibleDescendants bool         `json:"has_visible_descendants" db:"has_visible_descendants"`
	Revision              int64        `json:"revision" db:"revision"`
	Own                   EffortValues `json:"own" db:"own"`
	Total                 EffortTotals `json:"total" db:"total"`
}

// A missing account is an unavailable attribution, not the current operator.
type EffortUser struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Available   bool      `json:"available" db:"available"`
	Name        *string   `json:"name" db:"name"`
	DisplayName *string   `json:"display_name" db:"display_name"`
}

type EffortWorklog struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	IssueID         uuid.UUID  `json:"issue_id" db:"issue_id"`
	Owner           EffortUser `json:"owner" db:"owner"`
	Operator        EffortUser `json:"operator" db:"operator"`
	WorkDate        EffortDate `json:"work_date" db:"work_date"`
	DurationMinutes int64      `json:"duration_minutes" db:"duration_minutes"`
	Description     *string    `json:"description" db:"description"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type EffortActor struct {
	Kind         string     `json:"kind"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	SystemSource *string    `json:"system_source,omitempty"`
}

type EffortPlanningSnapshot struct {
	EstimateMinutes  *int64 `json:"estimate_minutes" db:"estimate_minutes"`
	RemainingMinutes *int64 `json:"remaining_minutes" db:"remaining_minutes"`
}

type EffortWorklogSnapshot struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	OwnerUserID     uuid.UUID  `json:"owner_user_id" db:"owner_user_id"`
	CreatedByUserID uuid.UUID  `json:"created_by_user_id" db:"created_by_user_id"`
	WorkDate        EffortDate `json:"work_date" db:"work_date"`
	DurationMinutes int64      `json:"duration_minutes" db:"duration_minutes"`
	Description     *string    `json:"description" db:"description"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// Exactly one projection is populated, as selected by event_type.
type EffortSnapshot struct {
	Planning *EffortPlanningSnapshot `json:"planning,omitempty"`
	Worklog  *EffortWorklogSnapshot  `json:"worklog,omitempty"`
}

type EffortEvent struct {
	Users           []EffortUser    `json:"users"`
	ID              uuid.UUID       `json:"id"`
	IssueID         uuid.UUID       `json:"issue_id"`
	EventType       string          `json:"event_type"`
	Actor           EffortActor     `json:"actor"`
	Cause           string          `json:"cause"`
	SnapshotVersion int             `json:"snapshot_version"`
	Before          *EffortSnapshot `json:"before"`
	After           *EffortSnapshot `json:"after"`
	Revision        int64           `json:"revision"`
	CreatedAt       time.Time       `json:"created_at"`
}
