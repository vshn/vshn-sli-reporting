package types

import "time"

type DowntimeWindow struct {
	ID           string                   `json:"id"`
	StartTime    *time.Time               `json:"start_time"`
	EndTime      *time.Time               `json:"end_time,omitempty"`
	Title        string                   `json:"title"`
	Description  string                   `json:"description,omitempty"`
	ExternalID   string                   `json:"external_id,omitempty"`
	ExternalLink string                   `json:"external_link,omitempty"`
	Affects      []AffectedClusterMatcher `json:"affects"`
}

type AffectedClusterMatcher = map[string]string
