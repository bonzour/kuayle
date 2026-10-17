package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/pkg/validate"
	"github.com/stretchr/testify/require"
)

func TestEffortEstimatePresenceAndPrecision(t *testing.T) {
	prefix := `"request_id":"` + uuid.NewString() + `","expected_revision":0`
	for _, value := range []string{"null", "0", "2147483647"} {
		var req SetEffortEstimateRequest
		require.NoError(t, json.Unmarshal([]byte(`{`+prefix+`,"estimate_minutes":`+value+`}`), &req))
		require.NoError(t, validate.Struct(req), value)
		require.True(t, req.EstimateMinutes.Set)
	}
	for _, body := range []string{`{` + prefix + `}`, `{` + prefix + `,"estimate_minutes":-1}`, `{` + prefix + `,"estimate_minutes":2147483648}`, `{"estimate_minutes":0}`} {
		var req SetEffortEstimateRequest
		require.NoError(t, json.Unmarshal([]byte(body), &req))
		require.Error(t, validate.Struct(req), body)
	}
	var req SetEffortEstimateRequest
	require.Error(t, json.Unmarshal([]byte(`{`+prefix+`,"estimate_minutes":1.5}`), &req))
	require.Error(t, json.Unmarshal([]byte(`{`+prefix+`,"estimate_minutes":"1h"}`), &req))
}

func TestEffortWorklogInputBounds(t *testing.T) {
	zero := int64(0)
	req := CreateEffortWorklogRequest{EffortMutationRequest: EffortMutationRequest{RequestID: uuid.NewString(), ExpectedRevision: &zero},
		OwnerUserID: uuid.NewString(), WorkDate: "2026-10-17", DurationMinutes: 1}
	require.NoError(t, validate.Struct(req))
	for _, duration := range []int64{0, -1, 1441} {
		req.DurationMinutes = duration
		require.Error(t, validate.Struct(req))
	}
	req.DurationMinutes = 1440
	text := strings.Repeat("界", 2000)
	req.Description = &text
	require.NoError(t, validate.Struct(req))
	text += "界"
	require.Error(t, validate.Struct(req))
}
