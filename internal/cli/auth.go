package cli

import (
	"github.com/Mario-pereyra/mapj/internal/auth"
)

func init() {
	auth.AddCommands(rootCmd)
}
