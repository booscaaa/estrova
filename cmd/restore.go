package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/booscaaa/estrova/internal/db"
	"github.com/spf13/cobra"
)

var restoreForce bool

var restoreCmd = &cobra.Command{
	Use:   "restore <arquivo>",
	Short: "Restaura o banco de dados a partir de um backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		src := args[0]

		if _, err := os.Stat(src); os.IsNotExist(err) {
			return fmt.Errorf("arquivo %q não encontrado", src)
		}

		dst := db.Path()

		if !restoreForce {
			fmt.Printf("Isso substituirá o banco atual em %s.\nConfirmar? [s/N] ", dst)
			reader := bufio.NewReader(os.Stdin)
			resp, _ := reader.ReadString('\n')
			if !strings.EqualFold(strings.TrimSpace(resp), "s") {
				fmt.Println("Restauração cancelada.")
				return nil
			}
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("restore falhou: %w", err)
		}

		info, _ := os.Stat(dst)
		fmt.Printf("Banco restaurado de %s (%d KB)\n", src, info.Size()/1024)
		return nil
	},
}

func init() {
	restoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "Substituir sem confirmação")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
