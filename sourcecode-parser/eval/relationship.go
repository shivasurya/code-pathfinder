package eval

// RelationshipMap represents relationships between entities and their attributes.
type RelationshipMap struct {
	// map[EntityName]map[RelatedEntityName]bool
	DirectRelationships map[string]map[string]bool
	// Original relationships for attribute-based queries
	Relationships map[string]map[string][]string
}

// NewRelationshipMap creates a new RelationshipMap.
func NewRelationshipMap() *RelationshipMap {
	return &RelationshipMap{
		DirectRelationships: make(map[string]map[string]bool),
		Relationships:       make(map[string]map[string][]string),
	}
}

// AddRelationship adds a relationship between an entity and its related entities through an attribute.
func (rm *RelationshipMap) AddRelationship(entity, attribute string, relatedEntities []string) {
	// Store the original relationship structure
	if rm.Relationships[entity] == nil {
		rm.Relationships[entity] = make(map[string][]string)
	}
	rm.Relationships[entity][attribute] = relatedEntities

	// Also store direct entity-to-entity relationships for faster lookups
	for _, related := range relatedEntities {
		// Create entity1 -> entity2 relationship
		if rm.DirectRelationships[entity] == nil {
			rm.DirectRelationships[entity] = make(map[string]bool)
		}
		rm.DirectRelationships[entity][related] = true

		// Create entity2 -> entity1 relationship (bidirectional)
		if rm.DirectRelationships[related] == nil {
			rm.DirectRelationships[related] = make(map[string]bool)
		}
		rm.DirectRelationships[related][entity] = true
	}
}

// HasRelationship checks if two entities are related through any attribute.
func (rm *RelationshipMap) HasRelationship(entity1, entity2 string) bool {
	// Use the optimized direct relationship lookup
	if relatedEntities, ok := rm.DirectRelationships[entity1]; ok {
		if _, related := relatedEntities[entity2]; related {
			return true
		}
	}

	return false
}
