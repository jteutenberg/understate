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
		{Label: "Total", Type: state.Numeric},
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

func (calc *Calculator) Answer(p *core.Predicate, frame *core.Frame, ctx core.QueryContext) <-chan *core.Predicate {
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
			if v1 == nil && v2 == nil && v3 == nil {
				close(answer)
				return
			}
			if v1 != nil && v2 != nil {
				if v3 != nil {
					// check correctness of the fact
					if v1.Index+v2.Index == v3.Index {
						answer <- p
					}
				} else {
					// perform sum
					v3 = calc.state.GetNumericAtomic(v1.Index + v2.Index)
				}
			} else if v1 == nil && v2 != nil {
				if v3 != nil {
					v1 = calc.state.GetNumericAtomic(v3.Index - v2.Index)
				} else {
					// only v2 is not nil, need to iterate over v1, v3 combinations
					for i := 0; ; i++ {
						v1 = calc.state.GetNumericAtomic(uint(i))
						v3 = calc.state.GetNumericAtomic(uint(i) + v2.Index)
						// ensure v1, v3 are not equal. Need a CanUnify check otherwise
						if uint(i) == uint(i)+v2.Index || p.VarRefs[0].Label != p.VarRefs[2].Label {
							answer <- &core.Predicate{
								Definition: p.Definition,
								VarRefs: []*core.VariableReference{
									{Label: p.VarRefs[0].Label, Ref: v1},
									{Label: p.VarRefs[1].Label, Ref: v2},
									{Label: p.VarRefs[2].Label, Ref: v3},
								},
							}
						}
						select {
						case <-ctx.Done():
							goto done
						default:
							// continue
						}
					}
				}
			} else if v2 == nil && v1 != nil {
				if v3 != nil {
					v2 = calc.state.GetNumericAtomic(v3.Index - v1.Index)
				} else {
					// need to iterate over v2, v3 combinations
					for i := 0; ; i++ {
						v2 = calc.state.GetNumericAtomic(uint(i))
						v3 = calc.state.GetNumericAtomic(uint(i) + v1.Index)
						// ensure v2, v3 are not equal. Need a CanUnify check otherwise
						if uint(i) == uint(i)+v1.Index || p.VarRefs[1].Label != p.VarRefs[2].Label {
							answer <- &core.Predicate{
								Definition: p.Definition,
								VarRefs: []*core.VariableReference{
									{Label: p.VarRefs[0].Label, Ref: v1},
									{Label: p.VarRefs[1].Label, Ref: v2},
									{Label: p.VarRefs[2].Label, Ref: v3},
								},
							}
						}
						select {
						case <-ctx.Done():
							goto done
						default:
							// continue
						}
					}
				}
			} else {
				// v3 must not be nil
				// need to iterate over v1, v2 combinations
				for i := uint(0); i <= v3.Index; i++ {
					v1 = calc.state.GetNumericAtomic(i)
					v2 = calc.state.GetNumericAtomic(v3.Index - i)
					// ensure v1, v2 are not equal. Need a CanUnify check otherwise
					if i == v3.Index-i || p.VarRefs[0].Label != p.VarRefs[1].Label {
						answer <- &core.Predicate{
							Definition: p.Definition,
							VarRefs: []*core.VariableReference{
								{Label: p.VarRefs[0].Label, Ref: v1},
								{Label: p.VarRefs[1].Label, Ref: v2},
								{Label: p.VarRefs[2].Label, Ref: v3},
							},
						}
					}
					select {
					case <-ctx.Done():
						goto done
					default:
						// continue
					}
				}
				answer <- core.Terminate
				close(answer)
				return
			}
			answer <- &core.Predicate{
				Definition: p.Definition,
				VarRefs: []*core.VariableReference{
					{Label: p.VarRefs[0].Label, Ref: v1},
					{Label: p.VarRefs[0].Label, Ref: v2},
					{Label: p.VarRefs[2].Label, Ref: v3},
				},
			}
			answer <- core.Terminate
		case "gt":
			v1 := calc.GetAtomicValue(p, 0)
			v2 := calc.GetAtomicValue(p, 1)
			if v1 != nil {
				if v2 == nil {
					for i := int(v1.Index) - 1; i >= 0; i-- {
						//(v1, i)
						answer <- &core.Predicate{
							Definition: p.Definition,
							VarRefs: []*core.VariableReference{
								{Label: p.VarRefs[0].Label, Ref: v1},
								{Label: p.VarRefs[1].Label, Ref: calc.state.GetNumericAtomic(uint(i))},
							},
						}
						select {
						case <-ctx.Done():
							goto done
						default:
							// continue
						}
					}
					answer <- core.Terminate
				} else {
					// fact: send (v1,v2) if v1 > v2, and then terminate
					if v1.Index > v2.Index {
						answer <- p
						answer <- core.Terminate
					}
				}
			} else {
				if v2 == nil {
					for i := uint(1); true; i++ {
						// send (1,0), (2, 0), (2, 1) ...
						for j := uint(0); j < i; j++ {
							// answer (i, j)
							answer <- &core.Predicate{Definition: p.Definition, VarRefs: []*core.VariableReference{
								{Label: p.VarRefs[0].Label, Ref: calc.state.GetNumericAtomic(i)},
								{Label: p.VarRefs[1].Label, Ref: calc.state.GetNumericAtomic(j)},
							}}
							select {
							case <-ctx.Done():
								goto done
							default:
								// continue
							}
						}
					}
					answer <- core.Terminate
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
						case <-ctx.Done():
							goto done
						default:
							// continue
						}
					}
					answer <- core.Terminate
				}
			}
		}
	done:
		close(answer)
	}()
	return answer
}
