package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEffortCalendarDate(t *testing.T) {
	for _, value := range []string{"2024-02-29", "0001-01-01", "9999-12-31"} {
		date, err := ParseEffortDate(value)
		require.NoError(t, err)
		data, err := json.Marshal(date)
		require.NoError(t, err)
		require.Equal(t, `"`+value+`"`, string(data))
		var decoded EffortDate
		require.NoError(t, json.Unmarshal(data, &decoded))
		require.Equal(t, date, decoded)
	}
	for _, value := range []string{"", "0000-01-01", "2023-02-29", "2026-10-17T00:00:00Z", "2026-1-01"} {
		_, err := ParseEffortDate(value)
		require.Error(t, err, value)
		var decoded EffortDate
		require.Error(t, json.Unmarshal([]byte(`"`+value+`"`), &decoded))
	}
	var decoded EffortDate
	require.Error(t, json.Unmarshal([]byte(`null`), &decoded))
	require.NoError(t, decoded.Scan(time.Date(2026, 10, 17, 0, 0, 0, 0, time.FixedZone("plus14", 14*3600))))
	require.Equal(t, EffortDate("2026-10-17"), decoded, "scanning a DATE must not convert to UTC")
	_, err := EffortDate("bad").Value()
	require.Error(t, err)
}

func TestEffortSnapshotsAndActors(t *testing.T) {
	user := uuid.New()
	source := "github"
	zero := int64(0)
	for _, actor := range []EffortActor{{Kind: "user", UserID: &user}, {Kind: "system", SystemSource: &source}} {
		original := EffortEvent{ID: uuid.New(), IssueID: uuid.New(), EventType: "remaining_set", Actor: actor, Cause: "closure", SnapshotVersion: 1,
			Before: &EffortSnapshot{Planning: &EffortPlanningSnapshot{}}, After: &EffortSnapshot{Planning: &EffortPlanningSnapshot{RemainingMinutes: &zero}}, Revision: 3}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		var restored EffortEvent
		require.NoError(t, json.Unmarshal(data, &restored))
		require.Equal(t, original, restored)
		require.Contains(t, string(data), `"estimate_minutes":null`)
		require.Contains(t, string(data), `"remaining_minutes":0`)
		require.NotContains(t, string(data), uuid.Nil.String())
	}
}

func TestEffortPermissionCeilings(t *testing.T) {
	require.True(t, IsValidScope(PermEffortsRead))
	require.True(t, IsValidScope(PermEffortsWrite))
	for _, role := range []string{RoleOwner, RoleAdmin, RoleMember, RoleGuest} {
		require.True(t, HasPermission(role, PermEffortsRead))
		require.Equal(t, role != RoleGuest, HasPermission(role, PermEffortsWrite))
	}
}

func TestDeletedEffortWorklogSnapshotRetainsAttribution(t *testing.T) {
	owner, operator, id := uuid.New(), uuid.New(), uuid.New()
	event := EffortEvent{ID: uuid.New(), IssueID: uuid.New(), EventType: "worklog_deleted",
		Actor: EffortActor{Kind: "user", UserID: &operator}, Cause: "manual", SnapshotVersion: EffortSnapshotVersion,
		Before: &EffortSnapshot{Worklog: &EffortWorklogSnapshot{ID: id, OwnerUserID: owner,
			CreatedByUserID: operator, WorkDate: "2026-10-16", DurationMinutes: 90}}, Revision: 4}
	data, err := json.Marshal(event)
	require.NoError(t, err)
	var restored EffortEvent
	require.NoError(t, json.Unmarshal(data, &restored))
	require.Equal(t, event, restored)
	require.Nil(t, restored.After)
	require.Equal(t, owner, restored.Before.Worklog.OwnerUserID)
	require.Equal(t, id, restored.Before.Worklog.ID)
}
