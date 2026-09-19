package knowledgebase

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

// Holds when its argument does not unify with any true facts.
// Answers are always the argument itself.
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
	//TODO: check for existing definition. If they differ, report a conflict.
	if existing, ok := kb.predicateDefinitions[pdef.Functor]; ok {
		if !existing.Equals(pdef) {
			fmt.Println("Conflict: ", existing.Functor, " already defined as ", existing.String(), " but now defined as ", pdef.String())
			return
		}
	}
	kb.predicateDefinitions[pdef.Functor] = pdef
	// add any new Types, and create special type predicates for them
	for _, argDef := range pdef.ArgDefinitions {
		if argDef.Type != nil && kb.predicateDefinitions[argDef.Type.Name] == nil {
			typePredicate := &core.PredicateDefinition{
				Functor: argDef.Type.Name,
				ArgDefinitions: []core.ArgumentDefinition{
					{
						Label: "X",
						Type:  nil,
					},
				},
			}
			kb.predicateDefinitions[argDef.Type.Name] = typePredicate
		}
	}
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

func (kb *KnowledgeBase) Exists(p *core.Predicate, ctx core.QueryContext) bool {
	answer := kb.Answer(p, core.NewFrame(), ctx)
	ans := <-answer
	if ans == nil || ans == core.Terminate {
		return false
	}
	return true
}

func (kb *KnowledgeBase) argsKey(p *core.Predicate, mask []bool) (string, bool) {
	var sb strings.Builder
	for i, arg := range p.VarRefs {
		if mask[i] {
			continue
		}
		if a, ok := arg.Dereference().Ref.(*core.Atomic); ok {
			sb.WriteString(strconv.Itoa(int(a.Index)))
			sb.WriteString(",")
		} else {
			return "", true
		}
	}
	return sb.String(), false
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
	//fmt.Println("At depth", searchCtx.depth, "checking history for", p.String())
	//for b, h := range searchCtx.history {
	//	fmt.Println(" hist", b, h)
	//}
	if searchCtx.InHistory(p) {
		close(answers)
		return answers
	}
	if searchCtx.depth > 100 {
		close(answers)
		return answers
	}

	go func() {
		if p.Definition == Not {
			subP := (p.VarRefs[0].Dereference().Ref).(*core.Predicate)
			if kb.Exists(subP, searchCtx) {
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
		searchCtx.AddHistory(p)
		searchCtx.depth++
		mask := make([]bool, len(p.VarRefs))
		for i := range mask {
			// ignore variables labelled with leading underscore
			mask[i] = p.VarRefs[i].Label[0] == '_'
		}
		//fmt.Println("Increased context depth to", searchCtx.depth, len(searchCtx.history))
	loopAnswerers:
		for _, answerer := range kb.answerers {
			subAnswer := answerer.Answer(p, frame, searchCtx)

			for {
				select {
				case <-searchCtx.Done():
					goto finished
				case ans := <-subAnswer:
					if ans == nil {
						// end of answers for this answerer
						continue loopAnswerers
					}
					argsKey, pass := kb.argsKey(ans, mask)
					if !pass && sent[argsKey] {
						continue
					}
					sent[argsKey] = true
					answers <- ans
					if ans == core.Terminate {
						goto finished
					}
					if p.IsFact() {
						// only one possible answer: a match
						goto finished
					}
				}
			}
		}
	finished:
		searchCtx.depth--
		searchCtx.PopHistory()
		close(answers)
	}()
	return answers
}
