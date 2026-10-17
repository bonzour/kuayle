package dto

import (
	"encoding/json"

	"github.com/kuayle/kuayle-backend/internal/domain"
)

// Read projections deliberately stay out of generic and public issue DTOs.
type EffortSettingsResponse = domain.EffortSettings
type EffortSummaryResponse = domain.EffortSummary
type EffortWorklogResponse = domain.EffortWorklog

const (
	EffortRevisionConflict = "EFFORT_REVISION_CONFLICT"
	EffortRequestConflict  = "EFFORT_REQUEST_CONFLICT"
	EffortDailyLimit       = "EFFORT_DAILY_LIMIT"
	EffortDisabled         = "EFFORT_DISABLED"
)

// Internal command contracts; HTTP mutations remain unregistered.
type EffortMutationRequest struct {
	RequestID        string `json:"request_id" validate:"required,uuid"`
	ExpectedRevision *int64 `json:"expected_revision" validate:"required,min=0,max=9007199254740991"`
}

// Unlike a pointer alone, Set distinguishes omission from explicit removal.
type OptionalEffortMinutes struct {
	Value *int64 `validate:"omitempty,min=0,max=2147483647"`
	Set   bool   `validate:"eq=true"`
}

func (m *OptionalEffortMinutes) UnmarshalJSON(data []byte) error {
	m.Set = true
	return json.Unmarshal(data, &m.Value)
}

func (m OptionalEffortMinutes) MarshalJSON() ([]byte, error) { return json.Marshal(m.Value) }

type SetEffortEstimateRequest struct {
	EffortMutationRequest
	EstimateMinutes OptionalEffortMinutes `json:"estimate_minutes"`
}

type SetEffortRemainingRequest struct {
	EffortMutationRequest
	RemainingMinutes *int64 `json:"remaining_minutes" validate:"required,min=0,max=2147483647"`
}

type CreateEffortWorklogRequest struct {
	EffortMutationRequest
	OwnerUserID     string            `json:"owner_user_id" validate:"required,uuid"`
	WorkDate        domain.EffortDate `json:"work_date" validate:"required"`
	DurationMinutes int64             `json:"duration_minutes" validate:"min=1,max=1440"`
	Description     *string           `json:"description" validate:"omitempty,max=2000"`
}

// All editable values are submitted together; owner is not editable.
type UpdateEffortWorklogRequest struct {
	EffortMutationRequest
	WorkDate        domain.EffortDate `json:"work_date" validate:"required"`
	DurationMinutes int64             `json:"duration_minutes" validate:"min=1,max=1440"`
	Description     *string           `json:"description" validate:"omitempty,max=2000"`
}

type DeleteEffortWorklogRequest struct{ EffortMutationRequest }

// Configuration is a complete desired value, restricted to interactive admins.
type SetEffortSettingsRequest struct {
	EffortMutationRequest
	Enabled  *bool  `json:"enabled" validate:"required"`
	Timezone string `json:"timezone" validate:"required"`
}

type EffortMutationReceipt struct {
	RequestID  string `json:"request_id"`
	ResourceID string `json:"resource_id"`
	Revision   int64  `json:"revision"`
}
