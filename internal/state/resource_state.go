package state

import "time"

// ResourceState is the persisted state record for a single managed resource.
// This mirrors the interfaces.ResourceState definition from the workflow engine.
type ResourceState struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Provider       string         `json:"provider"`
	ProviderID     string         `json:"provider_id"`
	ConfigHash     string         `json:"config_hash"`
	AppliedConfig  map[string]any `json:"applied_config"`
	Outputs        map[string]any `json:"outputs"`
	Dependencies   []string       `json:"dependencies"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	LastDriftCheck time.Time      `json:"last_drift_check,omitempty"`
}
