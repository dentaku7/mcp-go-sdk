package graph

import (
	"os"
	"path/filepath"
	"testing"

	"mcp-memory/internal/types"
)

func TestKnowledgeGraphManager(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_memory.json")

	// Create a new manager instance
	manager := NewKnowledgeGraphManager(tmpFile)

	// Test creating entities
	t.Run("CreateEntities", func(t *testing.T) {
		entities := []types.Entity{
			{
				ID:   "e1",
				Type: "person",
				Name: "John Doe",
			},
			{
				ID:   "e2",
				Type: "person",
				Name: "Jane Smith",
			},
		}

		created, err := manager.CreateEntities(entities)
		if err != nil {
			t.Errorf("Failed to create entities: %v", err)
		}

		if len(created) != len(entities) {
			t.Errorf("Expected %d entities to be created, got %d", len(entities), len(created))
		}

		// Try creating duplicate entity
		_, err = manager.CreateEntities([]types.Entity{entities[0]})
		if err == nil {
			t.Error("Expected error when creating duplicate entity")
		}

		// Test auto-generation of IDs
		autoGenEntities := []types.Entity{
			{
				Type: "person",
				Name: "Auto Gen 1",
			},
			{
				Type: "person",
				Name: "Auto Gen 2",
			},
		}

		created, err = manager.CreateEntities(autoGenEntities)
		if err != nil {
			t.Errorf("Failed to create entities with auto-generated IDs: %v", err)
		}

		if len(created) != len(autoGenEntities) {
			t.Errorf("Expected %d entities to be created, got %d", len(autoGenEntities), len(created))
		}

		for _, entity := range created {
			if entity.ID == "" {
				t.Error("Expected auto-generated ID to be non-empty")
			}
		}
	})

	// Test creating relations
	t.Run("CreateRelations", func(t *testing.T) {
		relations := []types.Relation{
			{
				ID:     "r1",
				Type:   "knows",
				Source: "e1",
				Target: "e2",
			},
		}

		created, err := manager.CreateRelations(relations)
		if err != nil {
			t.Errorf("Failed to create relations: %v", err)
		}

		if len(created) != len(relations) {
			t.Errorf("Expected %d relations to be created, got %d", len(relations), len(created))
		}

		// Try creating relation with non-existent entity
		_, err = manager.CreateRelations([]types.Relation{{
			ID:     "r2",
			Type:   "knows",
			Source: "e1",
			Target: "e3", // Non-existent
		}})
		if err == nil {
			t.Error("Expected error when creating relation with non-existent entity")
		}

		// Test auto-generation of IDs
		autoGenRelations := []types.Relation{
			{
				Type:   "knows",
				Source: "e1",
				Target: "e2",
			},
		}

		created, err = manager.CreateRelations(autoGenRelations)
		if err != nil {
			t.Errorf("Failed to create relations with auto-generated IDs: %v", err)
		}

		if len(created) != len(autoGenRelations) {
			t.Errorf("Expected %d relations to be created, got %d", len(autoGenRelations), len(created))
		}

		for _, relation := range created {
			if relation.ID == "" {
				t.Error("Expected auto-generated ID to be non-empty")
			}
		}
	})

	// Test adding observations
	t.Run("AddObservations", func(t *testing.T) {
		observations := []types.Observation{
			{
				ID:       "o1",
				EntityID: "e1",
				Type:     "hobby",
				Content:  "Likes hiking",
			},
		}

		created, err := manager.AddObservations(observations)
		if err != nil {
			t.Errorf("Failed to add observations: %v", err)
		}

		if len(created) != len(observations) {
			t.Errorf("Expected %d observations to be created, got %d", len(observations), len(created))
		}

		// Try adding observation to non-existent entity
		_, err = manager.AddObservations([]types.Observation{{
			ID:       "o2",
			EntityID: "e3", // Non-existent
			Type:     "hobby",
			Content:  "Likes swimming",
		}})
		if err == nil {
			t.Error("Expected error when adding observation to non-existent entity")
		}

		// Test auto-generation of IDs
		autoGenObservations := []types.Observation{
			{
				EntityID: "e1",
				Type:     "skill",
				Content:  "Expert in Python programming",
			},
		}

		created, err = manager.AddObservations(autoGenObservations)
		if err != nil {
			t.Errorf("Failed to add observations with auto-generated IDs: %v", err)
		}

		if len(created) != len(autoGenObservations) {
			t.Errorf("Expected %d observations to be created, got %d", len(autoGenObservations), len(created))
		}

		for _, observation := range created {
			if observation.ID == "" {
				t.Error("Expected auto-generated ID to be non-empty")
			}
		}
	})

	// Test reading graph
	t.Run("ReadGraph", func(t *testing.T) {
		graph := manager.ReadGraph()

		if len(graph.Entities) != 4 { // e1, e2, and two auto-generated
			t.Errorf("Expected 4 entities, got %d", len(graph.Entities))
		}

		if len(graph.Relations) != 2 { // r1 and one auto-generated
			t.Errorf("Expected 2 relations, got %d", len(graph.Relations))
		}

		if len(graph.Observations) != 2 { // o1 and one auto-generated
			t.Errorf("Expected 2 observations, got %d", len(graph.Observations))
		}
	})

	// Test searching nodes
	t.Run("SearchNodes", func(t *testing.T) {
		// Search by type
		results := manager.SearchNodes("person", nil)
		if len(results) != 4 { // All entities are persons
			t.Errorf("Expected 4 results when searching by type 'person', got %d", len(results))
		}

		// Search by metadata
		metadata := map[string]interface{}{
			"role": "developer",
		}
		results = manager.SearchNodes("", metadata)
		if len(results) != 0 { // No entities have this metadata
			t.Errorf("Expected 0 results when searching by metadata, got %d", len(results))
		}
	})

	// Test text-based search
	t.Run("SearchByText", func(t *testing.T) {
		// Add test data
		observations := []types.Observation{
			{
				ID:          "o2",
				EntityID:    "e1",
				Type:        "skill",
				Content:     "Expert in Python programming",
				Description: "Programming skills",
			},
		}
		_, err := manager.AddObservations(observations)
		if err != nil {
			t.Fatalf("Failed to add observation: %v", err)
		}

		// Test cases
		tests := []struct {
			name     string
			query    string
			expected int
		}{
			{"Search by name", "john", 1},
			{"Search by type", "person", 4}, // All entities are persons
			{"Search by observation content", "python", 1},
			{"Search by observation type", "skill", 1},
			{"Search by description", "programming", 1},
			{"Search with no matches", "nonexistent", 0},
			{"Empty query", "", 0},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				results := manager.SearchByText(tc.query)
				if len(results) != tc.expected {
					t.Errorf("Expected %d results for query '%s', got %d", tc.expected, tc.query, len(results))
				}
			})
		}
	})

	// Test opening nodes
	t.Run("OpenNodes", func(t *testing.T) {
		result, err := manager.OpenNodes([]string{"e1"})
		if err != nil {
			t.Errorf("Failed to open nodes: %v", err)
		}

		if len(result.Entities) != 1 { // e1
			t.Errorf("Expected 1 entity when opening node e1, got %d", len(result.Entities))
		}

		if len(result.Relations) != 2 { // r1 and one auto-generated
			t.Errorf("Expected 2 relations when opening node e1, got %d", len(result.Relations))
		}
	})

	// Test deleting observations
	t.Run("DeleteObservations", func(t *testing.T) {
		// Get all observation IDs
		graph := manager.ReadGraph()
		observationIDs := make([]string, 0, len(graph.Observations))
		for id := range graph.Observations {
			observationIDs = append(observationIDs, id)
		}

		err := manager.DeleteObservations(observationIDs)
		if err != nil {
			t.Errorf("Failed to delete observations: %v", err)
		}

		graph = manager.ReadGraph()
		if len(graph.Observations) != 0 {
			t.Errorf("Expected 0 observations after deletion, got %d", len(graph.Observations))
		}

		// Try deleting non-existent observation
		err = manager.DeleteObservations([]string{"o3"})
		if err == nil {
			t.Error("Expected error when deleting non-existent observation")
		}
	})

	// Test deleting relations
	t.Run("DeleteRelations", func(t *testing.T) {
		// Get all relations
		graph := manager.ReadGraph()
		relations := make([]types.Relation, 0, len(graph.Relations))
		for _, relation := range graph.Relations {
			relations = append(relations, relation)
		}

		err := manager.DeleteRelations(relations)
		if err != nil {
			t.Errorf("Failed to delete relations: %v", err)
		}

		graph = manager.ReadGraph()
		if len(graph.Relations) != 0 {
			t.Errorf("Expected 0 relations after deletion, got %d", len(graph.Relations))
		}
	})

	// Test deleting entities
	t.Run("DeleteEntities", func(t *testing.T) {
		// Get all entity IDs
		graph := manager.ReadGraph()
		entityIDs := make([]string, 0, len(graph.Entities))
		for id := range graph.Entities {
			entityIDs = append(entityIDs, id)
		}

		err := manager.DeleteEntities(entityIDs)
		if err != nil {
			t.Errorf("Failed to delete entities: %v", err)
		}

		graph = manager.ReadGraph()
		if len(graph.Entities) != 0 {
			t.Errorf("Expected 0 entities after deletion, got %d", len(graph.Entities))
		}

		// Try deleting non-existent entity
		err = manager.DeleteEntities([]string{"e3"})
		if err == nil {
			t.Error("Expected error when deleting non-existent entity")
		}
	})

	// Clean up
	os.Remove(tmpFile)
}

// Test file persistence
func TestKnowledgeGraphPersistence(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_memory.json")

	// Create a new manager instance
	manager1 := NewKnowledgeGraphManager(tmpFile)

	// Add some data
	entities := []types.Entity{
		{
			ID:   "e1",
			Type: "person",
			Name: "John Doe",
		},
	}

	_, err := manager1.CreateEntities(entities)
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a new manager instance with the same file
	manager2 := NewKnowledgeGraphManager(tmpFile)

	// Check if data persisted
	graph := manager2.ReadGraph()
	if len(graph.Entities) != 1 {
		t.Errorf("Expected 1 entity after loading from file, got %d", len(graph.Entities))
	}

	if entity, exists := graph.Entities["e1"]; !exists {
		t.Error("Entity e1 not found after loading from file")
	} else if entity.Name != "John Doe" {
		t.Errorf("Expected entity name 'John Doe', got '%s'", entity.Name)
	}

	// Clean up
	os.Remove(tmpFile)
}

// Test error cases
func TestKnowledgeGraphErrors(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_memory.json")

	// Create a new manager instance
	manager := NewKnowledgeGraphManager(tmpFile)

	// Test creating relation with empty source/target
	t.Run("CreateRelationEmptySource", func(t *testing.T) {
		_, err := manager.CreateRelations([]types.Relation{{
			Type:   "knows",
			Source: "",
			Target: "e2",
		}})
		if err == nil {
			t.Error("Expected error when creating relation with empty source")
		}
	})

	// Test creating observation with empty entity ID
	t.Run("CreateObservationEmptyEntityID", func(t *testing.T) {
		_, err := manager.AddObservations([]types.Observation{{
			Type:    "hobby",
			Content: "Likes hiking",
		}})
		if err == nil {
			t.Error("Expected error when creating observation with empty entity ID")
		}
	})

	// Clean up
	os.Remove(tmpFile)
}
