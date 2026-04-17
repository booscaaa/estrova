package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"

	"github.com/booscaaa/estrova/internal/db"
	"github.com/booscaaa/estrova/internal/strava"
)

//go:embed static
var staticFiles embed.FS

func Start(port int) {
	mux := http.NewServeMux()

	// Static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// API
	mux.HandleFunc("/api/goals", handleGoals)
	mux.HandleFunc("/api/goals/", handleGoalByID)
	mux.HandleFunc("/api/sessions/", handleSessions)
mux.HandleFunc("/api/summary", handleSummary)
	mux.HandleFunc("/api/conflicts", handleConflicts)
	mux.HandleFunc("/api/activities", handleActivities)
	mux.HandleFunc("/api/activities/", handleActivityByID)
	mux.HandleFunc("/api/dashboard", handleDashboard)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Web server iniciado em http://localhost%s\n", addr)
	_ = http.ListenAndServe(addr, mux)
}

func json200(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func handleGoals(w http.ResponseWriter, r *http.Request) {
	database, err := db.Open()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	defer database.Close()

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		goals, err := database.ListGoals(status)
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
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
		if result == nil {
			result = []goalWithProgress{}
		}
		json200(w, result)

	case http.MethodPost:
		var g db.Goal
		if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
			jsonErr(w, 400, "JSON inválido")
			return
		}
		id, err := database.CreateGoal(g)
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		g.ID = id
		g.Status = "active"
		json200(w, g)

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

func handleGoalByID(w http.ResponseWriter, r *http.Request) {
	database, err := db.Open()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Parse path: /api/goals/{id} or /api/goals/{id}/plan or /api/goals/{id}/activities
	path := r.URL.Path[len("/api/goals/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		jsonErr(w, 400, "goal id required")
		return
	}

	goalID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonErr(w, 400, "invalid goal id")
		return
	}

	// /api/goals/{id}/plan
	if len(parts) == 2 && parts[1] == "plan" {
		handlePlan(w, r, database, goalID)
		return
	}

	// /api/goals/{id}/activities
	if len(parts) == 2 && parts[1] == "activities" {
		goal, err := database.GetGoal(goalID)
		if err != nil {
			jsonErr(w, 404, "goal not found")
			return
		}
		// Filter by plan date range when a plan exists
		minDate, maxDate, _ := database.PlanDateRange(goalID)
		var rows []string
		if minDate != "" && maxDate != "" {
			rows, err = database.QueryActivitiesInRange(goal.SportType, minDate, maxDate, 200)
		} else {
			rows, err = database.QueryActivities(goal.SportType, "", "", 50)
		}
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		acts := make([]interface{}, 0, len(rows))
		for _, raw := range rows {
			var a interface{}
			if json.Unmarshal([]byte(raw), &a) == nil {
				acts = append(acts, a)
			}
		}
		json200(w, acts)
		return
	}

	// /api/goals/{id}
	switch r.Method {
	case http.MethodGet:
		goal, err := database.GetGoal(goalID)
		if err != nil {
			jsonErr(w, 404, "not found")
			return
		}
		total, done, _ := database.GoalProgress(goalID)
		json200(w, map[string]interface{}{
			"goal":           goal,
			"sessions_total": total,
			"sessions_done":  done,
		})

	case http.MethodPut:
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "JSON inválido")
			return
		}
		if err := database.UpdateGoalStatus(goalID, body.Status); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		json200(w, map[string]string{"ok": "updated"})

	case http.MethodDelete:
		if err := database.DeleteGoal(goalID); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		json200(w, map[string]string{"ok": "deleted"})

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

func handlePlan(w http.ResponseWriter, r *http.Request, database *db.DB, goalID int64) {
	if r.Method != http.MethodGet {
		jsonErr(w, 405, "method not allowed")
		return
	}

	goal, err := database.GetGoal(goalID)
	if err != nil {
		jsonErr(w, 404, "goal not found")
		return
	}

	sessions, err := database.GetPlanSessions(goalID)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}

	total, done, _ := database.GoalProgress(goalID)

	// Group sessions by week
	type weekData struct {
		Week     int               `json:"week"`
		Sessions []db.PlanSession  `json:"sessions"`
	}
	weekMap := map[int]*weekData{}
	weekOrder := []int{}
	for _, s := range sessions {
		if _, ok := weekMap[s.WeekNumber]; !ok {
			weekMap[s.WeekNumber] = &weekData{Week: s.WeekNumber}
			weekOrder = append(weekOrder, s.WeekNumber)
		}
		weekMap[s.WeekNumber].Sessions = append(weekMap[s.WeekNumber].Sessions, s)
	}

	weeks := make([]weekData, 0, len(weekOrder))
	for _, wn := range weekOrder {
		weeks = append(weeks, *weekMap[wn])
	}

	json200(w, map[string]interface{}{
		"goal":     goal,
		"weeks":    weeks,
		"progress": map[string]int{"total": total, "completed": done},
	})
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/sessions/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		jsonErr(w, 400, "session id required")
		return
	}

	sessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonErr(w, 400, "invalid session id")
		return
	}

	// /api/sessions/{id}/complete
	if len(parts) == 2 && parts[1] == "complete" {
		if r.Method != http.MethodPut {
			jsonErr(w, 405, "method not allowed")
			return
		}
		var body struct{ Completed bool `json:"completed"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "JSON inválido")
			return
		}
		database, err := db.Open()
		if err != nil { jsonErr(w, 500, err.Error()); return }
		defer database.Close()
		if err := database.ToggleSessionComplete(sessionID, body.Completed); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		json200(w, map[string]bool{"completed": body.Completed})
		return
	}

	// /api/sessions/{id}
	database, err := db.Open()
	if err != nil { jsonErr(w, 500, err.Error()); return }
	defer database.Close()

	switch r.Method {
	case http.MethodGet:
		s, err := database.GetPlanSessionByID(sessionID)
		if err != nil { jsonErr(w, 404, "session not found"); return }
		json200(w, s)

	case http.MethodPut:
		var body struct {
			SessionType string  `json:"session_type"`
			Description string  `json:"description"`
			PaceTarget  string  `json:"pace_target"`
			HRZone      string  `json:"hr_zone"`
			Notes       string  `json:"notes"`
			DistanceKm  float64 `json:"distance_km"`
			DurationMin int     `json:"duration_min"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "JSON inválido")
			return
		}
		if err := database.UpdatePlanSession(sessionID, body.SessionType, body.Description, body.PaceTarget, body.HRZone, body.Notes, body.DistanceKm, body.DurationMin); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		s, _ := database.GetPlanSessionByID(sessionID)
		json200(w, s)

	case http.MethodDelete:
		if err := database.DeletePlanSessionByID(sessionID); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		json200(w, map[string]string{"ok": "deleted"})

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

func handleSummary(w http.ResponseWriter, _ *http.Request) {
	database, err := db.Open()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	defer database.Close()

	summary, _ := database.ActivitySummaryBySport()
	count, _ := database.CountActivities()
	goals, _ := database.ListGoals("active")

	json200(w, map[string]interface{}{
		"total_activities":  count,
		"active_goals":      len(goals),
		"summary_by_sport":  summary,
	})
}

func handleDashboard(w http.ResponseWriter, _ *http.Request) {
	database, err := db.Open()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	defer database.Close()

	type goalWithProgress struct {
		db.Goal
		SessionsTotal int `json:"sessions_total"`
		SessionsDone  int `json:"sessions_done"`
	}

	summary, _ := database.ActivitySummaryBySport()
	count, _ := database.CountActivities()
	goals, _ := database.ListGoals("active")
	weekVol, _ := database.WeeklyVolume(84)
	paceTrend, _ := database.RunPaceTrend(84)
	conflicts, _ := database.DetectConflicts()

	goalsOut := make([]goalWithProgress, 0, len(goals))
	for _, g := range goals {
		total, done, _ := database.GoalProgress(g.ID)
		goalsOut = append(goalsOut, goalWithProgress{g, total, done})
	}
	if weekVol == nil {
		weekVol = []db.WeekVolumeRow{}
	}
	if paceTrend == nil {
		paceTrend = []db.PaceTrendRow{}
	}

	json200(w, map[string]interface{}{
		"total_activities": count,
		"active_goals":     len(goals),
		"summary_by_sport": summary,
		"weekly_volume":    weekVol,
		"pace_trend":       paceTrend,
		"goals":            goalsOut,
		"conflicts_count":  len(conflicts),
	})
}

func handleActivities(w http.ResponseWriter, r *http.Request) {
	database, err := db.Open()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	defer database.Close()

	q := r.URL.Query()
	sport := q.Get("sport")
	after := q.Get("after")
	before := q.Get("before")
	search := q.Get("q")
	limit := 100
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	rows, err := database.QueryActivitiesFiltered(sport, after, before, search, limit)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	total, _ := database.CountActivitiesFiltered(sport, after, before, search)

	acts := make([]interface{}, 0, len(rows))
	for _, raw := range rows {
		var a interface{}
		if json.Unmarshal([]byte(raw), &a) == nil {
			acts = append(acts, a)
		}
	}

	json200(w, map[string]interface{}{
		"total":      total,
		"count":      len(acts),
		"activities": acts,
	})
}

func newStravaClient(database *db.DB) (*strava.Client, error) {
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	clientSecret := os.Getenv("STRAVA_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("STRAVA_CLIENT_ID e STRAVA_CLIENT_SECRET não definidos")
	}
	token, err := strava.LoadToken(database)
	if err != nil {
		return nil, fmt.Errorf("token não encontrado — autentique primeiro")
	}
	return strava.NewClient(context.Background(), database, clientID, clientSecret, token)
}

func handleActivityByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/activities/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		jsonErr(w, 400, "activity id required")
		return
	}

	actID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonErr(w, 400, "invalid activity id")
		return
	}

	if len(parts) == 2 && parts[1] == "detail" {
		database, err := db.Open()
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		defer database.Close()

		client, err := newStravaClient(database)
		if err != nil {
			jsonErr(w, 503, err.Error())
			return
		}

		detail, err := client.GetActivity(actID)
		if err != nil {
			jsonErr(w, 502, err.Error())
			return
		}

		json200(w, detail)
		return
	}

	jsonErr(w, 404, "not found")
}

func handleConflicts(w http.ResponseWriter, _ *http.Request) {
	database, err := db.Open()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	defer database.Close()

	conflicts, err := database.DetectConflicts()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	if conflicts == nil {
		conflicts = []db.ConflictInfo{}
	}
	json200(w, conflicts)
}


func splitPath(path string) []string {
	var parts []string
	cur := ""
	for _, c := range path {
		if c == '/' {
			if cur != "" {
				parts = append(parts, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}
