package knowledgebase

import (
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type KnowledgeBase struct {
	core.Answerer
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

func (kb *KnowledgeBase) AddAtomic(name string, t *core.Type) {
	kb.state.GetAtomic(name, t)
}

func (kb *KnowledgeBase) SetTrue(p *core.Predicate) {
	kb.state.SetTrue(p)
}

func (kb *KnowledgeBase) Answer(p *core.Predicate) <-chan []*core.Atomic {
	answer := make(chan []*core.Atomic)
	go func() {
		for _, answerer := range kb.answerers {
			for ans := range answerer.Answer(p) {
				answer <- ans
			}
		}
		answer <- nil
		close(answer)
	}()
	return answer
}
