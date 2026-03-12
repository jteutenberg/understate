package knowledgebase

import (
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type KnowledgeBase struct {
	predicateDefinitions map[string]*core.PredicateDefinition
	state                *state.State

	answerers []core.Answerer
}

func NewKnowledgeBase() *KnowledgeBase {
	kb := &KnowledgeBase{
		predicateDefinitions: make(map[string]*core.PredicateDefinition),
		state:                state.NewState(),
		//TODO: each predicate definition should have its own ordering of answerers
		answerers: make([]core.Answerer, 0, 10),
	}
	kb.answerers = append(kb.answerers, kb.state)
	return kb
}

func (kb *KnowledgeBase) AddPredicateDefinition(pdef *core.PredicateDefinition) {
	kb.predicateDefinitions[pdef.Functor] = pdef
}
