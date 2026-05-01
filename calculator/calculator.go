package calculator

import (
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type Calculator struct {
	core.Answerer
	state *state.State
}

var Gt = &core.PredicateDefinition{
	Functor: "gt",
	ArgDefinitions: []core.ArgumentDefinition{
		{Label: "A", Type: state.Numeric},
		{Label: "B", Type: state.Numeric},
	},
}

var Sum = &core.PredicateDefinition{
	Functor: "sum",
	ArgDefinitions: []core.ArgumentDefinition{
		{Label: "A", Type: state.Numeric},
		{Label: "B", Type: state.Numeric},
	},
}

func NewCalculator(state *state.State) *Calculator {
	return &Calculator{
		state: state,
	}
}

func (calc *Calculator) GetAtomicValue(p *core.Predicate, arg int) *core.Atomic {
	a := p.GetArgument(arg)
	if atomic, ok := a.(*core.Atomic); ok {
		return atomic
	}
	return nil
}

func (calc *Calculator) Answer(p *core.Predicate, halt <-chan bool) <-chan *core.Predicate {
	answer := make(chan *core.Predicate)
	go func() {
		switch p.Definition.Functor {
		case "sum":
			v1 := calc.GetAtomicValue(p, 0)
			v2 := calc.GetAtomicValue(p, 1)
			v3 := calc.GetAtomicValue(p, 2)
			// cases: v1 and v2 are interchangable: 0, 1, or 2 set (three cases)
			// x2 cases for v3 being set
			// = 6 cases
			if v1 != nil && v2 != nil {
				if v3 != nil {
					// check correctness
				} else {
					// perform sum
				}
			}
			answer <- core.Terminate
		case "gt":
			v1 := calc.GetAtomicValue(p, 0)
			v2 := calc.GetAtomicValue(p, 1)
			if v1 != nil {
				if v2 == nil {
					for i := v1.Index - 1; i >= 0; i-- {
						//(v1, i)
						answer <- &core.Predicate{
							Definition: p.Definition,
							VarRefs: []*core.VariableReference{
								{Label: p.VarRefs[0].Label, Ref: v1},
								{Label: p.VarRefs[1].Label, Ref: calc.state.GetNumericAtomic(i)},
							},
						}
						select {
						case <-halt:
							goto done
						default:
							// continue
						}
					}
				} else {
					// fact: send (v1,v2) if v1 > v2, and then terminate
					if v1.Index > v2.Index {
						answer <- p
					}
				}
			} else {
				if v2 == nil {
					for i := 1; true; i++ {
						// send (1,0), (2, 0), (2, 1) ...
						for j := 0; j < i; j++ {
							// answer (i, j)
							answer <- &core.Predicate{Definition: p.Definition, VarRefs: []*core.VariableReference{
								{Label: p.VarRefs[0].Label, Ref: calc.state.GetNumericAtomic(i)},
								{Label: p.VarRefs[1].Label, Ref: calc.state.GetNumericAtomic(j)},
							}}
							select {
							case <-halt:
								goto done
							default:
								// continue
							}
						}
					}

				} else {
					// send (v2+1,v2), (v2+2, v2)...
					for i := v2.Index + 1; true; i++ {
						// answer (i, v2)
						answer <- &core.Predicate{
							Definition: p.Definition,
							VarRefs: []*core.VariableReference{
								{Label: p.VarRefs[0].Label, Ref: calc.state.GetNumericAtomic(i)},
								{Label: p.VarRefs[1].Label, Ref: v2},
							},
						}
						select {
						case <-halt:
							goto done
						default:
							// continue
						}
					}
				}
			}
		done:
			answer <- core.Terminate
		}
		close(answer)
	}()
	return answer
}
