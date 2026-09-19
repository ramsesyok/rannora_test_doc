package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd builds a fresh command tree. This keeps flag state isolated and
// leaves room for future validate and inspect subcommands.
func NewRootCmd(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "runnora-instructions",
		Short:         "runnora シナリオからレビュー用テスト手順書を生成する",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(newGenerateCmd(stdout, stderr))
	return root
}

func Execute() error {
	return NewRootCmd(os.Stdout, os.Stderr).Execute()
}
