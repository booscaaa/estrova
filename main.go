package main

import (
	"context"
	"fmt"
	"os"

	"github.com/booscaaa/estrova/internal/tools"
	"github.com/booscaaa/estrova/internal/web"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Start web server in background
	go web.Start(3030)

	s := server.NewMCPServer(
		"estrova",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	s.AddResource(
		mcp.NewResource("estrova://info", "Informações sobre o MCP do Strava", mcp.WithMIMEType("text/plain")),
		func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "estrova://info",
					MIMEType: "text/plain",
					Text: `Strava MCP Server v1.0.0 — Web UI: http://localhost:3030

Auth:
  estrova_authenticate      : autentica via OAuth2 (abre browser automaticamente)
  estrova_auth_status       : verifica status do token

Sincronização:
  estrova_sync              : sincroniza atividades para o banco SQLite

Atleta:
  estrova_get_athlete       : perfil do atleta
  estrova_get_athlete_stats : estatísticas (corrida, bike, natação)
  estrova_get_athlete_zones : zonas de FC e potência

Atividades:
  estrova_list_activities   : lista do banco local
  estrova_get_activity      : detalhes completos (laps, segmentos)

Objetivos & Planos:
  estrova_create_goal       : cria um objetivo de treino
  estrova_list_goals        : lista objetivos
  estrova_delete_goal       : remove objetivo
  estrova_analyze_for_goal  : coleta dados para gerar plano
  estrova_save_plan         : salva plano gerado
  estrova_get_plan          : retorna plano atual
`,
				},
			}, nil
		},
	)

	tools.RegisterAuthTools(s)
	tools.RegisterAthleteTools(s)
	tools.RegisterActivityTools(s)
	tools.RegisterGoalTools(s)

	fmt.Fprintln(os.Stderr, "Estrova MCP server iniciado — Web UI: http://localhost:3030")

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}
