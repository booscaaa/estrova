package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/booscaaa/estrova/internal/analysis"
	"github.com/booscaaa/estrova/internal/db"
	"github.com/booscaaa/estrova/internal/strava"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterActivityTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("estrova_sync",
			mcp.WithDescription("Sincroniza atividades do Strava para o banco de dados local. Busca apenas atividades novas desde a última sincronização. Use antes de listar ou analisar atividades."),
			mcp.WithNumber("pages",
				mcp.Description("Número máximo de páginas a buscar (200 atividades por página). Default: 5"),
			),
		),
		handleSync,
	)

	s.AddTool(
		mcp.NewTool("estrova_list_activities",
			mcp.WithDescription("Lista atividades do banco de dados local. Execute estrova_sync primeiro para ter dados atualizados."),
			mcp.WithNumber("limit",
				mcp.Description("Quantidade máxima de atividades (default: 30)"),
			),
			mcp.WithString("after",
				mcp.Description("Filtrar atividades após esta data (formato: YYYY-MM-DD)"),
			),
			mcp.WithString("before",
				mcp.Description("Filtrar atividades antes desta data (formato: YYYY-MM-DD)"),
			),
			mcp.WithString("type",
				mcp.Description("Filtrar por tipo (ex: Run, Ride, Swim, Walk)"),
			),
		),
		handleListActivities,
	)

	s.AddTool(
		mcp.NewTool("estrova_get_activity",
			mcp.WithDescription("Retorna detalhes completos de uma atividade específica (laps, segmentos, melhores esforços). Busca direto da API do Strava."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID da atividade no Strava"),
			),
		),
		handleGetActivity,
	)
}

func newStravaClient(ctx context.Context, database *db.DB) (*strava.Client, error) {
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	clientSecret := os.Getenv("STRAVA_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("STRAVA_CLIENT_ID e STRAVA_CLIENT_SECRET precisam estar definidos")
	}

	token, err := strava.LoadToken(database)
	if err != nil {
		return nil, fmt.Errorf("token não encontrado — use estrova_authenticate para autenticar")
	}

	return strava.NewClient(ctx, database, clientID, clientSecret, token)
}

func handleSync(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	client, err := newStravaClient(ctx, database)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	maxPages := 5
	if v, ok := req.Params.Arguments["pages"].(float64); ok && v > 0 {
		maxPages = int(v)
	}

	// Fetch only new activities since last sync
	var afterUnix int64
	lastDate, _ := database.LastSyncDate()
	if lastDate != "" {
		t, err := time.Parse("2006-01-02T15:04:05Z", lastDate)
		if err == nil {
			afterUnix = t.Unix()
		}
	}

	total := 0
	var allActivities []strava.ActivitySummary
	for page := 1; page <= maxPages; page++ {
		activities, err := client.ListActivities(page, 200, 0, afterUnix)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar página %d: %v", page, err)), nil
		}
		if len(activities) == 0 {
			break
		}

		for _, a := range activities {
			data, _ := json.Marshal(a)
			row := db.ActivityRow{
				StravaID:       a.ID,
				Name:           a.Name,
				Type:           a.Type,
				SportType:      a.SportType,
				StartDate:      a.StartDate,
				StartDateLocal: a.StartDateLocal,
				Distance:       a.Distance,
				MovingTime:     a.MovingTime,
				ElapsedTime:    a.ElapsedTime,
				ElevationGain:  a.TotalElevationGain,
				AverageSpeed:   a.AverageSpeed,
				MaxSpeed:       a.MaxSpeed,
				AverageHR:      a.AverageHeartrate,
				MaxHR:          a.MaxHeartrate,
				AverageWatts:   a.AverageWatts,
				MaxWatts:       float64(a.MaxWatts),
				SufferScore:    a.SufferScore,
				Kilojoules:     a.Kilojoules,
				PRCount:        a.PRCount,
				KudosCount:     a.Kudos,
				Data:           string(data),
			}
			if err := database.UpsertActivity(row); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("erro ao salvar atividade %d: %v", a.ID, err)), nil
			}
			total++
			allActivities = append(allActivities, a)
		}

		if len(activities) < 200 {
			break
		}
	}

	// After saving all activities, match to plan sessions
	matched := 0
	for _, a := range allActivities {
		actDate := a.StartDate
		if len(a.StartDateLocal) >= 10 {
			actDate = a.StartDateLocal[:10]
		}
		session, err := database.FindSessionForActivity(a.SportType, actDate)
		if err != nil || session == nil {
			// try with Type fallback
			session, _ = database.FindSessionForActivity(a.Type, actDate)
		}
		if session == nil {
			continue
		}

		actRow := db.ActivityRow{
			StravaID:     a.ID,
			Distance:     a.Distance,
			MovingTime:   a.MovingTime,
			AverageSpeed: a.AverageSpeed,
			AverageHR:    a.AverageHeartrate,
			MaxHR:        a.MaxHeartrate,
		}

		result := analysis.Analyze(*session, actRow)

		actualPace := ""
		if a.AverageSpeed > 0 {
			secPerKm := 1000.0 / a.AverageSpeed
			m := int(secPerKm) / 60
			s := int(secPerKm) % 60
			actualPace = fmt.Sprintf("%d:%02d/km", m, s)
		}

		perf := db.ActualPerformance{
			StravaID:    a.ID,
			DistanceKm:  a.Distance / 1000,
			DurationMin: a.MovingTime / 60,
			AvgHR:       a.AverageHeartrate,
			ActualPace:  actualPace,
			Analysis:    strings.Join(append([]string{result.Summary}, result.Details...), "\n"),
			Score:       result.Score,
		}

		if err := database.MarkSessionWithActivity(session.ID, perf); err == nil {
			matched++
		}
	}

	count, _ := database.CountActivities()
	return mcp.NewToolResultText(fmt.Sprintf(
		"Sincronização concluída!\nNovas atividades: %d\nSessões marcadas como concluídas: %d\nTotal no banco: %d",
		total, matched, count,
	)), nil
}

func handleListActivities(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	limit := 30
	if v, ok := req.Params.Arguments["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	sportType, _ := req.Params.Arguments["type"].(string)
	after, _ := req.Params.Arguments["after"].(string)
	before, _ := req.Params.Arguments["before"].(string)

	rows, err := database.QueryActivities(sportType, after, before, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao consultar banco: %v", err)), nil
	}

	if len(rows) == 0 {
		count, _ := database.CountActivities()
		if count == 0 {
			return mcp.NewToolResultText("Nenhuma atividade no banco. Execute estrova_sync primeiro."), nil
		}
		return mcp.NewToolResultText("Nenhuma atividade encontrada com os filtros informados."), nil
	}

	// Rebuild JSON array from stored rows
	result := "[\n"
	for i, row := range rows {
		if i > 0 {
			result += ",\n"
		}
		result += row
	}
	result += "\n]"

	return mcp.NewToolResultText(fmt.Sprintf("Total: %d atividades\n\n%s", len(rows), result)), nil
}

func handleGetActivity(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	client, err := newStravaClient(ctx, database)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	idStr, ok := req.Params.Arguments["id"].(string)
	if !ok || idStr == "" {
		return mcp.NewToolResultError("parâmetro 'id' é obrigatório"), nil
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return mcp.NewToolResultError("'id' deve ser um número inteiro"), nil
	}

	activity, err := client.GetActivity(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar atividade: %v", err)), nil
	}

	data, err := json.MarshalIndent(activity, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("erro ao serializar resposta"), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
