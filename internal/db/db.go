package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".estrova.db")
}

func Open() (*DB, error) {
	path := Path()

	sqlDB, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	db := &DB{sql: sqlDB}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (d *DB) Close() {
	_ = d.sql.Close()
}

// Backup creates a consistent copy of the database at dst using VACUUM INTO.
func (d *DB) Backup(dst string) error {
	_, err := d.sql.Exec(`VACUUM INTO ?`, dst)
	return err
}

func (d *DB) migrate() error {
	_, err := d.sql.Exec(`
		CREATE TABLE IF NOT EXISTS tokens (
			id            INTEGER PRIMARY KEY,
			access_token  TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			token_type    TEXT NOT NULL,
			expiry        DATETIME NOT NULL,
			updated_at    DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS athlete (
			id         INTEGER PRIMARY KEY,
			data       TEXT NOT NULL,
			synced_at  DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS activities (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			strava_id        INTEGER UNIQUE NOT NULL,
			name             TEXT,
			type             TEXT,
			sport_type       TEXT,
			start_date       TEXT,
			start_date_local TEXT,
			distance         REAL,
			moving_time      INTEGER,
			elapsed_time     INTEGER,
			elevation_gain   REAL,
			average_speed    REAL,
			max_speed        REAL,
			average_heartrate REAL,
			max_heartrate    REAL,
			average_watts    REAL,
			max_watts        REAL,
			suffer_score     REAL,
			kilojoules       REAL,
			pr_count         INTEGER,
			kudos_count      INTEGER,
			data             TEXT NOT NULL,
			synced_at        DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS goals (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			name         TEXT NOT NULL,
			description  TEXT,
			sport_type   TEXT NOT NULL,
			target_type  TEXT NOT NULL,
			target_value TEXT NOT NULL,
			target_date  TEXT,
			status       TEXT NOT NULL DEFAULT 'active',
			created_at   DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS plan_sessions (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			goal_id         INTEGER NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
			date            TEXT NOT NULL,
			week_number     INTEGER NOT NULL,
			day_of_week     TEXT NOT NULL,
			session_type    TEXT NOT NULL,
			sport_type      TEXT NOT NULL,
			description     TEXT,
			distance_km     REAL DEFAULT 0,
			duration_min    INTEGER DEFAULT 0,
			pace_target     TEXT,
			hr_zone         TEXT,
			notes           TEXT,
			completed       INTEGER NOT NULL DEFAULT 0,
			activity_id     INTEGER,
			created_at      DATETIME NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_activities_start_date ON activities(start_date_local);
		CREATE INDEX IF NOT EXISTS idx_activities_type ON activities(sport_type);
		CREATE INDEX IF NOT EXISTS idx_sessions_goal ON plan_sessions(goal_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_date ON plan_sessions(date);
	`)
	if err != nil {
		return err
	}
	// Additive migrations — ignore "duplicate column" errors
	for _, stmt := range []string{
		`ALTER TABLE plan_sessions ADD COLUMN nutrition_pre     TEXT DEFAULT ''`,
		`ALTER TABLE plan_sessions ADD COLUMN nutrition_during  TEXT DEFAULT ''`,
		`ALTER TABLE plan_sessions ADD COLUMN nutrition_post    TEXT DEFAULT ''`,
		`ALTER TABLE plan_sessions ADD COLUMN activity_strava_id  INTEGER DEFAULT 0`,
		`ALTER TABLE plan_sessions ADD COLUMN actual_distance_km  REAL    DEFAULT 0`,
		`ALTER TABLE plan_sessions ADD COLUMN actual_duration_min INTEGER DEFAULT 0`,
		`ALTER TABLE plan_sessions ADD COLUMN actual_avg_hr       REAL    DEFAULT 0`,
		`ALTER TABLE plan_sessions ADD COLUMN actual_pace         TEXT    DEFAULT ''`,
		`ALTER TABLE plan_sessions ADD COLUMN analysis            TEXT    DEFAULT ''`,
		`ALTER TABLE plan_sessions ADD COLUMN performance_score   REAL    DEFAULT 0`,
	} {
		_, _ = d.sql.Exec(stmt) // ignore error if column already exists
	}
	return nil
}

// ─── Token ────────────────────────────────────────────────────────────────────

type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
}

func (d *DB) SaveToken(t Token) error {
	_, err := d.sql.Exec(`
		INSERT INTO tokens (id, access_token, refresh_token, token_type, expiry, updated_at)
		VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			access_token=excluded.access_token, refresh_token=excluded.refresh_token,
			token_type=excluded.token_type, expiry=excluded.expiry, updated_at=excluded.updated_at
	`, t.AccessToken, t.RefreshToken, t.TokenType, t.Expiry.UTC().Format(time.RFC3339), time.Now().UTC())
	return err
}

func (d *DB) LoadToken() (*Token, error) {
	row := d.sql.QueryRow(`SELECT access_token, refresh_token, token_type, expiry FROM tokens WHERE id=1`)
	var t Token
	var expiry string
	if err := row.Scan(&t.AccessToken, &t.RefreshToken, &t.TokenType, &expiry); err != nil {
		return nil, err
	}
	t.Expiry, _ = time.Parse(time.RFC3339, expiry)
	return &t, nil
}

// ─── Athlete ──────────────────────────────────────────────────────────────────

func (d *DB) SaveAthlete(data string) error {
	_, err := d.sql.Exec(`
		INSERT INTO athlete (id, data, synced_at) VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET data=excluded.data, synced_at=excluded.synced_at
	`, data, time.Now().UTC())
	return err
}

func (d *DB) LoadAthlete() (string, error) {
	var data string
	err := d.sql.QueryRow(`SELECT data FROM athlete WHERE id=1`).Scan(&data)
	return data, err
}

// ─── Activities ───────────────────────────────────────────────────────────────

type ActivityRow struct {
	StravaID       int64
	Name           string
	Type           string
	SportType      string
	StartDate      string
	StartDateLocal string
	Distance       float64
	MovingTime     int
	ElapsedTime    int
	ElevationGain  float64
	AverageSpeed   float64
	MaxSpeed       float64
	AverageHR      float64
	MaxHR          float64
	AverageWatts   float64
	MaxWatts       float64
	SufferScore    float64
	Kilojoules     float64
	PRCount        int
	KudosCount     int
	Data           string
}

func (d *DB) UpsertActivity(a ActivityRow) error {
	_, err := d.sql.Exec(`
		INSERT INTO activities (
			strava_id, name, type, sport_type, start_date, start_date_local,
			distance, moving_time, elapsed_time, elevation_gain,
			average_speed, max_speed, average_heartrate, max_heartrate,
			average_watts, max_watts, suffer_score, kilojoules,
			pr_count, kudos_count, data, synced_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(strava_id) DO UPDATE SET
			name=excluded.name, data=excluded.data, synced_at=excluded.synced_at,
			average_heartrate=excluded.average_heartrate, suffer_score=excluded.suffer_score
	`,
		a.StravaID, a.Name, a.Type, a.SportType, a.StartDate, a.StartDateLocal,
		a.Distance, a.MovingTime, a.ElapsedTime, a.ElevationGain,
		a.AverageSpeed, a.MaxSpeed, a.AverageHR, a.MaxHR,
		a.AverageWatts, a.MaxWatts, a.SufferScore, a.Kilojoules,
		a.PRCount, a.KudosCount, a.Data, time.Now().UTC(),
	)
	return err
}

func (d *DB) QueryActivities(sportType, after, before string, limit int) ([]string, error) {
	query := `SELECT data FROM activities WHERE 1=1`
	args := []interface{}{}

	if sportType != "" {
		query += ` AND (sport_type=? OR type=?)`
		args = append(args, sportType, sportType)
	}
	if after != "" {
		query += ` AND start_date_local >= ?`
		args = append(args, after)
	}
	if before != "" {
		query += ` AND start_date_local <= ?`
		args = append(args, before)
	}
	query += ` ORDER BY start_date_local DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		results = append(results, data)
	}
	return results, rows.Err()
}

func (d *DB) QueryActivitiesFiltered(sportType, after, before, search string, limit int) ([]string, error) {
	query := `SELECT data FROM activities WHERE 1=1`
	args := []interface{}{}
	if sportType != "" {
		query += ` AND (sport_type=? OR type=?)`
		args = append(args, sportType, sportType)
	}
	if after != "" {
		query += ` AND substr(start_date_local,1,10) >= ?`
		args = append(args, after)
	}
	if before != "" {
		query += ` AND substr(start_date_local,1,10) <= ?`
		args = append(args, before)
	}
	if search != "" {
		query += ` AND name LIKE ?`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY start_date_local DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		results = append(results, data)
	}
	return results, rows.Err()
}

func (d *DB) CountActivitiesFiltered(sportType, after, before, search string) (int, error) {
	query := `SELECT COUNT(*) FROM activities WHERE 1=1`
	args := []interface{}{}
	if sportType != "" {
		query += ` AND (sport_type=? OR type=?)`
		args = append(args, sportType, sportType)
	}
	if after != "" {
		query += ` AND substr(start_date_local,1,10) >= ?`
		args = append(args, after)
	}
	if before != "" {
		query += ` AND substr(start_date_local,1,10) <= ?`
		args = append(args, before)
	}
	if search != "" {
		query += ` AND name LIKE ?`
		args = append(args, "%"+search+"%")
	}
	var count int
	err := d.sql.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (d *DB) CountActivities() (int, error) {
	var count int
	err := d.sql.QueryRow(`SELECT COUNT(*) FROM activities`).Scan(&count)
	return count, err
}

func (d *DB) LastSyncDate() (string, error) {
	var date sql.NullString
	err := d.sql.QueryRow(`SELECT MAX(start_date_local) FROM activities`).Scan(&date)
	if err != nil || !date.Valid {
		return "", err
	}
	return date.String, nil
}

func (d *DB) ActivitySummaryBySport() ([]map[string]interface{}, error) {
	rows, err := d.sql.Query(`
		SELECT sport_type, COUNT(*) as count,
			ROUND(SUM(distance)/1000, 1) as total_km,
			SUM(moving_time) as total_seconds,
			ROUND(AVG(average_heartrate), 0) as avg_hr
		FROM activities
		WHERE sport_type != ''
		GROUP BY sport_type
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var sportType string
		var count, totalSeconds int
		var totalKm, avgHR float64
		if err := rows.Scan(&sportType, &count, &totalKm, &totalSeconds, &avgHR); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"sport_type":    sportType,
			"count":         count,
			"total_km":      totalKm,
			"total_hours":   totalSeconds / 3600,
			"avg_heartrate": avgHR,
		})
	}
	return result, rows.Err()
}

type WeekVolumeRow struct {
	Week      string  `json:"week"`
	SportType string  `json:"sport_type"`
	Km        float64 `json:"km"`
	Count     int     `json:"count"`
	Hours     float64 `json:"hours"`
}

func (d *DB) WeeklyVolume(days int) ([]WeekVolumeRow, error) {
	rows, err := d.sql.Query(fmt.Sprintf(`
		SELECT strftime('%%Y-W%%W', substr(start_date_local,1,10)) as week,
		       sport_type,
		       ROUND(SUM(distance)/1000.0,1) as km,
		       COUNT(*) as cnt,
		       ROUND(SUM(moving_time)/3600.0,2) as hours
		FROM activities
		WHERE start_date_local >= date('now','-%d days') AND distance > 0
		GROUP BY week, sport_type
		ORDER BY week ASC
	`, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []WeekVolumeRow
	for rows.Next() {
		var r WeekVolumeRow
		if err := rows.Scan(&r.Week, &r.SportType, &r.Km, &r.Count, &r.Hours); err != nil {
			continue
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

type PaceTrendRow struct {
	Week     string  `json:"week"`
	AvgSpeed float64 `json:"avg_speed_ms"`
	AvgHR    float64 `json:"avg_hr"`
	Count    int     `json:"count"`
}

func (d *DB) RunPaceTrend(days int) ([]PaceTrendRow, error) {
	rows, err := d.sql.Query(fmt.Sprintf(`
		SELECT strftime('%%Y-W%%W', substr(start_date_local,1,10)) as week,
		       ROUND(AVG(average_speed),5) as avg_speed,
		       ROUND(AVG(CASE WHEN average_heartrate > 0 THEN average_heartrate END),1) as avg_hr,
		       COUNT(*) as cnt
		FROM activities
		WHERE (sport_type='Run' OR type='Run')
		  AND start_date_local >= date('now','-%d days')
		  AND average_speed > 0
		GROUP BY week ORDER BY week ASC
	`, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PaceTrendRow
	for rows.Next() {
		var r PaceTrendRow
		if err := rows.Scan(&r.Week, &r.AvgSpeed, &r.AvgHR, &r.Count); err != nil {
			continue
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ─── Goals ────────────────────────────────────────────────────────────────────

type Goal struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SportType   string `json:"sport_type"`
	TargetType  string `json:"target_type"`
	TargetValue string `json:"target_value"`
	TargetDate  string `json:"target_date"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func (d *DB) CreateGoal(g Goal) (int64, error) {
	res, err := d.sql.Exec(`
		INSERT INTO goals (name, description, sport_type, target_type, target_value, target_date, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?)
	`, g.Name, g.Description, g.SportType, g.TargetType, g.TargetValue, g.TargetDate, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ListGoals(status string) ([]Goal, error) {
	query := `SELECT id, name, description, sport_type, target_type, target_value, target_date, status, created_at FROM goals`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status=?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []Goal
	for rows.Next() {
		var g Goal
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.SportType, &g.TargetType, &g.TargetValue, &g.TargetDate, &g.Status, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func (d *DB) GetGoal(id int64) (*Goal, error) {
	row := d.sql.QueryRow(`SELECT id, name, description, sport_type, target_type, target_value, target_date, status, created_at FROM goals WHERE id=?`, id)
	var g Goal
	if err := row.Scan(&g.ID, &g.Name, &g.Description, &g.SportType, &g.TargetType, &g.TargetValue, &g.TargetDate, &g.Status, &g.CreatedAt); err != nil {
		return nil, err
	}
	return &g, nil
}

func (d *DB) UpdateGoalStatus(id int64, status string) error {
	_, err := d.sql.Exec(`UPDATE goals SET status=? WHERE id=?`, status, id)
	return err
}

func (d *DB) DeleteGoal(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM goals WHERE id=?`, id)
	return err
}

// ─── Plan Sessions ────────────────────────────────────────────────────────────

type PlanSession struct {
	ID               int64   `json:"id"`
	GoalID           int64   `json:"goal_id"`
	Date             string  `json:"date"`
	WeekNumber       int     `json:"week_number"`
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
	Completed        bool    `json:"completed"`
	ActivityID       *int64  `json:"activity_id,omitempty"`
	ActivityStravaID int64   `json:"activity_strava_id"`
	ActualDistanceKm float64 `json:"actual_distance_km"`
	ActualDurationMin int    `json:"actual_duration_min"`
	ActualAvgHR      float64 `json:"actual_avg_hr"`
	ActualPace       string  `json:"actual_pace"`
	Analysis         string  `json:"analysis"`
	PerformanceScore float64 `json:"performance_score"`
}

func (d *DB) GetPlanSessionByID(id int64) (*PlanSession, error) {
	row := d.sql.QueryRow(`
		SELECT id, goal_id, date, week_number, day_of_week, session_type, sport_type,
			description, distance_km, duration_min, pace_target, hr_zone, notes,
			COALESCE(nutrition_pre,''), COALESCE(nutrition_during,''), COALESCE(nutrition_post,''),
			completed,
			COALESCE(activity_strava_id,0), COALESCE(actual_distance_km,0),
			COALESCE(actual_duration_min,0), COALESCE(actual_avg_hr,0),
			COALESCE(actual_pace,''), COALESCE(analysis,''), COALESCE(performance_score,0)
		FROM plan_sessions WHERE id=?
	`, id)
	var s PlanSession
	var completed int
	err := row.Scan(
		&s.ID, &s.GoalID, &s.Date, &s.WeekNumber, &s.DayOfWeek,
		&s.SessionType, &s.SportType, &s.Description, &s.DistanceKm,
		&s.DurationMin, &s.PaceTarget, &s.HRZone, &s.Notes,
		&s.NutritionPre, &s.NutritionDuring, &s.NutritionPost,
		&completed,
		&s.ActivityStravaID, &s.ActualDistanceKm, &s.ActualDurationMin,
		&s.ActualAvgHR, &s.ActualPace, &s.Analysis, &s.PerformanceScore,
	)
	if err != nil {
		return nil, err
	}
	s.Completed = completed == 1
	return &s, nil
}

func (d *DB) DeletePlanSessionByID(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM plan_sessions WHERE id=?`, id)
	return err
}

// UpdatePlanSession alters mutable fields of an existing session.
func (d *DB) UpdatePlanSession(id int64, sessionType, description, paceTarget, hrZone, notes string, distanceKm float64, durationMin int) error {
	_, err := d.sql.Exec(`
		UPDATE plan_sessions SET
			session_type=?, description=?, pace_target=?, hr_zone=?,
			notes=?, distance_km=?, duration_min=?
		WHERE id=?
	`, sessionType, description, paceTarget, hrZone, notes, distanceKm, durationMin, id)
	return err
}

func (d *DB) DeletePlanSessions(goalID int64) error {
	_, err := d.sql.Exec(`DELETE FROM plan_sessions WHERE goal_id=?`, goalID)
	return err
}

func (d *DB) InsertPlanSession(s PlanSession) error {
	_, err := d.sql.Exec(`
		INSERT INTO plan_sessions (
			goal_id, date, week_number, day_of_week, session_type, sport_type,
			description, distance_km, duration_min, pace_target, hr_zone,
			notes, nutrition_pre, nutrition_during, nutrition_post, completed, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0,?)
	`, s.GoalID, s.Date, s.WeekNumber, s.DayOfWeek, s.SessionType, s.SportType,
		s.Description, s.DistanceKm, s.DurationMin, s.PaceTarget, s.HRZone,
		s.Notes, s.NutritionPre, s.NutritionDuring, s.NutritionPost, time.Now().UTC())
	return err
}

func (d *DB) GetPlanSessions(goalID int64) ([]PlanSession, error) {
	rows, err := d.sql.Query(`
		SELECT id, goal_id, date, week_number, day_of_week, session_type, sport_type,
			description, distance_km, duration_min, pace_target, hr_zone, notes,
			COALESCE(nutrition_pre,''), COALESCE(nutrition_during,''), COALESCE(nutrition_post,''),
			completed,
			COALESCE(activity_strava_id,0), COALESCE(actual_distance_km,0),
			COALESCE(actual_duration_min,0), COALESCE(actual_avg_hr,0),
			COALESCE(actual_pace,''), COALESCE(analysis,''), COALESCE(performance_score,0)
		FROM plan_sessions WHERE goal_id=? ORDER BY date ASC
	`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []PlanSession
	for rows.Next() {
		var s PlanSession
		var completed int
		if err := rows.Scan(
			&s.ID, &s.GoalID, &s.Date, &s.WeekNumber, &s.DayOfWeek,
			&s.SessionType, &s.SportType, &s.Description, &s.DistanceKm,
			&s.DurationMin, &s.PaceTarget, &s.HRZone, &s.Notes,
			&s.NutritionPre, &s.NutritionDuring, &s.NutritionPost,
			&completed,
			&s.ActivityStravaID, &s.ActualDistanceKm, &s.ActualDurationMin,
			&s.ActualAvgHR, &s.ActualPace, &s.Analysis, &s.PerformanceScore,
		); err != nil {
			return nil, err
		}
		s.Completed = completed == 1
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (d *DB) PlanDateRange(goalID int64) (minDate, maxDate string, err error) {
	err = d.sql.QueryRow(
		`SELECT COALESCE(MIN(date),''), COALESCE(MAX(date),'') FROM plan_sessions WHERE goal_id=?`,
		goalID,
	).Scan(&minDate, &maxDate)
	return
}

func (d *DB) QueryActivitiesInRange(sportType, after, before string, limit int) ([]string, error) {
	query := `SELECT data FROM activities WHERE 1=1`
	args := []interface{}{}

	if sportType != "" {
		query += ` AND (sport_type=? OR type=?)`
		args = append(args, sportType, sportType)
	}
	if after != "" {
		query += ` AND start_date_local >= ?`
		args = append(args, after)
	}
	if before != "" {
		// add one day to include the last day entirely
		query += ` AND start_date_local <= ?`
		args = append(args, before+"T23:59:59Z")
	}
	query += ` ORDER BY start_date_local DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		results = append(results, data)
	}
	return results, rows.Err()
}

func (d *DB) ToggleSessionComplete(sessionID int64, completed bool) error {
	v := 0
	if completed {
		v = 1
	}
	_, err := d.sql.Exec(`UPDATE plan_sessions SET completed=? WHERE id=?`, v, sessionID)
	return err
}

func (d *DB) GoalProgress(goalID int64) (total, done int, err error) {
	err = d.sql.QueryRow(
		`SELECT COUNT(*), SUM(CASE WHEN completed=1 THEN 1 ELSE 0 END) FROM plan_sessions WHERE goal_id=? AND session_type != 'Rest'`,
		goalID,
	).Scan(&total, &done)
	return
}

// ActualPerformance holds the real data from a completed activity.
type ActualPerformance struct {
	StravaID    int64
	DistanceKm  float64
	DurationMin int
	AvgHR       float64
	ActualPace  string
	Analysis    string
	Score       float64
}

// FindSessionForActivity busca sessão de plano que corresponda ao tipo de esporte e data (±1 dia), não ainda concluída via atividade.
func (d *DB) FindSessionForActivity(sportType, actDate string) (*PlanSession, error) {
	row := d.sql.QueryRow(`
		SELECT id, goal_id, date, week_number, day_of_week, session_type, sport_type,
			description, distance_km, duration_min, pace_target, hr_zone, notes,
			COALESCE(nutrition_pre,''), COALESCE(nutrition_during,''), COALESCE(nutrition_post,''),
			completed,
			COALESCE(activity_strava_id,0), COALESCE(actual_distance_km,0),
			COALESCE(actual_duration_min,0), COALESCE(actual_avg_hr,0),
			COALESCE(actual_pace,''), COALESCE(analysis,''), COALESCE(performance_score,0)
		FROM plan_sessions
		WHERE (sport_type=? OR sport_type='')
		  AND date BETWEEN date(?,'-1 day') AND date(?,'+1 day')
		  AND activity_strava_id=0
		  AND session_type != 'Rest'
		ORDER BY ABS(julianday(date) - julianday(?)) ASC
		LIMIT 1
	`, sportType, actDate, actDate, actDate)

	var s PlanSession
	var completed int
	if err := row.Scan(
		&s.ID, &s.GoalID, &s.Date, &s.WeekNumber, &s.DayOfWeek,
		&s.SessionType, &s.SportType, &s.Description, &s.DistanceKm,
		&s.DurationMin, &s.PaceTarget, &s.HRZone, &s.Notes,
		&s.NutritionPre, &s.NutritionDuring, &s.NutritionPost,
		&completed,
		&s.ActivityStravaID, &s.ActualDistanceKm, &s.ActualDurationMin,
		&s.ActualAvgHR, &s.ActualPace, &s.Analysis, &s.PerformanceScore,
	); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	s.Completed = completed == 1
	return &s, nil
}

// ConflictInfo represents a scheduling conflict between two goals on the same day.
type ConflictInfo struct {
	Date         string `json:"date"`
	GoalID1      int64  `json:"goal_id_1"`
	GoalName1    string `json:"goal_name_1"`
	SessionID1   int64  `json:"session_id_1"`
	SessionType1 string `json:"session_type_1"`
	PaceTarget1  string `json:"pace_target_1"`
	GoalID2      int64  `json:"goal_id_2"`
	GoalName2    string `json:"goal_name_2"`
	SessionID2   int64  `json:"session_id_2"`
	SessionType2 string `json:"session_type_2"`
	PaceTarget2  string `json:"pace_target_2"`
}

// DetectConflicts returns sessions from different goals scheduled on the same day where
// at least one session is a quality session (Tempo, Interval, Race).
func (d *DB) DetectConflicts() ([]ConflictInfo, error) {
	rows, err := d.sql.Query(`
		SELECT ps1.date,
		       g1.id, g1.name, ps1.id, ps1.session_type, COALESCE(ps1.pace_target,''),
		       g2.id, g2.name, ps2.id, ps2.session_type, COALESCE(ps2.pace_target,'')
		FROM plan_sessions ps1
		JOIN plan_sessions ps2 ON ps1.date = ps2.date AND ps1.goal_id < ps2.goal_id
		JOIN goals g1 ON g1.id = ps1.goal_id
		JOIN goals g2 ON g2.id = ps2.goal_id
		WHERE ps1.session_type != 'Rest' AND ps2.session_type != 'Rest'
		  AND ps1.sport_type = ps2.sport_type
		  AND (ps1.session_type IN ('Tempo','Interval','Race')
		       OR ps2.session_type IN ('Tempo','Interval','Race'))
		ORDER BY ps1.date
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conflicts []ConflictInfo
	for rows.Next() {
		var c ConflictInfo
		if err := rows.Scan(
			&c.Date,
			&c.GoalID1, &c.GoalName1, &c.SessionID1, &c.SessionType1, &c.PaceTarget1,
			&c.GoalID2, &c.GoalName2, &c.SessionID2, &c.SessionType2, &c.PaceTarget2,
		); err != nil {
			return nil, err
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, nil
}

// MarkSessionWithActivity marca a sessão como concluída, salva dados reais e análise.
func (d *DB) MarkSessionWithActivity(sessionID int64, p ActualPerformance) error {
	_, err := d.sql.Exec(`
		UPDATE plan_sessions SET
			completed=1,
			activity_strava_id=?,
			actual_distance_km=?,
			actual_duration_min=?,
			actual_avg_hr=?,
			actual_pace=?,
			analysis=?,
			performance_score=?
		WHERE id=?
	`, p.StravaID, p.DistanceKm, p.DurationMin, p.AvgHR,
		p.ActualPace, p.Analysis, p.Score, sessionID)
	return err
}
