package cmd

import (
	"fmt"
	"os"

	"github.com/booscaaa/estrova/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Atualiza o binário e as skills do estrova",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := updater.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			return err
		}
		return nil
	},
}
