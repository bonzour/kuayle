package repository

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTimesheetSnapshotNumericBoundaries(t *testing.T) {
	for _, raw := range []string{`{"total_minutes":9007199254740992}`, `{"total_count":9007199254740992}`, `{"total_minutes":9223372036854775808}`, `invalid json`} {
		result, err := decodeTimesheetSnapshot([]byte(raw))
		require.Error(t, err)
		require.Nil(t, result, "an invalid snapshot must never publish partial data")
	}
	result, err := decodeTimesheetSnapshot([]byte(`{"total_minutes":9007199254740991,"total_count":9007199254740991}`))
	require.NoError(t, err)
	require.EqualValues(t, 9007199254740991, result.TotalMinutes)
	require.EqualValues(t, 9007199254740991, result.TotalCount)
}
