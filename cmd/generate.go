package cmd

import (
	"fmt"
	"io"

	"github.com/ramsesyok/runnora-docgen/internal/generator"
	"github.com/spf13/cobra"
)

func newGenerateCmd(stdout, stderr io.Writer) *cobra.Command {
	var opts generator.Options
	cmd := &cobra.Command{
		Use:   "generate <runbook.yml>...",
		Short: "runbook を分割された Quarto 原稿へ変換する",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RunbookPaths = args
			result, err := generator.Generate(cmd.Context(), opts)
			if err != nil {
				return err
			}
			for _, path := range result.Files {
				fmt.Fprintln(stdout, path)
			}
			for _, warning := range result.Warnings {
				fmt.Fprintf(stderr, "warning: %s\n", warning)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&opts.OutputDir, "out", "o", "generated", "生成した .qmd を置くディレクトリ")
	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "", "runnora 設定 YAML")
	cmd.Flags().StringSliceVar(&opts.BeforeSQL, "before-sql", nil, "追加の前処理 SQL（複数指定可）")
	cmd.Flags().StringSliceVar(&opts.AfterSQL, "after-sql", nil, "追加の後処理 SQL（複数指定可）")
	cmd.Flags().StringSliceVar(&opts.ProtoPaths, "proto", nil, "RPC 種別判定に使う .proto（複数指定可）")
	cmd.Flags().StringVar(&opts.BaseDir, "base-dir", "", "config と追加 SQL の相対パスの基準（既定: 現在のディレクトリ）")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "既存の生成原稿を上書きする")
	return cmd
}
