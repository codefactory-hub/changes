//go:build !devtools

package cli

import "context"

func (a *App) runOptionalCommand(_ context.Context, _ []string) (bool, error) {
	return false, nil
}
