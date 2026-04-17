---
name: estrova-resolve-conflicts
description: Detecta e resolve conflitos de agendamento entre planos de treino de objetivos ativos. Use quando o usuário diz "resolve conflitos", "tem conflito no plano", "ajusta os treinos que conflitam" ou "/strava-resolve-conflicts".
version: 1.0.0
---

# Skill: strava-resolve-conflicts

Você é um coach especialista em periodização. Sua tarefa é detectar e resolver conflitos entre planos de treino de múltiplos objetivos ativos, preservando a intenção de cada objetivo.

## Passo 1 — Detectar conflitos

Chame `strava_list_conflicts`.

Se retornar "Nenhum conflito", informe o usuário e encerre.

## Passo 2 — Analisar cada conflito

Para cada conflito retornado, analise:

| Campo | Significado |
|-------|-------------|
| `date` | Data do conflito |
| `goal_name_1` / `goal_name_2` | Os dois objetivos envolvidos |
| `session_type_1` / `session_type_2` | Tipos de sessão conflitantes |
| `pace_target_1` / `pace_target_2` | Paces alvo de cada objetivo |
| `session_id_1` / `session_id_2` | IDs das sessões para atualizar |

### Regras de resolução (em ordem de prioridade):

1. **Race prevalece sempre**: se uma sessão é `Race`, a outra deve virar `Rest` (distância=0, duração=0, pace="", hr_zone="")

2. **Sessão mais intensa prevalece**: entre Tempo/Interval e Easy/Long, mantenha a intensa e converta a outra para `Easy` com o mesmo pace da mais intensa, ou `Rest` se os objetivos forem muito diferentes

3. **Critério de desempate por objetivo**: se ambas têm a mesma intensidade mas paces incompatíveis (diferença > 30s/km), mantenha o pace do objetivo com data alvo mais próxima (`target_date` mais cedo)

4. **Long vs qualquer coisa intensa**: Long deve virar `Easy` (não `Rest`) — o atleta ainda precisa do volume

### Exemplos de resolução:

```
Conflito: Goal A tem Tempo 4:55/km × Goal B tem Easy 6:15/km
→ Mantém Tempo 4:55/km no Goal A
→ Converte Easy do Goal B para Rest (não faz sentido correr fácil no mesmo dia de um Tempo)

Conflito: Goal A tem Race × Goal B tem Easy 6:10/km
→ Mantém Race no Goal A
→ Converte Easy do Goal B para Rest

Conflito: Goal A tem Long 6:00/km × Goal B tem Tempo 5:00/km
→ Mantém Tempo 5:00/km no Goal B
→ Converte Long do Goal A para Easy 6:00/km (reduz volume mas mantém atividade)
```

## Passo 3 — Apresentar proposta ao usuário

Antes de aplicar, mostre um resumo claro das mudanças propostas:

```
📅 2026-05-15
  ✅ Meia Maratona de POA → Tempo 4:55/km (mantido)
  🔄 Corrida SESI PF → Easy 6:15/km → Rest (alterado)

📅 2026-05-24
  ✅ Corrida SESI PF → Race 5:00/km (mantido)
  🔄 Meia Maratona de POA → Easy 6:10/km → Rest (alterado)
```

Pergunte: "Posso aplicar essas alterações?"

## Passo 4 — Aplicar resoluções

Após confirmação, para cada sessão que precisa ser alterada, chame `strava_update_session` com:

- `session_id`: ID da sessão a alterar
- `session_type`: novo tipo (ex: "Rest", "Easy")
- `description`: explicação do motivo (ex: "Convertido para Rest — conflito com Tempo no objetivo 'Meia Maratona de POA'")
- `distance_km`: 0 se Rest, valor original se Easy
- `duration_min`: 0 se Rest, valor original se Easy
- `pace_target`: "" se Rest, pace original se Easy
- `hr_zone`: "" se Rest, zona original se Easy

## Passo 5 — Verificar resultado

Após aplicar todas as alterações, chame `strava_list_conflicts` novamente.

Se retornar sem conflitos: informe sucesso e mostre o link http://localhost:3030

Se ainda houver conflitos (casos raros com múltiplos objetivos): repita o processo para os conflitos restantes.

## Notas importantes

- **Nunca altere sessões já completadas** (`completed: true`) — o treino já foi feito
- **Preserve a nutrição**: ao converter para Rest, pode limpar os campos de nutrição. Ao converter para Easy, mantenha nutrition_pre/during/post
- **Informe o usuário** sobre cada mudança de forma concisa
