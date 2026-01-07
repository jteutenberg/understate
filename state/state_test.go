package state

import (
	"github.com/jteutenberg/understate/core"
	"testing"
)

func TestFactAnswer(t *testing.T) {
	cow := Atomic{Index: 1, Value: "cow"}
	grass := Atomic{Index: 2, Value: "grass"}
	plantType := Type{
		Name: "Plant",
		Atomics: AtomicSet{
			grass: true,
		},
	}
	herbivoreType := Type{
		Name: "Herbivore",
		Atomics: AtomicSet{
			cow: true,
		},
	}
	// true fact: cow eats grass
	eatDef := &PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []ArgumentDefinition{
			{Label: "Eater", Type: &herbivoreType},
			{Label: "Food", Type: &plantType},
		},
	}
	trueFact := &Predicate{
		Definition: eatDef,
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*VariableReference{
			{Label: "X", Ref: &cow},
			{Label: "Y", Ref: &grass},
		},
	}
	// false fact: cow does not eat cow
	falseFact := &Predicate{
		Definition: eatDef,
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*VariableReference{
			{Label: "X", Ref: &cow},
			{Label: "Y", Ref: &cow},
		},
	}
	state := &State{
		TrueFacts: map[string][]*Predicate{
			"eat": {trueFact},
		},
		FalseFacts: map[string][]*Predicate{
			"eat": {falseFact},
		},
	}
	//find what cows eat
	query := &Predicate{
		Definition: eatDef,
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*VariableReference{
			{Label: "X", Ref: &cow},
			{Label: "Y", Ref: nil},
		},
	}
	answer, _ := state.Answer(query)
	ansCount := 0
	for ans := range answer {
		ansCount++
		if ans[1] != &grass || ans[0] != &cow {
			t.Errorf("expected answer to be grass, got %v", ans[1])
		}
	}
	if ansCount != 1 {
		t.Errorf("expected 1 answer, got %d", ansCount)
	}
}
