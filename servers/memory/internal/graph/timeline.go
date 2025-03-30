package graph

import (
	"fmt"
	"sort"
	"time"

	"mcp-memory/internal/types"
)

// GetEntityTimeline retrieves observations for a specific entity, filtered by time and optional criteria.
// Results are sorted chronologically (ascending).
func (m *KnowledgeGraphManager) GetEntityTimeline(entityID string, startTime, endTime time.Time, observationType string, tags []string) ([]types.Observation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. Check if entity exists
	if _, exists := m.graph.Entities[entityID]; !exists {
		return nil, fmt.Errorf("entity with ID %s does not exist", entityID)
	}

	// 2. Filter observations
	var timeline []types.Observation
	for _, obs := range m.graph.Observations {
		// Filter by EntityID
		if obs.EntityID != entityID {
			continue
		}

		// Filter by Timestamp
		if !startTime.IsZero() && obs.Timestamp.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && obs.Timestamp.After(endTime) {
			continue
		}

		// Filter by Observation Type
		if observationType != "" && obs.Type != observationType {
			continue
		}

		// Filter by Tags (match if any provided tag exists in observation tags)
		if len(tags) > 0 {
			tagMatch := false
			tagSet := make(map[string]struct{}, len(obs.Tags)) // Use a set for efficient lookup
			for _, obsTag := range obs.Tags {
				tagSet[obsTag] = struct{}{}
			}
			for _, userTag := range tags {
				if _, found := tagSet[userTag]; found {
					tagMatch = true
					break
				}
			}
			if !tagMatch {
				continue
			}
		}

		// Passed all filters
		timeline = append(timeline, obs)
	}

	// 3. Sort by timestamp (ascending)
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})

	return timeline, nil
}
