package cmd

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-docs/internal/provider"
)

type generateCmd struct {
	commonCmd

	flagLegacySidebar    bool
	flagIgnoreDeprecated bool

	flagProviderName         string
	flagRenderedProviderName string

	flagRenderedWebsiteDir string
	flagExamplesDir        string
	flagWebsiteTmpDir      string
	flagWebsiteSourceDir   string
	tfVersion              string
}

func (cmd *generateCmd) Synopsis() string {
	return "generates a plugin website from code, templates, and examples for the current directory"
}

func (cmd *generateCmd) Help() string {
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

	strBuilder.WriteString(fmt.Sprintf("\nUsage: tfplugindocs generate [<args>]\n\n"))
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

func (cmd *generateCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	fs.BoolVar(&cmd.flagLegacySidebar, "legacy-sidebar", false, "generate the legacy .erb sidebar file")
	fs.StringVar(&cmd.flagProviderName, "provider-name", "", "provider name, as used in Terraform configurations")
	fs.StringVar(&cmd.flagRenderedProviderName, "rendered-provider-name", "", "provider name, as generated in documentation (ex. page titles, ...)")
	fs.StringVar(&cmd.flagRenderedWebsiteDir, "rendered-website-dir", "docs", "output directory")
	fs.StringVar(&cmd.flagExamplesDir, "examples-dir", "examples", "examples directory")
	fs.StringVar(&cmd.flagWebsiteTmpDir, "website-temp-dir", "", "temporary directory (used during generation)")
	fs.StringVar(&cmd.flagWebsiteSourceDir, "website-source-dir", "templates", "templates directory")
	fs.StringVar(&cmd.tfVersion, "tf-version", "", "terraform binary version to download")
	fs.BoolVar(&cmd.flagIgnoreDeprecated, "ignore-deprecated", false, "don't generate documentation for deprecated resources and data-sources")
	return fs
}

func (cmd *generateCmd) Run(args []string) int {
	fs := cmd.Flags()
	err := fs.Parse(args)
	if err != nil {
		cmd.ui.Error(fmt.Sprintf("unable to parse flags: %s", err))
		return 1
	}

	return cmd.run(cmd.runInternal)
}

func (cmd *generateCmd) runInternal() error {
	err := provider.Generate(
		cmd.ui,
		cmd.flagLegacySidebar,
		cmd.flagProviderName,
		cmd.flagRenderedProviderName,
		cmd.flagRenderedWebsiteDir,
		cmd.flagExamplesDir,
		cmd.flagWebsiteTmpDir,
		cmd.flagWebsiteSourceDir,
		cmd.tfVersion,
		cmd.flagIgnoreDeprecated,
	)
	if err != nil {
		return fmt.Errorf("unable to generate website: %w", err)
	}

	return nil
}
