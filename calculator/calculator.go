package calculator

import "github.com/jteutenberg/understate/core"

type Calculator struct {
	core.Answerer
}

func (*Calculator) Answer(p *core.Predicate) <-chan []*core.Atomic {
	answer := make(chan []*core.Atomic)
	if p.Definition.Functor == "sum" {

	} else if p.Definition.Functor == "gt" {
		v1 := p.GetArgument(0)
		v2 := p.GetArgument(1)
		if v1 != nil {
			if v2 == nil {
				// TODO: someone should be storing numeric atomics for reuse when generating here
				// send (v1, v1-1), ...
			} else {
				// fact: send (v1,v2) if v1 > v2, and then send nil to terminate

			}
		} else {
			if v2 == nil {
				// send (1,0), (2, 0), (2, 1) ...

			} else {
				// send (v2+1,v2), (v2+2, v2)...
			}
		}
	} else {
		close(answer)
	}
	return answer
}
