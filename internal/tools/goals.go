package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/booscaaa/estrova/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterGoalTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("estrova_create_goal",
			mcp.WithDescription("Cria um novo objetivo de treino (ex: correr 10km em 50min, completar uma maratona, pedalar 100km)."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Nome do objetivo (ex: 'Maratona de São Paulo 2026')")),
			mcp.WithString("description", mcp.Description("Descrição detalhada do objetivo")),
			mcp.WithString("sport_type", mcp.Required(), mcp.Description("Modalidade: Run, Ride, Swim, Walk")),
			mcp.WithString("target_type", mcp.Required(), mcp.Description("Tipo de meta: distance (distância), pace (ritmo), time (tempo total), event (evento)")),
			mcp.WithString("target_value", mcp.Required(), mcp.Description("Valor alvo (ex: '42.2km', '5:00/km', 'sub-4h', 'completar')")),
			mcp.WithString("target_date", mcp.Description("Data alvo no formato YYYY-MM-DD")),
		),
		handleCreateGoal,
	)

	s.AddTool(
		mcp.NewTool("estrova_list_goals",
			mcp.WithDescription("Lista todos os objetivos de treino cadastrados."),
			mcp.WithString("status", mcp.Description("Filtrar por status: active, completed, cancelled (default: todos)")),
		),
		handleListGoals,
	)

	s.AddTool(
		mcp.NewTool("estrova_delete_goal",
			mcp.WithDescription("Remove um objetivo de treino e seu plano associado."),
			mcp.WithString("id", mcp.Required(), mcp.Description("ID do objetivo")),
		),
		handleDeleteGoal,
	)

	s.AddTool(
		mcp.NewTool("estrova_analyze_for_goal",
			mcp.WithDescription("Coleta dados do atleta e atividades recentes para análise e geração de plano de treino. Retorna contexto completo para o Claude criar um plano personalizado."),
			mcp.WithString("goal_id", mcp.Required(), mcp.Description("ID do objetivo")),
			mcp.WithNumber("recent_activities", mcp.Description("Quantidade de atividades recentes para analisar (default: 20)")),
		),
		handleAnalyzeForGoal,
	)

	s.AddTool(
		mcp.NewTool("estrova_save_plan",
			mcp.WithDescription("Salva o plano de treino gerado para um objetivo. O plano deve estar em formato JSON com semanas e sessões."),
			mcp.WithString("goal_id", mcp.Required(), mcp.Description("ID do objetivo")),
			mcp.WithString("plan_json", mcp.Required(), mcp.Description(`Plano em JSON com formato:
{
  "weeks": [
    {
      "week": 1,
      "phase": "Base",
      "focus": "Construção aeróbica",
      "sessions": [
        {
          "date": "2026-04-21",
          "day_of_week": "Monday",
          "session_type": "Easy",
          "sport_type": "Run",
          "description": "Corrida leve em zona 2",
          "distance_km": 6,
          "duration_min": 40,
          "pace_target": "6:30/km",
          "hr_zone": "Z2"
        }
      ]
    }
  ]
}`)),
		),
		handleSavePlan,
	)

	s.AddTool(
		mcp.NewTool("estrova_get_plan",
			mcp.WithDescription("Retorna o plano de treino atual de um objetivo, organizado por semanas."),
			mcp.WithString("goal_id", mcp.Required(), mcp.Description("ID do objetivo")),
		),
		handleGetPlan,
	)

	s.AddTool(
		mcp.NewTool("estrova_list_conflicts",
			mcp.WithDescription("Lista conflitos de agendamento entre objetivos ativos: dias onde dois objetivos têm sessões incompatíveis (ex: Tempo x Easy no mesmo dia)."),
		),
		handleListConflicts,
	)

	s.AddTool(
		mcp.NewTool("estrova_update_session",
			mcp.WithDescription("Atualiza campos de uma sessão do plano de treino (tipo, descrição, pace, zona HR, distância, duração, notas)."),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("ID da sessão")),
			mcp.WithString("session_type", mcp.Required(), mcp.Description("Tipo: Easy, Long, Tempo, Interval, Race, Rest, Cross, Strength")),
			mcp.WithString("description", mcp.Description("Descrição da sessão")),
			mcp.WithString("pace_target", mcp.Description("Pace alvo (ex: 6:30/km)")),
			mcp.WithString("hr_zone", mcp.Description("Zona de FC (ex: Z2)")),
			mcp.WithString("notes", mcp.Description("Notas adicionais")),
			mcp.WithNumber("distance_km", mcp.Description("Distância em km (0 para Rest)")),
			mcp.WithNumber("duration_min", mcp.Description("Duração em minutos (0 para Rest)")),
		),
		handleUpdateSession,
	)
}

func handleCreateGoal(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	g := db.Goal{
		Name:        req.Params.Arguments["name"].(string),
		SportType:   req.Params.Arguments["sport_type"].(string),
		TargetType:  req.Params.Arguments["target_type"].(string),
		TargetValue: req.Params.Arguments["target_value"].(string),
	}
	if v, ok := req.Params.Arguments["description"].(string); ok {
		g.Description = v
	}
	if v, ok := req.Params.Arguments["target_date"].(string); ok {
		g.TargetDate = v
	}

	id, err := database.CreateGoal(g)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao criar objetivo: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Objetivo criado com sucesso!\nID: %d\nNome: %s\nMeta: %s %s\n\nAgora use estrova_analyze_for_goal com goal_id=%d para gerar o plano de treino.",
		id, g.Name, g.TargetType, g.TargetValue, id,
	)), nil
}

func handleListGoals(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	status, _ := req.Params.Arguments["status"].(string)
	goals, err := database.ListGoals(status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao listar objetivos: %v", err)), nil
	}

	if len(goals) == 0 {
		return mcp.NewToolResultText("Nenhum objetivo cadastrado. Use estrova_create_goal para criar um."), nil
	}

	type goalWithProgress struct {
		db.Goal
		SessionsTotal int `json:"sessions_total"`
		SessionsDone  int `json:"sessions_done"`
	}

	result := make([]goalWithProgress, 0, len(goals))
	for _, g := range goals {
		total, done, _ := database.GoalProgress(g.ID)
		result = append(result, goalWithProgress{g, total, done})
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteGoal(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	idStr := req.Params.Arguments["id"].(string)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return mcp.NewToolResultError("'id' deve ser número inteiro"), nil
	}

	if err := database.DeleteGoal(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao deletar objetivo: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Objetivo %d removido.", id)), nil
}

func handleAnalyzeForGoal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	goalIDStr := req.Params.Arguments["goal_id"].(string)
	goalID, err := strconv.ParseInt(goalIDStr, 10, 64)
	if err != nil {
		return mcp.NewToolResultError("'goal_id' deve ser número inteiro"), nil
	}

	goal, err := database.GetGoal(goalID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("objetivo %d não encontrado", goalID)), nil
	}

	limit := 20
	if v, ok := req.Params.Arguments["recent_activities"].(float64); ok && v > 0 {
		limit = int(v)
	}

	// Load recent activities for this sport type
	recentJSON, err := database.QueryActivities(goal.SportType, "", "", limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar atividades: %v", err)), nil
	}

	// Load athlete profile
	athleteJSON, _ := database.LoadAthlete()

	// Activity summary by sport
	summary, _ := database.ActivitySummaryBySport()

	today := time.Now().Format("2006-01-02")

	type blockedDate struct {
		Date     string `json:"date"`
		GoalName string `json:"goal_name"`
		Reason   string `json:"reason"`
	}

	type schedulingConstraints struct {
		Instruction      string        `json:"instruction"`
		HardBlockedDates []blockedDate `json:"hard_blocked_dates"`
	}

	type otherGoalSessions struct {
		GoalID    int64            `json:"goal_id"`
		GoalName  string           `json:"goal_name"`
		SportType string           `json:"sport_type"`
		Sessions  []db.PlanSession `json:"sessions"`
	}

	type analysisPayload struct {
		Today               string                   `json:"today"`
		Goal                *db.Goal                 `json:"goal"`
		Athlete             interface{}              `json:"athlete"`
		Summary             []map[string]interface{} `json:"activity_summary_by_sport"`
		Activities          []interface{}            `json:"recent_activities"`
		OtherGoalSessions   []otherGoalSessions      `json:"other_goals_sessions,omitempty"`
		SchedulingContraints *schedulingConstraints   `json:"scheduling_constraints,omitempty"`
	}

	var athleteObj interface{}
	_ = json.Unmarshal([]byte(athleteJSON), &athleteObj)

	acts := make([]interface{}, 0, len(recentJSON))
	for _, raw := range recentJSON {
		var a interface{}
		if json.Unmarshal([]byte(raw), &a) == nil {
			acts = append(acts, a)
		}
	}

	// Collect future sessions from other active goals and compute hard blocked dates
	var otherSessions []otherGoalSessions
	var hardBlocked []blockedDate
	intenseSessions := map[string]bool{"Tempo": true, "Interval": true, "Race": true}

	allGoals, _ := database.ListGoals("active")
	for _, g := range allGoals {
		if g.ID == goalID {
			continue
		}
		sessions, err := database.GetPlanSessions(g.ID)
		if err != nil || len(sessions) == 0 {
			continue
		}
		var future []db.PlanSession
		for _, s := range sessions {
			if s.Date < today {
				continue
			}
			future = append(future, s)
			if intenseSessions[s.SessionType] {
				reason := fmt.Sprintf("%s tem %s", g.Name, s.SessionType)
				if s.PaceTarget != "" {
					reason += " (" + s.PaceTarget + ")"
				}
				hardBlocked = append(hardBlocked, blockedDate{
					Date:     s.Date,
					GoalName: g.Name,
					Reason:   reason,
				})
			}
		}
		if len(future) > 0 {
			otherSessions = append(otherSessions, otherGoalSessions{
				GoalID:    g.ID,
				GoalName:  g.Name,
				SportType: g.SportType,
				Sessions:  future,
			})
		}
	}

	var constraints *schedulingConstraints
	if len(hardBlocked) > 0 {
		constraints = &schedulingConstraints{
			Instruction:      "OBRIGATÓRIO: nas datas listadas em hard_blocked_dates outro objetivo já tem sessão intensa (Tempo/Interval/Race). Você DEVE agendar apenas Rest ou Easy nessas datas — NUNCA Tempo, Interval ou Race.",
			HardBlockedDates: hardBlocked,
		}
	}

	payload := analysisPayload{
		Today:               today,
		Goal:                goal,
		Athlete:             athleteObj,
		Summary:             summary,
		Activities:          acts,
		OtherGoalSessions:   otherSessions,
		SchedulingContraints: constraints,
	}

	data, _ := json.MarshalIndent(payload, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleSavePlan(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	goalIDStr := req.Params.Arguments["goal_id"].(string)
	goalID, err := strconv.ParseInt(goalIDStr, 10, 64)
	if err != nil {
		return mcp.NewToolResultError("'goal_id' deve ser número inteiro"), nil
	}

	planJSON := req.Params.Arguments["plan_json"].(string)

	var plan struct {
		Weeks []struct {
			Week     int    `json:"week"`
			Phase    string `json:"phase"`
			Focus    string `json:"focus"`
			Sessions []struct {
				Date             string  `json:"date"`
				DayOfWeek        string  `json:"day_of_week"`
				SessionType      string  `json:"session_type"`
				SportType        string  `json:"sport_type"`
				Description      string  `json:"description"`
				DistanceKm       float64 `json:"distance_km"`
				DurationMin      int     `json:"duration_min"`
				PaceTarget       string  `json:"pace_target"`
				HRZone           string  `json:"hr_zone"`
				Notes            string  `json:"notes"`
				NutritionPre     string  `json:"nutrition_pre"`
				NutritionDuring  string  `json:"nutrition_during"`
				NutritionPost    string  `json:"nutrition_post"`
			} `json:"sessions"`
		} `json:"weeks"`
	}

	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("JSON inválido: %v", err)), nil
	}

	// Delete old plan and save new one
	if err := database.DeletePlanSessions(goalID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao limpar plano antigo: %v", err)), nil
	}

	totalSessions := 0
	for _, week := range plan.Weeks {
		for _, s := range week.Sessions {
			session := db.PlanSession{
				GoalID:          goalID,
				Date:            s.Date,
				WeekNumber:      week.Week,
				DayOfWeek:       s.DayOfWeek,
				SessionType:     s.SessionType,
				SportType:       s.SportType,
				Description:     s.Description,
				DistanceKm:      s.DistanceKm,
				DurationMin:     s.DurationMin,
				PaceTarget:      s.PaceTarget,
				HRZone:          s.HRZone,
				Notes:           s.Notes,
				NutritionPre:    s.NutritionPre,
				NutritionDuring: s.NutritionDuring,
				NutritionPost:   s.NutritionPost,
			}
			if err := database.InsertPlanSession(session); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("erro ao salvar sessão: %v", err)), nil
			}
			totalSessions++
		}
	}

	conflicts, _ := database.DetectConflicts()

	msg := fmt.Sprintf(
		"Plano salvo com sucesso!\nSemanas: %d\nSessões totais: %d\n\nAcesse http://localhost:3030 para visualizar o plano.",
		len(plan.Weeks), totalSessions,
	)

	if len(conflicts) > 0 {
		msg += fmt.Sprintf("\n\n⚠️  CONFLITOS DETECTADOS (%d):\n", len(conflicts))
		for _, c := range conflicts {
			msg += fmt.Sprintf("  • %s: %s (%s %s) x %s (%s %s)\n",
				c.Date,
				c.GoalName1, c.SessionType1, c.PaceTarget1,
				c.GoalName2, c.SessionType2, c.PaceTarget2,
			)
		}
		msg += "\nPor favor, revise o plano e ajuste as sessões conflitantes para Rest ou Easy."
	}

	return mcp.NewToolResultText(msg), nil
}

func handleGetPlan(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	goalIDStr := req.Params.Arguments["goal_id"].(string)
	goalID, err := strconv.ParseInt(goalIDStr, 10, 64)
	if err != nil {
		return mcp.NewToolResultError("'goal_id' deve ser número inteiro"), nil
	}

	goal, err := database.GetGoal(goalID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("objetivo %d não encontrado", goalID)), nil
	}

	sessions, err := database.GetPlanSessions(goalID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar plano: %v", err)), nil
	}

	if len(sessions) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf(
			"Nenhum plano encontrado para o objetivo '%s'.\nUse estrova_analyze_for_goal para gerar um plano.",
			goal.Name,
		)), nil
	}

	// Group by week
	weekMap := map[int][]db.PlanSession{}
	for _, s := range sessions {
		weekMap[s.WeekNumber] = append(weekMap[s.WeekNumber], s)
	}

	total, done, _ := database.GoalProgress(goalID)

	result := map[string]interface{}{
		"goal":           goal,
		"progress":       map[string]int{"total": total, "completed": done},
		"sessions_by_week": weekMap,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListConflicts(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	conflicts, err := database.DetectConflicts()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao detectar conflitos: %v", err)), nil
	}
	if len(conflicts) == 0 {
		return mcp.NewToolResultText("Nenhum conflito detectado entre os objetivos ativos."), nil
	}

	data, _ := json.MarshalIndent(conflicts, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("%d conflito(s) detectado(s):\n\n%s", len(conflicts), string(data))), nil
}

func handleUpdateSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database, err := db.Open()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao abrir banco: %v", err)), nil
	}
	defer database.Close()

	idStr := req.Params.Arguments["session_id"].(string)
	sessionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return mcp.NewToolResultError("'session_id' deve ser número inteiro"), nil
	}

	sessionType := req.Params.Arguments["session_type"].(string)
	description, _ := req.Params.Arguments["description"].(string)
	paceTarget, _ := req.Params.Arguments["pace_target"].(string)
	hrZone, _ := req.Params.Arguments["hr_zone"].(string)
	notes, _ := req.Params.Arguments["notes"].(string)
	distanceKm, _ := req.Params.Arguments["distance_km"].(float64)
	durationMin := 0
	if v, ok := req.Params.Arguments["duration_min"].(float64); ok {
		durationMin = int(v)
	}

	if err := database.UpdatePlanSession(sessionID, sessionType, description, paceTarget, hrZone, notes, distanceKm, durationMin); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("erro ao atualizar sessão: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Sessão %d atualizada: %s", sessionID, sessionType)), nil
}
