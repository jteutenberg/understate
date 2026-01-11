package core

import (
	"testing"

	"github.com/jteutenberg/bitset-go"
)

func TestSimpleUnify(t *testing.T) {
	cow := Atomic{Index: 1, Value: "cow"}
	grass := Atomic{Index: 2, Value: "grass"}
	plantType := Type{
		Name:    "Plant",
		Atomics: bitset.NewIntSetFromInts([]int{grass.Index}),
	}
	herbivoreType := Type{
		Name:    "Herbivore",
		Atomics: bitset.NewIntSetFromInts([]int{cow.Index}),
	}
	def := &PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []ArgumentDefinition{
			{Label: "Eater", Type: &herbivoreType},
			{Label: "Food", Type: &plantType},
		},
	}

	predA := &Predicate{
		Definition: def,
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*VariableReference{
			{Label: "X", Ref: &cow},
			{Label: "Y", Ref: nil},
		},
	}
	predB := &Predicate{
		Definition: def,
		VarLabels:  []string{"?3", "?4"},
		VarRefs: []*VariableReference{
			{Label: "A", Ref: nil},
			{Label: "B", Ref: &grass},
		},
	}
	if !predA.CanUnify(predB) {
		t.Errorf("predicates cannot unify")
	}
	err := predA.Unify(predB)
	if err != nil {
		t.Errorf("error unifying predicates: %v", err)
	}
	if predA.String() != "eat(cow, grass)" {
		t.Errorf("unified predicate: %v", predA)
	}
}
