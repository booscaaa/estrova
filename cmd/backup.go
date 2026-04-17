package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/booscaaa/estrova/internal/db"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup [arquivo]",
	Short: "Faz backup do banco de dados para um arquivo",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		dst := fmt.Sprintf("estrova-backup-%s.db", time.Now().Format("2006-01-02"))
		if len(args) == 1 {
			dst = args[0]
		}

		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("arquivo %q já existe, escolha outro nome", dst)
		}

		d, err := db.Open()
		if err != nil {
			return fmt.Errorf("não foi possível abrir o banco: %w", err)
		}
		defer d.Close()

		if err := d.Backup(dst); err != nil {
			return fmt.Errorf("backup falhou: %w", err)
		}

		info, _ := os.Stat(dst)
		fmt.Printf("Backup salvo em: %s (%d KB)\n", dst, info.Size()/1024)
		return nil
	},
}
