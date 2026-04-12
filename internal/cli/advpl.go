package cli

import (
	"github.com/spf13/cobra"
)

var advplCmd = &cobra.Command{
	Use:   "advpl",
	Short: "Compile, inspect RPO, manage patches, and monitor TOTVS Application Servers",
	Long: `Manage TOTVS AdvPL/TLPP development via the TDS Language Server (advpls).

TWO-STEP MODEL:
  1. Register an AppServer connection profile (one-time setup):
     mapj advpl connection add PRODUCAO --server 192.168.1.100 --port 5025 \
       --environment producao --user admin --password "" --use

  2. Compile, inspect RPO, manage patches:
     mapj advpl compile --files src/MATA110.PRW --includes /path/to/includes
     mapj advpl rpo inspect --filter "MATA*"
     mapj advpl patch generate --resources MATA110.PRW --type PTM --output ./patches/

SUBCOMMANDS:
  mapj advpl connection     Manage AppServer connection profiles
  mapj advpl compile         Compile AdvPL/TLPP sources
  mapj advpl rpo             Inspect and manage RPO (Repository of Objects)
  mapj advpl patch           Generate, validate, apply, and inspect patches
  mapj advpl monitor         Monitor connected users and server state

PREREQUISITES:
  The advpls binary must be installed. Options:
    npm install -g @totvs/tds-ls
    Download from https://github.com/totvs/tds-ls/releases
    Place in PATH or ~/.config/mapj/bin/

Run 'mapj advpl <command> --help' for full output schema.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(advplCmd)
}
