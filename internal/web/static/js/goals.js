let conflictSessionIDs = new Set();
let planConflicts = [];
const sessionStore = {};

function openGoalFromDashboard(id) {
  pendingGoalID = id;
  showPage('goals');
}

async function loadGoals(status) {
  document.getElementById('goal-detail').classList.add('hidden');
  document.getElementById('goals-list-container').classList.remove('hidden');
  document.getElementById('goals-list-container').innerHTML = '<div class="loading">Carregando...</div>';

  const goals = await fetch(`/api/goals${status?'?status='+status:''}`).then(r=>r.json());
  allGoals = goals;

  if (pendingGoalID) {
    const g = goals.find(x => x.id === pendingGoalID);
    pendingGoalID = null;
    if (g) { openGoalDetail(g.id, g.name); return; }
  }

  if (!goals.length) {
    document.getElementById('goals-list-container').innerHTML = `
      <div class="empty">
        <div class="empty-icon">🎯</div>
        <p>Nenhum objetivo encontrado.</p>
        <button class="btn btn-primary" onclick="openCreateGoal()">+ Criar Objetivo</button>
      </div>`;
    return;
  }
  document.getElementById('goals-list-container').innerHTML = `<div class="goals-grid">${goals.map(g=>goalCardHTML(g,false)).join('')}</div>`;
}

function filterGoals(status, el) {
  document.querySelectorAll('#page-goals .tab').forEach(t=>t.classList.remove('active'));
  el.classList.add('active');
  loadGoals(status);
}

function goalCardHTML(g, clickable) {
  const pct = g.sessions_total > 0 ? Math.round(g.sessions_done/g.sessions_total*100) : 0;
  const click = clickable || true ? `onclick="openGoalDetail(${g.id},'${g.name.replace(/'/g,"\\'")}',event)"` : '';
  return `
    <div class="goal-card" ${click} id="goal-card-${g.id}">
      <div class="goal-header">
        <div>
          <div class="goal-name">${g.name}</div>
          <div class="goal-sport">${g.description||''}</div>
        </div>
        <button class="btn-icon" onclick="deleteGoal(${g.id},event)" title="Remover">🗑</button>
      </div>
      <div class="goal-target">
        ${sportBadge(g.sport_type)}
        <span class="goal-badge badge-date">🎯 ${g.target_value}</span>
        ${g.target_date ? `<span class="goal-badge badge-date">📅 ${g.target_date}</span>` : ''}
      </div>
      ${g.sessions_total > 0 ? `
        <div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div>
        <div class="progress-label"><span>${g.sessions_done}/${g.sessions_total} sessões</span><span>${pct}%</span></div>
      ` : `<div class="progress-label" style="margin-top:12px"><span class="text-muted">Nenhum plano gerado ainda</span></div>`}
    </div>`;
}

function openGoalDetail(id, name, evt) {
  if (evt && evt.target.classList.contains('btn-icon')) return;
  currentGoalID = id;
  document.getElementById('goals-list-container').classList.add('hidden');
  document.getElementById('goal-detail').classList.remove('hidden');

  const g = allGoals.find(x => x.id === id) || { name };
  document.getElementById('detail-title').textContent = g.name;
  document.getElementById('detail-description').textContent = g.description || '';

  const targetTypeLabel = { distance: 'Distância', pace: 'Ritmo', time: 'Tempo total', event: 'Evento' }[g.target_type] || g.target_type || '';
  document.getElementById('detail-badges').innerHTML = [
    g.sport_type ? sportBadge(g.sport_type) : '',
    g.target_value ? `<span class="goal-badge badge-date">🎯 ${targetTypeLabel ? targetTypeLabel + ': ' : ''}${g.target_value}</span>` : '',
    g.target_date ? `<span class="goal-badge badge-date">📅 ${g.target_date}</span>` : '',
  ].join('');

  const statusColor = { active: '#4caf7d', completed: '#5b8dd9', cancelled: '#888' }[g.status] || '#888';
  const statusLabel = { active: 'Ativo', completed: 'Concluído', cancelled: 'Cancelado' }[g.status] || g.status || '';
  document.getElementById('detail-status-badge').innerHTML = statusLabel
    ? `<span style="font-size:11px;font-weight:600;padding:4px 10px;border-radius:12px;background:${statusColor}22;color:${statusColor};border:1px solid ${statusColor}44">${statusLabel}</span>`
    : '';

  showDetailTab('plan', document.querySelector('#goal-detail .tab'));
  loadPlan(id);
}

function hideGoalDetail() {
  document.getElementById('goal-detail').classList.add('hidden');
  document.getElementById('goals-list-container').classList.remove('hidden');
}

function showDetailTab(tab, el) {
  document.querySelectorAll('#goal-detail .tab').forEach(t=>t.classList.remove('active'));
  if (el) el.classList.add('active');
  document.getElementById('detail-plan').classList.toggle('hidden', tab !== 'plan');
  document.getElementById('detail-activities').classList.toggle('hidden', tab !== 'activities');
  if (tab === 'activities') loadGoalActivities(currentGoalID);
}

async function loadPlan(goalID) {
  document.getElementById('plan-loading').classList.remove('hidden');
  document.getElementById('plan-content').innerHTML = '';

  const [data, conflicts] = await Promise.all([
    fetch(`/api/goals/${goalID}/plan`).then(r=>r.json()).catch(()=>null),
    fetch('/api/conflicts').then(r=>r.json()).catch(()=>[]),
  ]);

  planConflicts = conflicts || [];
  conflictSessionIDs = new Set();
  for (const c of planConflicts) {
    conflictSessionIDs.add(c.session_id_1);
    conflictSessionIDs.add(c.session_id_2);
  }

  document.getElementById('plan-loading').classList.add('hidden');

  if (!data || !data.weeks || !data.weeks.length) {
    document.getElementById('plan-content').innerHTML = `
      <div class="empty">
        <div class="empty-icon">📋</div>
        <p>Nenhum plano de treino gerado ainda.</p>
        <p class="text-muted" style="font-size:12px;margin-top:8px">Use o comando <strong>/estrova-plan</strong> no Claude para gerar um plano personalizado para este objetivo.</p>
      </div>`;
    return;
  }

  const { weeks, progress, goal } = data;
  const pct = progress.total > 0 ? Math.round(progress.completed/progress.total*100) : 0;

  let html = `
    <div class="card card-sm" style="margin-bottom:20px">
      <div style="display:flex;gap:24px;flex-wrap:wrap">
        <div><div class="stat-value" style="font-size:20px">${weeks.length}</div><div class="stat-label">Semanas</div></div>
        <div><div class="stat-value" style="font-size:20px">${progress.total}</div><div class="stat-label">Sessões</div></div>
        <div><div class="stat-value text-success" style="font-size:20px">${progress.completed}</div><div class="stat-label">Concluídas</div></div>
        <div style="flex:1;min-width:200px;align-self:center">
          <div class="progress-bar" style="height:10px"><div class="progress-fill" style="width:${pct}%"></div></div>
          <div class="progress-label"><span>Progresso geral</span><span>${pct}%</span></div>
        </div>
      </div>
    </div>`;

  for (const week of weeks) {
    const totalWeek = week.sessions.filter(s=>s.session_type!=='Rest').length;
    const doneWeek = week.sessions.filter(s=>s.completed&&s.session_type!=='Rest').length;
    html += `
      <div class="week-block">
        <div class="week-header" onclick="toggleWeek(this)">
          <span class="week-num">Semana ${week.week}</span>
          <span class="week-meta">${doneWeek}/${totalWeek} sessões concluídas</span>
          <span class="week-toggle">▾</span>
        </div>
        <div class="week-sessions">
          ${week.sessions.map(s => sessionRowHTML(s)).join('')}
        </div>
      </div>`;
  }

  document.getElementById('plan-content').innerHTML = html;
}

function sessionRowHTML(s) {
  sessionStore[s.id] = s;
  const icon = sessionIcon(s.session_type);
  const isRest = s.session_type === 'Rest';
  const isRace = s.session_type === 'Race';

  return `
    <div class="session-row ${s.completed?'completed':''} ${isRace?'is-race':''} ${conflictSessionIDs.has(s.id)?'has-conflict':''}" id="session-row-${s.id}"
         onclick="openSessionDrawer(${s.id})">
      <span class="session-date">${formatDate(s.date)}</span>
      <span class="session-icon">${icon}</span>
      <div class="session-info">
        <div class="session-type">
          ${s.session_type}${isRace?'<span class="race-badge">🏆 PROVA</span>':''}${isRest?'':!isRace?` · ${s.sport_type}`:''}${conflictSessionIDs.has(s.id)?'<span class="conflict-badge" title="Conflito com outro objetivo neste dia">⚠️ conflito</span>':''}
        </div>
        <div class="session-desc">${s.description||''}</div>
      </div>
      <div class="session-metrics">
        ${s.distance_km>0?`<span class="metric-pill">${s.distance_km}km</span>`:''}
        ${s.duration_min>0?`<span class="metric-pill">${s.duration_min}min</span>`:''}
        ${s.pace_target?`<span class="metric-pill pace">${s.pace_target}</span>`:''}
        ${s.hr_zone?`<span class="metric-pill zone">${s.hr_zone}</span>`:''}
      </div>
      ${!isRest ? `
        <button class="check-btn ${s.completed?'done':''}"
                onclick="event.stopPropagation(); toggleSession(${s.id},${s.completed},this)"
                title="${s.completed?'Desmarcar':'Concluir'}">
          ${s.completed?'✓':''}
        </button>` : '<span></span>'}
    </div>`;
}

function openSessionDrawer(sessionId) {
  const s = sessionStore[sessionId];
  if (!s) return;

  const icon = sessionIcon(s.session_type);
  const isRace = s.session_type === 'Race';
  const isRest = s.session_type === 'Rest';

  document.getElementById('sd-icon').textContent = icon;
  document.getElementById('sd-type').innerHTML =
    s.session_type + (isRace ? ' <span class="race-badge">🏆 PROVA</span>' : '');
  document.getElementById('sd-sub').textContent =
    `${formatDate(s.date)}  ·  Semana ${s.week_number}  ·  ${s.day_of_week}`;

  let body = '';

  const metrics = [
    s.distance_km > 0 ? `<div class="drawer-metric accent"><div class="drawer-metric-val">${s.distance_km}km</div><div class="drawer-metric-lbl">Distância</div></div>` : '',
    s.duration_min > 0 ? `<div class="drawer-metric"><div class="drawer-metric-val">${s.duration_min}min</div><div class="drawer-metric-lbl">Duração</div></div>` : '',
    s.pace_target ? `<div class="drawer-metric accent"><div class="drawer-metric-val">${s.pace_target}</div><div class="drawer-metric-lbl">Pace alvo</div></div>` : '',
    s.hr_zone ? `<div class="drawer-metric zone"><div class="drawer-metric-val">${s.hr_zone}</div><div class="drawer-metric-lbl">Zona FC</div></div>` : '',
    s.sport_type ? `<div class="drawer-metric"><div class="drawer-metric-val">${s.sport_type}</div><div class="drawer-metric-lbl">Modalidade</div></div>` : '',
  ].filter(Boolean).join('');

  if (metrics) {
    body += `<div class="drawer-section">
      <div class="drawer-section-title">📊 Métricas do Treino</div>
      <div class="drawer-metrics">${metrics}</div>
    </div>`;
  }

  if (s.description) {
    body += `<div class="drawer-section">
      <div class="drawer-section-title">📋 Descrição</div>
      <div class="drawer-desc">${s.description}</div>
    </div>`;
  }

  if (s.notes) {
    body += `<div class="drawer-section">
      <div class="drawer-section-title">📝 Observações</div>
      <div class="drawer-notes">${s.notes}</div>
    </div>`;
  }

  if (s.completed && s.analysis) {
    const lines = s.analysis.split('\n').filter(Boolean);
    const summary = lines[0] || '';
    const details = lines.slice(1);

    const scoreColor = s.performance_score >= 90 ? 'var(--success)'
      : s.performance_score >= 70 ? 'var(--warning)' : 'var(--danger)';

    body += `<div class="drawer-section">
      <div class="drawer-section-title">📈 Análise de Desempenho</div>
      <div style="background:var(--surface2);border-radius:var(--radius-sm);padding:14px;margin-bottom:10px;display:flex;align-items:center;gap:12px">
        <div style="font-size:28px;font-weight:700;color:${scoreColor}">${Math.round(s.performance_score)}</div>
        <div>
          <div style="font-size:13px;font-weight:600">${summary}</div>
          <div style="font-size:11px;color:var(--muted);margin-top:2px">Pontuação de aderência ao plano</div>
        </div>
      </div>
      ${details.map(d => `<div style="font-size:13px;padding:6px 0;border-bottom:1px solid var(--border);line-height:1.5">${d}</div>`).join('')}
    </div>`;
  }

  if (s.completed && (s.actual_distance_km > 0 || s.actual_duration_min > 0)) {
    const rows = [
      s.distance_km > 0 && s.actual_distance_km > 0 ?
        `<tr><td>Distância</td><td>${s.distance_km}km</td><td>${s.actual_distance_km.toFixed(1)}km</td></tr>` : '',
      s.duration_min > 0 && s.actual_duration_min > 0 ?
        `<tr><td>Duração</td><td>${s.duration_min}min</td><td>${s.actual_duration_min}min</td></tr>` : '',
      s.pace_target && s.actual_pace ?
        `<tr><td>Pace</td><td>${s.pace_target}</td><td>${s.actual_pace}</td></tr>` : '',
      s.actual_avg_hr > 0 ?
        `<tr><td>FC Média</td><td>${s.hr_zone || '—'}</td><td>${Math.round(s.actual_avg_hr)} bpm</td></tr>` : '',
    ].filter(Boolean).join('');

    if (rows) {
      body += `<div class="drawer-section">
        <div class="drawer-section-title">🎯 Planejado vs Realizado</div>
        <table style="width:100%;border-collapse:collapse;font-size:13px">
          <thead>
            <tr style="color:var(--muted);font-size:11px;text-transform:uppercase">
              <th style="text-align:left;padding:6px 0">Métrica</th>
              <th style="text-align:left;padding:6px 0">Planejado</th>
              <th style="text-align:left;padding:6px 0;color:var(--accent)">Realizado</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>`;
    }
  }

  const hasNutr = s.nutrition_pre || s.nutrition_during || s.nutrition_post;
  if (hasNutr && !isRest) {
    const nutrClass = isRace ? 'race-nutr' : '';
    body += `<div class="drawer-section">
      <div class="drawer-section-title">${isRace ? '🏆 Nutrição da Prova' : '🥦 Plano Nutricional'}</div>
      ${s.nutrition_pre ? `<div class="nutr-card ${nutrClass}">
        <div class="nutr-label">🍽 Pré-treino</div>
        <div class="nutr-text">${s.nutrition_pre}</div>
      </div>` : ''}
      ${s.nutrition_during ? `<div class="nutr-card ${nutrClass}">
        <div class="nutr-label">⚡ Durante</div>
        <div class="nutr-text">${s.nutrition_during}</div>
      </div>` : ''}
      ${s.nutrition_post ? `<div class="nutr-card ${nutrClass}">
        <div class="nutr-label">🥗 Pós-treino</div>
        <div class="nutr-text">${s.nutrition_post}</div>
      </div>` : ''}
    </div>`;
  } else if (!hasNutr && !isRest) {
    body += `<div class="drawer-section">
      <div class="drawer-section-title">🥦 Nutrição</div>
      <div class="nutr-text" style="color:var(--muted);font-size:12px">Nenhuma orientação nutricional cadastrada para esta sessão. Regenere o plano com /estrova-plan para incluir nutrição.</div>
    </div>`;
  }

  const myConflicts = planConflicts.filter(c => c.session_id_1 === s.id || c.session_id_2 === s.id);
  if (myConflicts.length > 0) {
    body += `<div class="drawer-section">
      <div class="drawer-section-title" style="color:#ff8c42">⚠️ Conflitos com outros objetivos</div>
      <div class="conflict-section">`;
    for (const c of myConflicts) {
      const isOther1 = c.session_id_1 !== s.id;
      const otherId   = isOther1 ? c.session_id_1 : c.session_id_2;
      const otherGoal = isOther1 ? c.goal_name_1  : c.goal_name_2;
      const otherType = isOther1 ? c.session_type_1 : c.session_type_2;
      const otherPace = isOther1 ? c.pace_target_1  : c.pace_target_2;
      const typeIcon = {Easy:'🟢',Long:'🔵',Tempo:'🟠',Interval:'🔴',Race:'🏆',Rest:'⬜',Cross:'🟣',Strength:'⚫'}[otherType]||'⚪';
      body += `
        <div class="conflict-item">
          <div class="conflict-item-info">
            <div class="conflict-item-goal">📌 ${otherGoal}</div>
            <div class="conflict-item-type">${typeIcon} ${otherType}</div>
            <div class="conflict-item-meta">${c.date}${otherPace ? ' · '+otherPace : ''}</div>
          </div>
          <div class="conflict-item-actions">
            <button class="btn-conflict-edit" onclick="openEditConflictSession(${otherId},'${otherGoal.replace(/'/g,"\\'")}')">✏️ Editar</button>
            <button class="btn-conflict-del" onclick="deleteConflictSession(${otherId}, this)">🗑 Excluir</button>
          </div>
        </div>`;
    }
    body += `</div></div>`;
  }

  document.getElementById('sd-body').innerHTML = body;

  if (!isRest) {
    document.getElementById('sd-footer').innerHTML = `
      <button class="btn ${s.completed?'btn-secondary':'btn-primary'}" style="flex:1"
              id="drawer-check-btn"
              onclick="toggleSessionFromDrawer(${s.id}, ${s.completed})">
        ${s.completed ? '↩ Marcar como pendente' : '✓ Marcar como concluído'}
      </button>`;
  } else {
    document.getElementById('sd-footer').innerHTML = '';
  }

  document.getElementById('session-drawer').classList.add('open');
  document.getElementById('drawer-overlay').classList.add('open');
  document.body.style.overflow = 'hidden';
}

function closeDrawer() {
  document.getElementById('session-drawer').classList.remove('open');
  document.getElementById('drawer-overlay').classList.remove('open');
  document.body.style.overflow = '';
}

async function toggleSessionFromDrawer(id, wasCompleted) {
  const newState = !wasCompleted;
  await fetch(`/api/sessions/${id}/complete`, {
    method: 'PUT',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({completed: newState})
  });
  if (sessionStore[id]) sessionStore[id].completed = newState;
  const row = document.getElementById(`session-row-${id}`);
  if (row) {
    row.classList.toggle('completed', newState);
    const btn = row.querySelector('.check-btn');
    if (btn) {
      btn.classList.toggle('done', newState);
      btn.textContent = newState ? '✓' : '';
      btn.setAttribute('onclick', `event.stopPropagation(); toggleSession(${id},${newState},this)`);
    }
  }
  const drawerBtn = document.getElementById('drawer-check-btn');
  if (drawerBtn) {
    drawerBtn.textContent = newState ? '↩ Marcar como pendente' : '✓ Marcar como concluído';
    drawerBtn.className = `btn ${newState?'btn-secondary':'btn-primary'}`;
    drawerBtn.setAttribute('onclick', `toggleSessionFromDrawer(${id},${newState})`);
  }
  toast(newState ? 'Sessão concluída! ✓' : 'Sessão desmarcada');
}

function toggleWeek(header) {
  const sessions = header.nextElementSibling;
  const toggle = header.querySelector('.week-toggle');
  sessions.style.display = sessions.style.display === 'none' ? '' : 'none';
  toggle.textContent = sessions.style.display === 'none' ? '▸' : '▾';
}

async function toggleSession(id, wasCompleted, btn) {
  const newState = !wasCompleted;
  await fetch(`/api/sessions/${id}/complete`, {
    method: 'PUT',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({completed: newState})
  });
  btn.classList.toggle('done', newState);
  btn.textContent = newState ? '✓' : '';
  btn.setAttribute('onclick', `toggleSession(${id},${newState},this)`);
  const row = document.getElementById(`session-row-${id}`);
  row.classList.toggle('completed', newState);
  toast(newState ? 'Sessão concluída! ✓' : 'Sessão desmarcada');
}

function openCreateGoal() { openModal('modal-goal'); }

async function createGoal() {
  const name = document.getElementById('g-name').value.trim();
  const sport = document.getElementById('g-sport').value;
  const ttype = document.getElementById('g-target-type').value;
  const tval = document.getElementById('g-target-val').value.trim();
  const date = document.getElementById('g-date').value;
  const desc = document.getElementById('g-desc').value.trim();

  if (!name || !tval) { toast('Preencha nome e valor alvo', 'error'); return; }

  const goal = await fetch('/api/goals', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name, description:desc, sport_type:sport, target_type:ttype, target_value:tval, target_date:date})
  }).then(r=>r.json());

  closeModal('modal-goal');
  toast(`Objetivo "${goal.name}" criado!`);
  loadDashboard();
  ['g-name','g-desc','g-target-val','g-date'].forEach(id => document.getElementById(id).value='');
}

async function deleteGoal(id, evt) {
  evt.stopPropagation();
  if (!confirm('Remover este objetivo e seu plano?')) return;
  await fetch(`/api/goals/${id}`, {method:'DELETE'});
  toast('Objetivo removido');
  loadDashboard();
  loadGoals('active');
}
