package types

import "time"

// Entity represents a node in the knowledge graph
type Entity struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Relation represents a connection between two entities
type Relation struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	Target        string                 `json:"target"`
	Description   string                 `json:"description,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Weight        float64                `json:"weight,omitempty"`
	Bidirectional bool                   `json:"bidirectional,omitempty"`
}

// Observation represents a piece of information about an entity
type Observation struct {
	ID          string                 `json:"id"`
	EntityID    string                 `json:"entity_id"`
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Tags        []string               `json:"tags,omitempty"`
}

// KnowledgeGraph represents the in-memory graph structure
type KnowledgeGraph struct {
	Entities     map[string]Entity      `json:"entities"`
	Relations    map[string]Relation    `json:"relations"`
	Observations map[string]Observation `json:"observations"`
}

// KnowledgeGraphResult represents the graph structure with map-based storage
type KnowledgeGraphResult struct {
	Entities  map[string]Entity   `json:"entities"`
	Relations map[string]Relation `json:"relations"`
}

// EntityFilterCriteria defines criteria for filtering entities
type EntityFilterCriteria struct {
	Type                string `json:"type,omitempty"`
	NameContains        string `json:"name_contains,omitempty"`
	DescriptionContains string `json:"description_contains,omitempty"`
	// TODO: Add metadata filters? (e.g., MetadataHasKey, MetadataEquals)
}
