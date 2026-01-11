package state

import (
	"testing"

	"github.com/jteutenberg/bitset-go"
	"github.com/jteutenberg/understate/core"
)

func TestFactAnswer(t *testing.T) {
	cow := core.Atomic{Index: 1, Value: "cow"}
	grass := core.Atomic{Index: 2, Value: "grass"}
	plantType := core.Type{
		Name:    "Plant",
		Atomics: bitset.NewIntSetFromInts([]int{grass.Index}),
	}
	herbivoreType := core.Type{
		Name:    "Herbivore",
		Atomics: bitset.NewIntSetFromInts([]int{cow.Index}),
	}
	// true fact: cow eats grass
	eatDef := &core.PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "Eater", Type: &herbivoreType},
			{Label: "Food", Type: &plantType},
		},
	}
	trueFact := &core.Predicate{
		Definition: eatDef,
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: &cow},
			{Label: "Y", Ref: &grass},
		},
	}
	// false fact: cow does not eat cow
	falseFact := &core.Predicate{
		Definition: eatDef,
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: &cow},
			{Label: "Y", Ref: &cow},
		},
	}
	state := &State{
		TrueFacts: map[string][]*core.Predicate{
			"eat": {trueFact},
		},
		FalseFacts: map[string][]*core.Predicate{
			"eat": {falseFact},
		},
	}
	//find what cows eat
	query := &core.Predicate{
		Definition: eatDef,
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
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
