package core

import (
	"fmt"
	"strconv"
	"strings"
)

var Terminate = &Predicate{Definition: &PredicateDefinition{Functor: "Terminate"}}
var Pass = &Predicate{Definition: &PredicateDefinition{Functor: "Pass"}}

type ArgumentDefinition struct {
	Label         string
	Type          *Type
	SubDefinition *PredicateDefinition
}

type PredicateDefinition struct {
	Functor        string
	ArgDefinitions []ArgumentDefinition
}

type Unifiable interface {
	CanUnify(other Unifiable) bool
	Unify(other Unifiable) error
	Clone() Unifiable
}

type VariableReference struct {
	Label string    // names like &1 &2
	Ref   Unifiable // *Atomic, *Predicate, *VariableReference, or nil
}

type Predicate struct {
	Definition *PredicateDefinition
	VarRefs    []*VariableReference
}

type Frame struct {
	Vars   map[string]*VariableReference
	nextID int
}

type QueryContext interface {
	Done() <-chan struct{}
	Cancel()
}

type qContext struct {
	done      chan struct{}
	cancelled bool
}

type Answerer interface {
	Answer(p *Predicate, frame *Frame, ctx QueryContext, history *SearchHistory) <-chan *Predicate
	GetName() string
}

type SearchHistory struct {
	queryString string
	depth       int
	prev        *SearchHistory
}

func NewFrame() *Frame {
	return &Frame{
		Vars:   make(map[string]*VariableReference),
		nextID: 0,
	}
}

// Clone a frame, including all predicates and atomics bound to variables
func (frame *Frame) Clone() *Frame {
	if frame == nil {
		return NewFrame()
	}
	newSharedVars := make(map[string]*VariableReference)
	for label, varRef := range frame.Vars {
		// dereference everything
		vr := varRef.Dereference()
		if _, ok := newSharedVars[label]; ok {
			continue
		}
		if vr.Ref != nil {
			if p, ok := vr.Ref.(*Predicate); ok {
				newSharedVars[label] = &VariableReference{
					Label: label,
					Ref:   p.CloneInFrame(frame),
				}
			} else {
				// clone the variable bound to a non-predicate
				newSharedVars[label] = vr.Clone().(*VariableReference)
			}
		} else {
			// clone the unbound variable
			newSharedVars[label] = vr.Clone().(*VariableReference)
			if vr != varRef {
				newSharedVars[varRef.Label] = newSharedVars[label]
			}
		}
		if vr.Label != label {
			newSharedVars[vr.Label] = newSharedVars[label]
		}
	}
	return &Frame{
		Vars:   newSharedVars,
		nextID: frame.nextID,
	}
}

func NewQueryContext() QueryContext {
	return &qContext{
		done:      make(chan struct{}),
		cancelled: false,
	}
}

func (ctx *qContext) Done() <-chan struct{} {
	return ctx.done
}

func (ctx *qContext) Cancel() {
	if !ctx.cancelled {
		ctx.cancelled = true
		close(ctx.done)
	}
}

// Create a new Predicate, constructing VariableReferences and updating the frame as required
func NewPredicate(definition *PredicateDefinition, labels []string, args []Unifiable, frame *Frame) *Predicate {
	p := &Predicate{
		Definition: definition,
		VarRefs:    make([]*VariableReference, len(labels)),
	}
	for i, label := range labels {
		if varRef, ok := args[i].(*VariableReference); ok {
			varRef = varRef.Dereference()
			// if the variable reference is already in the frame, use it
			if vr, ok := frame.Vars[varRef.Label]; ok {
				p.VarRefs[i] = vr
				continue
			} else if varRef.Ref == nil {
				// if it is truly variable, store using its label
				//TODO: handle underscore "ignore" variables
				frame.Vars[varRef.Label] = varRef
				p.VarRefs[i] = frame.Vars[varRef.Label]
				continue
			}
		}
		// an atomic or predicate, possibly pointed to by a variable reference
		// ensure a new unique label is used
		frame.nextID++
		if atom, ok := args[i].(*Atomic); ok {
			label = atom.Value
		} else {
			label = "&" + strconv.Itoa(frame.nextID)
		}
		frame.Vars[label] = &VariableReference{
			Label: label,
			Ref:   args[i],
		}
		p.VarRefs[i] = frame.Vars[label]
	}
	return p
}

func (p *Predicate) String() string {
	s := fmt.Sprintf("%s(", p.Definition.Functor)
	for i, varRef := range p.VarRefs {
		s += varRef.String()
		if i < len(p.VarRefs)-1 {
			s += ", "
		}
	}
	s += ")"
	return s
}

func (p *Predicate) StringVerbose() string {
	sb := strings.Builder{}
	sb.WriteString(p.Definition.Functor)
	sb.WriteString("(")
	for i, varRef := range p.VarRefs {
		sb.WriteString(varRef.StringVerbose())
		if i < len(p.VarRefs)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(")")
	return sb.String()
}

func (p *Predicate) CanonicalArgsString(varCount int) string {
	sb := strings.Builder{}
	first := true
	for _, varRef := range p.VarRefs {
		if first {
			first = false
		} else {
			sb.WriteString(",")
		}
		vr := varRef.Dereference()
		if vr.Ref == nil {
			sb.WriteString("_")
			sb.WriteString(strconv.Itoa(varCount))
			varCount++
		} else if at, ok := vr.Ref.(*Atomic); ok {
			if len(at.Value) < 3 {
				sb.WriteString(at.Value)
			} else {
				sb.WriteString("_")
				sb.WriteString(strconv.Itoa(int(at.Index)))
			}
		} else if p, ok := vr.Ref.(*Predicate); ok {
			sb.WriteString(p.Definition.Functor)
			sb.WriteString("(")
			sb.WriteString(p.CanonicalArgsString(varCount))
			sb.WriteString(")")
		}
	}
	return sb.String()
}

func (p *Predicate) IsFact() bool {
	for _, varRef := range p.VarRefs {
		vr := varRef.Dereference()
		if vr.Ref == nil {
			return false
		} else if p2, ok := vr.Ref.(*Predicate); ok {
			if !p2.IsFact() {
				return false
			}
		}
	}
	return true
}

func (p *Predicate) GetArgument(index int) Unifiable {
	return p.VarRefs[index].Dereference().Ref
}

func (r *VariableReference) String() string {
	if r.Ref == nil {
		return r.Label
	}
	if a, ok := r.Ref.(*Atomic); ok {
		return a.String()
	}
	if v, ok := r.Ref.(*VariableReference); ok {
		return v.String()
	}
	if p, ok := r.Ref.(*Predicate); ok {
		return p.String()
	}
	return "BadRef"
}

func (r *VariableReference) StringVerbose() string {
	if v, ok := r.Ref.(*VariableReference); ok {
		return r.Label + " -> " + v.StringVerbose()
	}
	return "[" + r.Label + "]" + r.String()
}

func (r *VariableReference) Dereference() *VariableReference {
	for ref, ok := r.Ref.(*VariableReference); ok; ref, ok = r.Ref.(*VariableReference) {
		r = ref
	}
	return r
}

func (r *VariableReference) Unify(other Unifiable) error {
	// check for final referencing two non-variables. Call their Unify.
	finalR := r.Dereference()
	finalOtherRef := other
	if vRefOther, ok := other.(*VariableReference); ok {
		finalOther := vRefOther.Dereference()
		finalOtherRef = finalOther.Ref
		// check for same variable reference
		if finalR == finalOther {
			return nil
		}
		// if this is a variable too, then just point to the other
		if finalR.Ref == nil {
			finalR.Ref = finalOther
			return nil
		} else if finalOtherRef == nil {
			return finalOther.Unify(finalR.Ref)
		}
	}
	// we now have sorted the cases for
	// - both were the same variable
	// - both were different variables
	// - the other was a variable reference to something concrete

	// now if this is variable but other is directly concrete
	if finalR.Ref == nil {
		finalR.Ref = finalOtherRef
		return nil
	}

	// and finally if both are non-variables, unify them
	return finalR.Ref.Unify(finalOtherRef)
}

func (r *VariableReference) CanUnify(other Unifiable) bool {
	vr := r.Dereference()
	if vr.Ref == nil {
		return true
	}
	if bother, ok := other.(*VariableReference); ok {
		bother = bother.Dereference()
		if bother.Ref == nil {
			return true
		}
		return vr.Ref.CanUnify(bother.Ref)
	}
	return vr.Ref.CanUnify(other)
}

func (r *VariableReference) Clone() Unifiable {
	if r.Ref == nil {
		return &VariableReference{
			Label: r.Label,
			Ref:   nil,
		}
	}
	return &VariableReference{
		Label: r.Label,
		Ref:   r.Ref.Clone(),
	}
}

func (a *Predicate) Unify(other Unifiable) error {
	b, ok := other.(*Predicate)
	if !ok || a.Definition.Functor != b.Definition.Functor {
		return fmt.Errorf("functors do not match")
	}
	for i, argA := range a.VarRefs {
		argB := b.VarRefs[i]
		if err := argA.Unify(argB); err != nil {
			return err
		}
	}
	return nil
}

/*
* Unification of two Predicates. This requires matching functors and
* arguments that unify with one another.
 */
func (a *Predicate) CanUnify(other Unifiable) bool {
	b, ok := other.(*Predicate)
	if !ok {
		return false
	}
	if a.Definition.Functor != b.Definition.Functor {
		return false
	}

	// now unify each argument
	for i, argA := range a.VarRefs {
		argB := b.VarRefs[i]
		//TODO: what if two arguments are the same variable?
		// need to check that both other arguments can unify
		// and vice-versa
		if !argA.CanUnify(argB) {
			return false
		}
		if argA.Dereference().Ref == nil && argB.Dereference().Ref != nil {
			// now need to check if we have another unified argument
			for j := i + 1; j < len(a.VarRefs); j++ {
				if a.VarRefs[j].Label == argA.Label {
					// ensure the other's arguments can unify
					if !argB.CanUnify(b.VarRefs[j]) {
						return false
					}
				}
			}
		} else if argA.Dereference().Ref != nil && argB.Dereference().Ref == nil {
			// now need to check if we have another unified argument
			for j := i + 1; j < len(a.VarRefs); j++ {
				if b.VarRefs[j].Label == argB.Label {
					// ensure this can unify with the other argument too
					if !argA.CanUnify(b.VarRefs[j]) {
						return false
					}
				}
			}
		}
	}
	return true
}

func (a *Predicate) CloneInFrame(frame *Frame) *Predicate {
	p := &Predicate{
		Definition: a.Definition,
		VarRefs:    make([]*VariableReference, len(a.VarRefs)),
	}
	for i, varRef := range a.VarRefs {
		// handle recursive predicates
		d := varRef.Dereference()
		if p2, ok := d.Ref.(*Predicate); ok {
			// find the variable references for this predicate too
			p2 = p2.CloneInFrame(frame)
			// and make a new reference to it, not shared.
			p.VarRefs[i] = &VariableReference{
				Label: varRef.Label,
				Ref:   p2,
			}
			continue
		}
		if d.Ref != nil {
			// not a predicate. So probably an atomic.
			p.VarRefs[i] = &VariableReference{
				Label: varRef.Label,
				Ref:   d.Ref.Clone(),
			}
		} else if vr, ok := frame.Vars[varRef.Label]; ok {
			p.VarRefs[i] = vr
		} else {
			fmt.Println(frame.Vars, "missing", varRef.Label, "at index", i)
			panic("variable reference not found in frame " + varRef.Label + " " + varRef.String())
		}
	}
	return p
}

func (a *Predicate) Clone() Unifiable {
	p := &Predicate{
		Definition: a.Definition,
		VarRefs:    make([]*VariableReference, len(a.VarRefs)),
	}
	newVars := make(map[string]*VariableReference)
	needPredicates := make([]*VariableReference, 0, len(a.VarRefs))
	origPredicates := make([]*Predicate, 0, len(a.VarRefs))
	// clone everything except predicates
	for i, varRef := range a.VarRefs {
		v := varRef.Dereference()
		if _, ok := newVars[v.Label]; !ok {
			if op, pok := v.Ref.(*Predicate); pok {
				vr := &VariableReference{
					Label: v.Label,
					Ref:   nil,
				}
				newVars[v.Label] = vr
				needPredicates = append(needPredicates, vr)
				origPredicates = append(origPredicates, op)
			} else {
				vr := &VariableReference{
					Label: v.Label,
					Ref:   nil,
				}
				if v.Ref != nil {
					vr.Ref = v.Ref.Clone()
				}
				newVars[v.Label] = vr
			}
		}
		p.VarRefs[i] = newVars[v.Label]
	}
	frame := &Frame{
		Vars:   newVars,
		nextID: 0,
	}
	// clone the predicates
	for i, vr := range needPredicates {
		vr.Ref = origPredicates[i].CloneInFrame(frame)
	}
	return p
}

func AnswerConjunction(answerer Answerer, queries []*Predicate, frame *Frame, ctx QueryContext, history *SearchHistory) <-chan []*Predicate {
	// need to scan for any not() predicates. Then if there are any variables that are not present
	addTypeQueries := make(map[int]*Predicate)
	seenArgs := make(map[string]bool)
	for i, query := range queries {
		if query.Definition.Functor == "not" {
			notQuery := query.GetArgument(0).(*Predicate)
			for j, varRef := range notQuery.VarRefs {
				vr := varRef.Dereference()
				if vr.Ref == nil && !seenArgs[vr.Label] {
					// possibly need to add a dummy type query for this. Add just before this query
					seenArgs[vr.Label] = true
					queryType := notQuery.Definition.ArgDefinitions[j].Type
					def := &PredicateDefinition{Functor: queryType.Name, ArgDefinitions: []ArgumentDefinition{{Type: queryType}}}
					addTypeQueries[i] = NewPredicate(def, []string{vr.Label}, []Unifiable{&VariableReference{Label: vr.Label, Ref: nil}}, frame)
					continue
				}
			}
		} else {
			// add variable names to seen args
			for _, varRef := range query.VarRefs {
				vr := varRef.Dereference()
				if vr.Ref == nil {
					seenArgs[vr.Label] = true
				}
			}
		}
	}
	// now update query to add any required type predicates
	if len(addTypeQueries) > 0 {
		newQueries := make([]*Predicate, 0, len(queries)+len(addTypeQueries))
		for i, query := range queries {
			if typeQuery, ok := addTypeQueries[i]; ok {
				newQueries = append(newQueries, typeQuery)
			}
			newQueries = append(newQueries, query)
		}
		queries = newQueries
	}
	answers := make(chan []*Predicate, 2)
	go func() {
		stack := make([]<-chan *Predicate, 0, len(queries))
		frameStack := make([]*Frame, 0, len(queries))
		partialAnswers := make([][]*Predicate, 0, len(queries))
		fr := frame.Clone()
		frameStack = append(frameStack, fr)
		partialAnswers = append(partialAnswers, []*Predicate{queries[0].CloneInFrame(fr)})
		stack = append(stack, answerer.Answer(partialAnswers[0][0], frameStack[0], ctx, history))
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// continue
			}
			ans := <-stack[len(stack)-1]
			if ans == Terminate || ans == nil {
				// no more answers
				stack = stack[:len(stack)-1]
				partialAnswers = partialAnswers[:len(partialAnswers)-1]
				frameStack = frameStack[:len(frameStack)-1]
				if len(stack) == 0 {
					// we're done
					close(answers)
					return
				}
				// backtrack to the previous query
				continue
			}
			// so now we need to unify this answer with the current query, and continue
			// NOTE: if the next query is a fact already, we can skip over the cloning bit
			// as there is no need to unify the arguments

			if !partialAnswers[len(stack)-1][len(stack)-1].CanUnify(ans) {
				continue
			}
			nextFrame := frameStack[len(stack)-1].Clone()
			nextPartialAnswer := make([]*Predicate, len(stack)+1)
			for i, p := range partialAnswers[len(stack)-1] {
				nextPartialAnswer[i] = p.CloneInFrame(nextFrame)
			}
			// unify with the latest result and store
			nextPartialAnswer[len(stack)-1].Unify(ans)
			if len(stack) == len(queries) {
				answers <- nextPartialAnswer[:len(stack)]
			} else {
				// recurse
				// add the last query to the partial answer
				nextPartialAnswer[len(stack)] = queries[len(stack)].CloneInFrame(nextFrame)
				partialAnswers = append(partialAnswers, nextPartialAnswer)
				frameStack = append(frameStack, nextFrame)
				stack = append(stack, answerer.Answer(nextPartialAnswer[len(stack)], nextFrame, ctx, history))
			}
		}
	}()
	return answers
}

func (def *PredicateDefinition) Equals(other *PredicateDefinition) bool {
	if def == other {
		return true
	}
	if def == nil || other == nil {
		return false // only one is nil, as they aren't equal
	}
	if def.Functor != other.Functor || len(def.ArgDefinitions) != len(other.ArgDefinitions) {
		return false
	}
	for i, d := range def.ArgDefinitions {
		odd := other.ArgDefinitions[i]
		if !d.Equals(&odd) {
			return false
		}
	}
	return true
}

func (def *ArgumentDefinition) Equals(other *ArgumentDefinition) bool {
	if def.Type != nil && other.Type != nil && def.Type != other.Type {
		return false
	}
	return def.SubDefinition.Equals(other.SubDefinition)
}

func (def *ArgumentDefinition) String() string {
	if def.Type != nil {
		return def.Type.Name
	}
	return def.Label + ":" + def.SubDefinition.Functor
}

func (def *PredicateDefinition) String() string {
	sb := strings.Builder{}
	sb.WriteString(def.Functor)
	sb.WriteString("(")
	for i, argDef := range def.ArgDefinitions {
		sb.WriteString(argDef.String())
		if i < len(def.ArgDefinitions)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(")")
	return sb.String()
}

func NewSearchHistory() *SearchHistory {
	return &SearchHistory{
		queryString: "",
		depth:       0,
		prev:        nil,
	}
}

func historyString(p *Predicate) string {
	sb := strings.Builder{}
	sb.WriteString(p.Definition.Functor)
	sb.WriteString("(")
	sb.WriteString(p.CanonicalArgsString(0))
	sb.WriteString(")")
	return sb.String()
}

func (sh *SearchHistory) AddHistory(p *Predicate) *SearchHistory {
	s := historyString(p)
	newHistory := &SearchHistory{
		queryString: s,
		depth:       sh.depth + 1,
		prev:        sh,
	}
	return newHistory
}

func (sh *SearchHistory) Contains(p *Predicate) bool {
	s := historyString(p)
	for h := sh; h != nil; h = h.prev {
		if h.queryString == s {
			return true
		}
	}
	return false
}

func (sh *SearchHistory) String() string {
	sb := strings.Builder{}
	for h := sh; h != nil; h = h.prev {
		sb.WriteString(" ")
		sb.WriteString(strconv.Itoa(h.depth))
		sb.WriteString(": ")
		sb.WriteString(h.queryString)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (sh *SearchHistory) Depth() int {
	return sh.depth
}
