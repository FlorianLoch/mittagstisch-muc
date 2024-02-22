package schema

import (
	"entgo.io/ent"
)

// Place holds the schema definition for the Place entity.
type Place struct {
	ent.Schema
}

// Fields of the Place.
func (Place) Fields() []ent.Field {
	return []ent.Field{}
}

func (Place) Indexes() []ent.Index {
	return []ent.Index{}
}

// Edges of the Place.
func (Place) Edges() []ent.Edge {
	return nil
}
