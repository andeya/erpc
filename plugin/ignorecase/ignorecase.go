// Package ignorecase dynamically ignoring the case of path
package ignorecase

import (
	"strings"

	"github.com/henrylee2cn/erpc/v6"
)

// NewIgnoreCase Returns a ignoreCase plugin.
func NewIgnoreCase() *ignoreCase {
	return &ignoreCase{}
}

type ignoreCase struct{}

var (
	_ erpc.PostReadCallHeaderPlugin = new(ignoreCase)
	_ erpc.PostReadPushHeaderPlugin = new(ignoreCase)
)

func (i *ignoreCase) Name() string {
	return "ignoreCase"
}

func (i *ignoreCase) PostReadCallHeader(ctx erpc.ReadCtx) *erpc.Status {
	// Dynamic transformation path is lowercase
	ctx.ResetServiceMethod(strings.ToLower(ctx.ServiceMethod()))
	return nil
}

func (i *ignoreCase) PostReadPushHeader(ctx erpc.ReadCtx) *erpc.Status {
	// Dynamic transformation path is lowercase
	ctx.ResetServiceMethod(strings.ToLower(ctx.ServiceMethod()))
	return nil
}
