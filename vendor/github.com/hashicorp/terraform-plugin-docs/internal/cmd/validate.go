package cmd

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-docs/internal/provider"
)

type validateCmd struct {
	commonCmd
}

func (cmd *validateCmd) Synopsis() string {
	return "validates a plugin website for the current directory"
}

func (cmd *validateCmd) Help() string {
	strBuilder := &strings.Builder{}

	longestName := 0
	longestUsage := 0
	cmd.Flags().VisitAll(func(f *flag.Flag) {
		if len(f.Name) > longestName {
			longestName = len(f.Name)
		}
		if len(f.Usage) > longestUsage {
			longestUsage = len(f.Usage)
		}
	})

	strBuilder.WriteString(fmt.Sprintf("\nUsage: tfplugindocs validate [<args>]\n\n"))
	cmd.Flags().VisitAll(func(f *flag.Flag) {
		if f.DefValue != "" {
			strBuilder.WriteString(fmt.Sprintf("    --%s <ARG> %s%s%s  (default: %q)\n",
				f.Name,
				strings.Repeat(" ", longestName-len(f.Name)+2),
				f.Usage,
				strings.Repeat(" ", longestUsage-len(f.Usage)+2),
				f.DefValue,
			))
		} else {
			strBuilder.WriteString(fmt.Sprintf("    --%s <ARG> %s%s%s\n",
				f.Name,
				strings.Repeat(" ", longestName-len(f.Name)+2),
				f.Usage,
				strings.Repeat(" ", longestUsage-len(f.Usage)+2),
			))
		}
	})
	strBuilder.WriteString("\n")

	return strBuilder.String()
}

func (cmd *validateCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	return fs
}

func (cmd *validateCmd) Run(args []string) int {
	fs := cmd.Flags()
	err := fs.Parse(args)
	if err != nil {
		cmd.ui.Error(fmt.Sprintf("unable to parse flags: %s", err))
		return 1
	}

	return cmd.run(cmd.runInternal)
}

func (cmd *validateCmd) runInternal() error {
	err := provider.Validate(cmd.ui)
	if err != nil {
		return fmt.Errorf("unable to validate website: %w", err)
	}

	return nil
}
