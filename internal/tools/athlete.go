package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/booscaaa/estrova/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterAthleteTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("estrova_get_athlete",
			mcp.WithDescription("Retorna o perfil do atleta autenticado: nome, cidade, país, gênero."),
		),
		handleGetAthlete,
	)

	s.AddTool(
		mcp.NewTool("estrova_get_athlete_stats",
			mcp.WithDescription("Retorna estatísticas do atleta: totais recentes, do ano e histórico para corrida, ciclismo e natação."),
		),
		handleGetAthleteStats,
	)

	s.AddTool(
		mcp.NewTool("estrova_get_athlete_zones",
			mcp.WithDescription("Retorna as zonas de frequência cardíaca e potência configuradas para o atleta."),
		),
		handleGetAthleteZones,
	)
}

func handleGetAthlete(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	// Try cache first
	if cached, err := database.LoadAthlete(); err == nil {
		return mcp.NewToolResultText(cached), nil
	}

	client, err := newStravaClient(ctx, database)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	athlete, err := client.GetAthlete()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar atleta: %v", err)), nil
	}

	data, err := json.MarshalIndent(athlete, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("erro ao serializar resposta"), nil
	}

	_ = database.SaveAthlete(string(data))

	return mcp.NewToolResultText(string(data)), nil
}

func handleGetAthleteStats(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	client, err := newStravaClient(ctx, database)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	athlete, err := client.GetAthlete()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar atleta: %v", err)), nil
	}

	stats, err := client.GetAthleteStats(athlete.ID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar stats: %v", err)), nil
	}

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("erro ao serializar resposta"), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func handleGetAthleteZones(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	client, err := newStravaClient(ctx, database)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	zones, err := client.GetAthleteZones()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar zonas: %v", err)), nil
	}

	data, err := json.MarshalIndent(zones, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("erro ao serializar resposta"), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
