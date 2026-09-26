package knowledgebase

import (
	"github.com/jteutenberg/understate/core"
)

type SearchContext struct {
	subContext core.QueryContext
}

func NewSearchContext(subContext core.QueryContext) *SearchContext {
	ctx := &SearchContext{
		subContext: subContext,
	}
	return ctx
}

func (ctx *SearchContext) Done() <-chan struct{} {
	return ctx.subContext.Done()
}

func (ctx *SearchContext) Cancel() {
	ctx.subContext.Cancel()
}
