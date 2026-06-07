package knowledgebase

import (
	"context"

	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

var Not = &core.PredicateDefinition{
	Functor: "not",
	ArgDefinitions: []core.ArgumentDefinition{
		{
			Label: "P",
			Type:  nil,
		},
	},
}

var Eq = &core.PredicateDefinition{
	Functor: "eq",
	ArgDefinitions: []core.ArgumentDefinition{
		{
			Label: "A",
			Type:  nil,
		},
		{
			Label: "B",
			Type:  nil,
		},
	},
}

type KnowledgeBase struct {
	core.Answerer
	predicateDefinitions map[string]*core.PredicateDefinition
	State                *state.State

	answerers []core.Answerer
}

func NewKnowledgeBase() *KnowledgeBase {
	kb := &KnowledgeBase{
		predicateDefinitions: make(map[string]*core.PredicateDefinition),
		State:                state.NewState(),
		//TODO: each predicate definition should have its own ordering of answerers
		answerers: make([]core.Answerer, 0, 10),
	}
	kb.answerers = append(kb.answerers, kb.State)
	kb.AddPredicateDefinition(Not)
	kb.AddPredicateDefinition(Eq)
	return kb
}

func (kb *KnowledgeBase) AddPredicateDefinition(pdef *core.PredicateDefinition) {
	kb.predicateDefinitions[pdef.Functor] = pdef
}

func (kb *KnowledgeBase) AddAtomic(name string, t *core.Type) {
	kb.State.GetAtomic(name, t)
}

func (kb *KnowledgeBase) AddAnswerer(answerer core.Answerer) {
	kb.answerers = append(kb.answerers, answerer)
}

func (kb *KnowledgeBase) SetTrue(p *core.Predicate) {
	kb.State.SetTrue(p)
}

func (kb *KnowledgeBase) Exists(p *core.Predicate) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	answer := kb.Answer(p, core.NewFrame(), ctx)
	ans := <-answer
	if ans == nil || ans == core.Terminate {
		return false
	}
	return true
}

func (kb *KnowledgeBase) Answer(p *core.Predicate, frame *core.Frame, ctx context.Context) <-chan *core.Predicate {
	answers := make(chan *core.Predicate)
	go func() {
		if p.Definition == Not {
			subP := (p.VarRefs[0].Dereference().Ref).(*core.Predicate)
			if kb.Exists(subP) {
				answers <- core.Terminate
				close(answers)
				return
			} else {
				answers <- p
				close(answers)
				return
			}
		}
		if p.Definition == Eq {
			a := p.VarRefs[0]
			b := p.VarRefs[1]
			if a.CanUnify(b) {
				cp := p.CloneInFrame(frame)
				cp.VarRefs[0].Unify(cp.VarRefs[1])
				answers <- cp
				// nothing else should do stuff with Eq predicates
				answers <- core.Terminate
				close(answers)
				return
			} else {
				answers <- core.Terminate
				close(answers)
				return
			}
		}
	loopAnswerers:
		for _, answerer := range kb.answerers {
			// TODO: is a new context needed here?
			subAnswer := answerer.Answer(p, frame, ctx)
			for {
				select {
				case <-ctx.Done():
					close(answers)
					return
				case ans := <-subAnswer:
					if ans == nil {
						// end of answers for this answerer
						continue loopAnswerers
					}
					answers <- ans
					if ans == core.Terminate {
						close(answers)
						return
					}
				}
			}
		}
		close(answers)
	}()
	return answers
}
