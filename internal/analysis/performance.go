package analysis

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/booscaaa/estrova/internal/db"
)

type Result struct {
	Score   float64
	Summary string
	Details []string
}

func Analyze(session db.PlanSession, act db.ActivityRow) Result {
	var details []string
	score := 100.0

	// ── Distância ─────────────────────────────────────────────────────────────
	if session.DistanceKm > 0 && act.Distance > 0 {
		actualKm := act.Distance / 1000
		pct := (actualKm - session.DistanceKm) / session.DistanceKm * 100
		switch {
		case pct >= -8 && pct <= 12:
			details = append(details, fmt.Sprintf("✅ Distância: %.1fkm (meta %.1fkm) — dentro do planejado.", actualKm, session.DistanceKm))
		case pct < -8:
			score -= 20
			details = append(details, fmt.Sprintf("⚠️ Distância: %.1fkm ficou abaixo da meta de %.1fkm (%.0f%% do planejado). Tente completar o volume nos próximos treinos.", actualKm, session.DistanceKm, 100+pct))
		default:
			details = append(details, fmt.Sprintf("💪 Distância: %.1fkm superou a meta de %.1fkm (+%.0f%%). Atenção ao overtraining cumulativo.", actualKm, session.DistanceKm, pct))
		}
	}

	// ── Pace ──────────────────────────────────────────────────────────────────
	if act.AverageSpeed > 0 && session.PaceTarget != "" {
		actualSec := 1000.0 / act.AverageSpeed
		targetSec := parsePaceSec(session.PaceTarget)
		actualStr := formatPace(act.AverageSpeed)
		if targetSec > 0 {
			diff := (actualSec - float64(targetSec)) / float64(targetSec) * 100
			switch {
			case math.Abs(diff) <= 7:
				details = append(details, fmt.Sprintf("✅ Pace: %s (meta %s) — bom controle de ritmo.", actualStr, session.PaceTarget))
			case diff > 7: // mais lento
				score -= 15
				details = append(details, fmt.Sprintf("⚠️ Pace: %s ficou mais lento que a meta %s. Inclua treinos de fartlek e tempo run para desenvolver velocidade.", actualStr, session.PaceTarget))
			default: // mais rápido
				if session.SessionType == "Easy" || session.SessionType == "Long" {
					score -= 5
					details = append(details, fmt.Sprintf("⚡ Pace: %s foi mais rápido que a meta %s. Em treinos fáceis/longos, reduza a intensidade para não comprometer a recuperação.", actualStr, session.PaceTarget))
				} else {
					details = append(details, fmt.Sprintf("💪 Pace: %s superou a meta %s — excelente desempenho!", actualStr, session.PaceTarget))
				}
			}
		} else if act.AverageSpeed > 0 {
			details = append(details, fmt.Sprintf("ℹ️ Pace realizado: %s", actualStr))
		}
	}

	// ── Duração ───────────────────────────────────────────────────────────────
	if session.DurationMin > 0 && act.MovingTime > 0 {
		actualMin := act.MovingTime / 60
		diff := float64(actualMin-session.DurationMin) / float64(session.DurationMin) * 100
		switch {
		case math.Abs(diff) <= 12:
			details = append(details, fmt.Sprintf("✅ Duração: %dmin (meta %dmin) — dentro do planejado.", actualMin, session.DurationMin))
		case diff < -12:
			score -= 10
			details = append(details, fmt.Sprintf("⚠️ Duração: %dmin abaixo dos %dmin planejados. Complete o tempo de treino para atingir o estímulo aeróbico necessário.", actualMin, session.DurationMin))
		default:
			details = append(details, fmt.Sprintf("ℹ️ Duração: %dmin (meta %dmin). Volume extra OK se a intensidade estava controlada.", actualMin, session.DurationMin))
		}
	}

	// ── FC ────────────────────────────────────────────────────────────────────
	if act.AverageHR > 0 {
		hrStr := fmt.Sprintf("%.0f bpm", act.AverageHR)
		if act.MaxHR > 0 {
			hrStr += fmt.Sprintf(" (máx %.0f bpm)", act.MaxHR)
		}
		if session.HRZone != "" {
			details = append(details, fmt.Sprintf("❤️ FC: %s — zona alvo %s.", hrStr, session.HRZone))
		} else {
			details = append(details, fmt.Sprintf("❤️ FC média: %s", hrStr))
		}
	}

	// ── Resumo ────────────────────────────────────────────────────────────────
	var summary string
	switch {
	case score >= 90:
		summary = "✅ Treino executado conforme o plano"
	case score >= 70:
		summary = "⚠️ Treino com pequenos desvios do plano"
	default:
		summary = "❌ Treino abaixo do esperado — revisar execução"
	}

	return Result{
		Score:   math.Max(0, score),
		Summary: summary,
		Details: details,
	}
}

func parsePaceSec(pace string) int {
	// "5:30/km" or "5:30"
	pace = strings.TrimSuffix(pace, "/km")
	pace = strings.TrimSpace(pace)
	parts := strings.Split(pace, ":")
	if len(parts) != 2 {
		return 0
	}
	m, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	s, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0
	}
	return m*60 + s
}

func formatPace(speedMS float64) string {
	if speedMS <= 0 {
		return ""
	}
	secPerKm := 1000.0 / speedMS
	m := int(secPerKm) / 60
	s := int(secPerKm) % 60
	return fmt.Sprintf("%d:%02d/km", m, s)
}
