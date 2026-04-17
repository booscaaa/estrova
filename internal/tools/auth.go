package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/booscaaa/estrova/internal/db"
	"github.com/booscaaa/estrova/internal/strava"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterAuthTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("estrova_authenticate",
			mcp.WithDescription("Inicia o fluxo OAuth2 completo: abre o browser automaticamente, aguarda o callback e salva o token no banco de dados. Use este tool para autenticar no Strava."),
		),
		handleAuthenticate,
	)

	s.AddTool(
		mcp.NewTool("estrova_auth_status",
			mcp.WithDescription("Verifica se já existe um token de acesso válido salvo no banco de dados."),
		),
		handleAuthStatus,
	)
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch {
	case runtime.GOOS == "linux" && isWSL():
		cmd = "powershell.exe"
		args = []string{"-c", fmt.Sprintf("Start-Process '%s'", url)}
	case runtime.GOOS == "linux":
		cmd = "xdg-open"
		args = []string{url}
	case runtime.GOOS == "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "cmd"
		args = []string{"/c", "start", url}
	}

	_ = exec.Command(cmd, args...).Start()
}

func handleAuthenticate(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	clientSecret := os.Getenv("STRAVA_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return mcp.NewToolResultError("STRAVA_CLIENT_ID e STRAVA_CLIENT_SECRET precisam estar definidos"), nil
	}

	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco de dados: %v", err)), nil
	}
	defer database.Close()

	codeChan, stopServer := strava.StartCallbackServer()
	defer stopServer()

	authURL := strava.GetAuthURL(clientID)
	openBrowser(authURL)

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var code string
	select {
	case code = <-codeChan:
	case <-timeoutCtx.Done():
		return mcp.NewToolResultError(
			"Timeout: nenhuma resposta em 2 minutos.\nURL de autorização:\n" + authURL,
		), nil
	}

	token, err := strava.ExchangeCode(ctx, clientID, clientSecret, code)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao trocar código: %v", err)), nil
	}

	if err := strava.SaveToken(database, token); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao salvar token: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Autenticado com sucesso!\nToken salvo no banco de dados.\nExpira em: %s\n\nAgora use estrova_sync para sincronizar suas atividades.",
		token.Expiry.Format(time.RFC3339),
	)), nil
}

func handleAuthStatus(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco de dados: %v", err)), nil
	}
	defer database.Close()

	token, err := strava.LoadToken(database)
	if err != nil {
		return mcp.NewToolResultText("Nenhum token encontrado. Use estrova_authenticate para autenticar."), nil
	}

	status := "válido"
	if token.Expiry.Before(time.Now()) {
		status = "expirado (será renovado automaticamente na próxima requisição)"
	}

	count, _ := database.CountActivities()

	return mcp.NewToolResultText(fmt.Sprintf(
		"Token: %s\nExpira em: %s\nAtividades no banco: %d",
		status,
		token.Expiry.Format(time.RFC3339),
		count,
	)), nil
}
