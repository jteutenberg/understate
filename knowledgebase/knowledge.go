package knowledgebase

import (
	"strconv"
	"strings"

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
	ctx := core.NewQueryContext()
	defer ctx.Cancel()
	answer := kb.Answer(p, core.NewFrame(), ctx)
	ans := <-answer
	if ans == nil || ans == core.Terminate {
		return false
	}
	return true
}

func (kb *KnowledgeBase) argsKey(p *core.Predicate, mask []bool) string {
	var sb strings.Builder
	for i, arg := range p.VarRefs {
		if mask[i] {
			continue
		}
		a := arg.Dereference().Ref.(*core.Atomic)
		sb.WriteString(strconv.Itoa(int(a.Index)))
		sb.WriteString(",")
	}
	return sb.String()
}

func (kb *KnowledgeBase) GetName() string {
	return "KnowledgeBase"
}

func (kb *KnowledgeBase) Answer(p *core.Predicate, frame *core.Frame, ctx core.QueryContext) <-chan *core.Predicate {
	answers := make(chan *core.Predicate, 1)
	// ensure we are using a SearchContext
	var searchCtx *SearchContext
	if sCtx, ok := ctx.(*SearchContext); ok {
		searchCtx = sCtx
	} else {
		searchCtx = NewSearchContext(ctx)
	}
	if searchCtx.depth > 100 {
		close(answers)
		return answers
	}

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
		sent := map[string]bool{}
		searchCtx.depth++
		mask := make([]bool, len(p.VarRefs))
		for i := range mask {
			// ignore variables labelled with leading underscore
			mask[i] = p.VarRefs[i].Label[0] == '_'
		}
	loopAnswerers:
		for _, answerer := range kb.answerers {
			subAnswer := answerer.Answer(p, frame, searchCtx)

			for {
				select {
				case <-searchCtx.Done():
					searchCtx.depth--
					close(answers)
					return
				case ans := <-subAnswer:
					if ans == nil {
						// end of answers for this answerer
						continue loopAnswerers
					}
					argsKey := kb.argsKey(ans, mask)
					if sent[argsKey] {
						continue
					}
					sent[argsKey] = true
					answers <- ans
					if ans == core.Terminate {
						searchCtx.depth--
						close(answers)
						return
					}
				}
			}
		}
		searchCtx.depth--
		close(answers)
	}()
	return answers
}
