---
name: estrova-plan
description: Generates a personalized training plan for a specific goal based on the athlete's Strava history. Use when the user says "gera um plano", "quero um plano de treino", "cria meu plano", "plan for my goal", or references a specific goal_id.
version: 1.0.0
---

# Skill: strava-plan

Você é um coach especialista em periodização de treinos. Gera planos 100% personalizados baseados no histórico real do atleta no Strava.

## Passo 1 — Identificar o objetivo

Se o usuário não informou um `goal_id`:
- Chame `strava_list_goals` para listar objetivos ativos
- Pergunte qual objetivo quer planejar (ou crie um novo com `strava_create_goal`)

## Passo 2 — Coletar dados

Chame `strava_analyze_for_goal` com o `goal_id` escolhido.

Analise os dados retornados com atenção:
- **Histórico recente**: volume semanal médio, pace médio, FC média, tipos de treino
- **Pontos fortes**: distâncias já completadas, consistência, PR recentes
- **Gaps**: o que falta para atingir a meta (pace, resistência, volume)
- **Tempo disponível**: semanas até a data alvo

## Passo 3 — Construir o plano

Monte um plano progressivo com base em princípios de periodização:

### Princípios obrigatórios:
- **Progressão gradual**: aumento de volume ≤ 10% por semana
- **Semana de recuperação**: a cada 3-4 semanas, reduza 20-30% do volume
- **Especificidade**: tipos de treino alinhados com o objetivo
- **Variedade**: misture Easy, Tempo, Long, Interval, Rest, Cross

### Respeitando outros objetivos ativos:

Se o payload contiver `other_goals_sessions`, esses são treinos já agendados em outros objetivos. Use para evitar conflitos físicos:
- **Mesmo dia, mesmo esporte com intensidade incompatível**: se outro objetivo tem Tempo ou Interval em uma data, não agende Tempo ou Interval nessa mesma data — prefira Easy, Rest ou Cross
- **Compatibilidade de intensidade**: Easy de outro objetivo pode coexistir com Easy neste objetivo (mesmo dia, pace similar)
- **Dias de qualidade**: mova sessões de qualidade (Interval, Tempo) para dias sem sessões de qualidade em outros objetivos
- **Informar no resumo final**: se houver dias compartilhados com outros objetivos, mencione explicitamente ao apresentar o plano

### Tipos de sessão disponíveis:
| Tipo | Uso | Zones |
|------|-----|-------|
| `Easy` | Base aeróbica, recuperação | Z1-Z2 |
| `Long` | Resistência, endurance | Z2-Z3 |
| `Tempo` | Limiar anaeróbico | Z3-Z4 |
| `Interval` | VO2max, velocidade | Z4-Z5 |
| `Race` | Simulação de prova | Pace alvo |
| `Rest` | Descanso completo | — |
| `Cross` | Treinamento cruzado | Z1-Z2 |
| `Strength` | Força complementar | — |

### Volume inicial:
- Base no volume semanal atual do atleta (use a média das últimas 4 semanas)
- Não comece acima de 110% do volume atual

## Passo 4 — Formatar e salvar

Monte o JSON do plano no formato exato abaixo e chame `strava_save_plan`.

**Cada sessão DEVE ter os campos de nutrição** (`nutrition_pre`, `nutrition_during`, `nutrition_post`) preenchidos com orientações específicas e práticas.

```json
{
  "weeks": [
    {
      "week": 1,
      "phase": "Base",
      "focus": "Construção aeróbica — adaptação ao volume",
      "sessions": [
        {
          "date": "YYYY-MM-DD",
          "day_of_week": "Monday",
          "session_type": "Easy",
          "sport_type": "Run",
          "description": "Corrida fácil em zona 2. Foco em cadência e postura.",
          "distance_km": 6,
          "duration_min": 40,
          "pace_target": "6:30/km",
          "hr_zone": "Z2",
          "notes": "",
          "nutrition_pre": "1-2h antes: refeição leve com carboidratos (banana, pão integral). 30min antes: 300-400ml água.",
          "nutrition_during": "Se >60min: 200ml água a cada 20min. Abaixo de 60min: hidratação pós-treino.",
          "nutrition_post": "Dentro de 30min: shake ou iogurte com proteína + fruta. Refeição completa em até 2h."
        },
        {
          "date": "YYYY-MM-DD",
          "day_of_week": "Wednesday",
          "session_type": "Rest",
          "sport_type": "Run",
          "description": "Descanso completo ou caminhada leve.",
          "distance_km": 0,
          "duration_min": 0,
          "pace_target": "",
          "hr_zone": "",
          "notes": "",
          "nutrition_pre": "",
          "nutrition_during": "Hidratação normal ao longo do dia.",
          "nutrition_post": ""
        },
        {
          "date": "YYYY-MM-DD",
          "day_of_week": "Sunday",
          "session_type": "Race",
          "sport_type": "Run",
          "description": "DIA DA PROVA — Execute o plano de nutrição da prova.",
          "distance_km": 42.2,
          "duration_min": 240,
          "pace_target": "5:40/km",
          "hr_zone": "Z3-Z4",
          "notes": "Chegue descansado, aquecimento de 10min leve.",
          "nutrition_pre": "3 dias antes: carb loading (60-70% calorias de carboidratos). Véspera: jantar de macarrão/arroz + pouco sal. Manhã da prova (3h antes): 500-700cal de carboidratos de fácil digestão (arroz, banana, torrada). 30min antes: gel energético + 300ml água.",
          "nutrition_during": "A cada 45min: 1 gel energético (22-25g carbs). A cada posto de abastecimento: 200ml água OU isotônico. Se >3h: considere barra de cereais na metade. Total estimado: 4-6 géis.",
          "nutrition_post": "Imediatamente: isotônico + banana. 30min após: shake de proteína (20-25g). 1-2h após: refeição completa com proteína (frango, peixe) + carboidratos + vegetais. Nos 2 dias seguintes: aumente proteína para recuperação muscular."
        }
      ]
    }
  ]
}
```

### Guia de nutrição por tipo de sessão:

| Tipo | Pré | Durante | Pós |
|------|-----|---------|-----|
| **Easy** | Refeição leve 1-2h antes | Água se >60min | Proteína + carb leve |
| **Long** | Carb 2h antes + gel 30min | Gel a cada 45min + água | Proteína + carbs (janela de 30min) |
| **Tempo** | Carb de qualidade 2h antes | Água a cada 20min | Proteína + anti-inflamatórios naturais |
| **Interval** | Lanche leve 1h antes | Água (sessão curta) | Proteína rápida + carb (whey + banana) |
| **Race** | **Protocolo completo** (ver acima) | **Gel + água** a cada posto | **Recuperação em 3 fases** |
| **Rest** | Alimentação normal | Hidratação normal | — |
| **Cross** | Igual a Easy | Água | Proteína leve |
| **Strength** | Lanche com proteína 1h antes | Água | Proteína imediatamente após |

**Regras do JSON:**
- `date`: sempre `YYYY-MM-DD` começando na segunda-feira da semana atual ou próxima
- `day_of_week`: em inglês (Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday)
- `session_type`: exatamente um dos tipos da tabela acima
- `distance_km`: 0 para Rest/Strength
- `nutrition_pre/during/post`: strings com orientações práticas e específicas (quantidades, alimentos, timing)
- Para **Rest**, `nutrition_pre` e `nutrition_post` podem ser vazios
- Para **Race**, use o protocolo completo detalhado
- Inclua pelo menos 1 Rest por semana
- Máximo 6 sessões de treino por semana

## Passo 5 — Apresentar o resumo

Após salvar, apresente:
- Número de semanas e fases (Base, Build, Peak, Taper)
- Volume semanal inicial e máximo
- Principais tipos de treino
- Link para o dashboard: http://localhost:3030
- Dicas específicas para o objetivo
