package cli

import (
	"github.com/Mario-pereyra/mapj/internal/auth"
)

// Note: auth commands are added via auth.go init() side effect for modularity.
// The auth package defines its own commands and wires them to rootCmd via AddCommands.
// This keeps auth-related code self-contained in the auth package.
func init() {
	auth.AddCommands(rootCmd)
}
