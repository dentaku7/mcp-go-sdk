package graph_test

import (
	"mcp-memory/internal/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetEntityTimeline tests the GetEntityTimeline method
func TestGetEntityTimeline(t *testing.T) {
	manager, _ := setupTestManager(t)

	// --- Setup Data --- //
	entityID := "test-entity-1"
	otherEntityID := "other-entity-2"

	_, err := manager.CreateEntities([]types.Entity{
		{ID: entityID, Type: "test", Name: "Test Entity"},
		{ID: otherEntityID, Type: "test", Name: "Other Entity"},
	})
	require.NoError(t, err, "Failed to create test entities")

	// Observations for test-entity-1
	time1 := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 1, 2, 9, 0, 0, 0, time.UTC)
	time4 := time.Date(2023, 1, 2, 15, 0, 0, 0, time.UTC)
	time5 := time.Date(2023, 1, 3, 11, 0, 0, 0, time.UTC)

	obs1 := types.Observation{ID: "obs1", EntityID: entityID, Type: "log", Content: "Log entry 1", Timestamp: time1, Tags: []string{"system", "info"}}
	obs2 := types.Observation{ID: "obs2", EntityID: entityID, Type: "event", Content: "Event A occurred", Timestamp: time2, Tags: []string{"critical", "alert"}}
	obs3 := types.Observation{ID: "obs3", EntityID: entityID, Type: "log", Content: "Log entry 2", Timestamp: time3, Tags: []string{"system", "debug"}}
	obs4 := types.Observation{ID: "obs4", EntityID: entityID, Type: "metric", Content: "CPU usage: 50%", Timestamp: time4, Tags: []string{"performance"}}
	obs5 := types.Observation{ID: "obs5", EntityID: entityID, Type: "event", Content: "Event B occurred", Timestamp: time5, Tags: []string{"info"}}

	// Observation for other-entity-2
	otherObs := types.Observation{ID: "otherObs", EntityID: otherEntityID, Type: "log", Content: "Other entity log", Timestamp: time1}

	_, err = manager.AddObservations([]types.Observation{obs1, obs2, obs3, obs4, obs5, otherObs})
	require.NoError(t, err, "Failed to add test observations")

	// --- Test Cases --- //
	tests := []struct {
		name             string
		entityID         string
		startTime        time.Time
		endTime          time.Time
		observationType  string
		tags             []string
		expectedObsIDs   []string // Expected IDs in chronological order
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name:           "Get all observations for entity",
			entityID:       entityID,
			expectedObsIDs: []string{"obs1", "obs2", "obs3", "obs4", "obs5"},
		},
		{
			name:           "Filter by start time",
			entityID:       entityID,
			startTime:      time3, // Include obs3 onwards
			expectedObsIDs: []string{"obs3", "obs4", "obs5"},
		},
		{
			name:           "Filter by end time",
			entityID:       entityID,
			endTime:        time2, // Include obs1, obs2
			expectedObsIDs: []string{"obs1", "obs2"},
		},
		{
			name:           "Filter by time range",
			entityID:       entityID,
			startTime:      time2, // Include obs2, obs3, obs4
			endTime:        time4,
			expectedObsIDs: []string{"obs2", "obs3", "obs4"},
		},
		{
			name:            "Filter by type 'log'",
			entityID:        entityID,
			observationType: "log",
			expectedObsIDs:  []string{"obs1", "obs3"},
		},
		{
			name:            "Filter by type 'event'",
			entityID:        entityID,
			observationType: "event",
			expectedObsIDs:  []string{"obs2", "obs5"},
		},
		{
			name:            "Filter by type 'metric'",
			entityID:        entityID,
			observationType: "metric",
			expectedObsIDs:  []string{"obs4"},
		},
		{
			name:            "Filter by non-existent type",
			entityID:        entityID,
			observationType: "nonexistent",
			expectedObsIDs:  []string{}, // Empty slice
		},
		{
			name:           "Filter by single tag 'system'",
			entityID:       entityID,
			tags:           []string{"system"},
			expectedObsIDs: []string{"obs1", "obs3"}, // obs1 and obs3 have 'system'
		},
		{
			name:           "Filter by single tag 'info'",
			entityID:       entityID,
			tags:           []string{"info"},
			expectedObsIDs: []string{"obs1", "obs5"}, // obs1 and obs5 have 'info'
		},
		{
			name:           "Filter by multiple tags (OR logic) 'alert' or 'performance'",
			entityID:       entityID,
			tags:           []string{"alert", "performance"},
			expectedObsIDs: []string{"obs2", "obs4"}, // obs2 has 'alert', obs4 has 'performance'
		},
		{
			name:           "Filter by tag present in multiple observations ('system', 'info')",
			entityID:       entityID,
			tags:           []string{"system", "info"},
			expectedObsIDs: []string{"obs1", "obs3", "obs5"}, // obs1(sys,info), obs3(sys), obs5(info)
		},
		{
			name:           "Filter by non-existent tag",
			entityID:       entityID,
			tags:           []string{"nonexistenttag"},
			expectedObsIDs: []string{}, // Empty slice
		},
		{
			name:            "Filter by type and time range",
			entityID:        entityID,
			startTime:       time2,
			endTime:         time5,
			observationType: "log",
			expectedObsIDs:  []string{"obs3"}, // Only obs3 is 'log' between time2 and time5
		},
		{
			name:           "Filter by tag and time range",
			entityID:       entityID,
			startTime:      time1,
			endTime:        time3,
			tags:           []string{"critical"},
			expectedObsIDs: []string{"obs2"}, // Only obs2 has 'critical' between time1 and time3
		},
		{
			name:            "Filter by type and tag",
			entityID:        entityID,
			observationType: "log",
			tags:            []string{"info"},
			expectedObsIDs:  []string{"obs1"}, // Only obs1 is 'log' AND has 'info'
		},
		{
			name:            "Filter by type, tag, and time range",
			entityID:        entityID,
			startTime:       time1,
			endTime:         time5,
			observationType: "event",
			tags:            []string{"info"},
			expectedObsIDs:  []string{"obs5"}, // Only obs5 is 'event', has 'info', and is in range
		},
		{
			name:           "Get timeline for other entity",
			entityID:       otherEntityID,
			expectedObsIDs: []string{"otherObs"}, // Only the one observation for this entity
		},
		{
			name:             "Get timeline for non-existent entity",
			entityID:         "non-existent-entity",
			expectedObsIDs:   []string{}, // Should be empty
			expectError:      true,
			expectedErrorMsg: "entity with ID non-existent-entity does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeline, err := manager.GetEntityTimeline(tt.entityID, tt.startTime, tt.endTime, tt.observationType, tt.tags)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)

				// Check the number of results
				assert.Len(t, timeline, len(tt.expectedObsIDs), "Incorrect number of observations returned")

				// Check the IDs and order
				actualIDs := make([]string, len(timeline))
				for i, obs := range timeline {
					actualIDs[i] = obs.ID
				}
				assert.Equal(t, tt.expectedObsIDs, actualIDs, "Observation IDs or order mismatch")
			}
		})
	}
}
