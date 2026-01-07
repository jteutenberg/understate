package core

import "fmt"

type Atomic struct {
	Index int
	Value string
	Type  *Type
}

// the marker for Termination
var Terminate = Atomic{Index: -1, Value: "Terminate"}

type AtomicSet map[Atomic]bool

type Type struct {
	Name    string
	Atomics AtomicSet
}

func (a *Atomic) String() string {
	if a.Value == "" {
		return fmt.Sprintf("%v", a.Index)
	}
	return a.Value
}

func (a *Atomic) CanUnify(b Unifiable) bool {
	if b, ok := b.(*Atomic); ok {
		return a.Type == b.Type && a.Index == b.Index
	}
	return false
}

func (a *Atomic) Unify(b Unifiable) error {
	if b, ok := b.(*Atomic); ok {
		if a.Type != b.Type || a.Index != b.Index {
			return fmt.Errorf("atomic %v does not unify with %v", a, b)
		}
		return nil
	}
	return fmt.Errorf("atomic %v does not unify with non-atomic %v", a, b)
}

func (a *Atomic) Clone() Unifiable {
	return a
}
