let actsDebounceTimer = null;

function debounceActivities() {
  clearTimeout(actsDebounceTimer);
  actsDebounceTimer = setTimeout(loadActivitiesPage, 350);
}

function clearActsFilters() {
  document.getElementById('acts-search').value = '';
  document.getElementById('acts-sport').value = '';
  document.getElementById('acts-after').value = '';
  document.getElementById('acts-before').value = '';
  document.getElementById('acts-limit').value = '100';
  loadActivitiesPage();
}

async function loadActivities() { loadActivitiesPage(); }

async function loadActivitiesPage() {
  const sport  = document.getElementById('acts-sport')?.value || '';
  const after  = document.getElementById('acts-after')?.value || '';
  const before = document.getElementById('acts-before')?.value || '';
  const search = document.getElementById('acts-search')?.value || '';
  const limit  = document.getElementById('acts-limit')?.value || '100';

  document.getElementById('activities-container').innerHTML = '<div class="loading">Carregando...</div>';

  const params = new URLSearchParams();
  if (sport)  params.set('sport', sport);
  if (after)  params.set('after', after);
  if (before) params.set('before', before);
  if (search) params.set('q', search);
  params.set('limit', limit);

  const data = await fetch(`/api/activities?${params}`).then(r=>r.json()).catch(()=>null);
  if (!data) {
    document.getElementById('activities-container').innerHTML = '<div class="acts-empty">Erro ao carregar atividades.</div>';
    return;
  }

  const acts = data.activities || [];
  const sub = `${data.count} de ${data.total} atividade(s)${sport ? ' · '+sport : ''}`;
  document.getElementById('acts-count-sub').textContent = sub;

  if (!acts.length) {
    document.getElementById('activities-container').innerHTML = `
      <div class="acts-empty">
        <div style="font-size:40px;margin-bottom:12px">🏃</div>
        <p>Nenhuma atividade encontrada com os filtros aplicados.</p>
        <p style="font-size:12px;margin-top:8px;color:var(--muted)">Use strava_sync para sincronizar atividades do Strava.</p>
      </div>`;
    return;
  }

  document.getElementById('activities-container').innerHTML = renderActivitiesHTML(acts);
}

async function loadGoalActivities(goalID) {
  document.getElementById('acts-loading').classList.remove('hidden');
  document.getElementById('acts-content').innerHTML = '';

  const [acts, planData] = await Promise.all([
    fetch(`/api/goals/${goalID}/activities`).then(r=>r.json()).catch(()=>[]),
    fetch(`/api/goals/${goalID}/plan`).then(r=>r.json()).catch(()=>null),
  ]);

  document.getElementById('acts-loading').classList.add('hidden');

  let header = '';
  if (planData?.weeks?.length) {
    const sessions = planData.weeks.flatMap(w => w.sessions);
    const dates = sessions.map(s=>s.date).filter(Boolean).sort();
    const goal = planData.goal;
    header = `
      <div class="card card-sm" style="margin-bottom:16px;display:flex;gap:20px;flex-wrap:wrap;align-items:center">
        <div><span class="nutr-label">Modalidade</span><div style="font-size:13px;margin-top:2px">${sportBadge(goal.sport_type)}</div></div>
        <div><span class="nutr-label">Período do plano</span><div style="font-size:13px;color:var(--text);margin-top:2px">${formatDate(dates[0])} → ${formatDate(dates[dates.length-1])}</div></div>
        <div><span class="nutr-label">Atividades no período</span><div style="font-size:20px;font-weight:700;color:var(--accent)">${acts.length}</div></div>
      </div>`;
  }

  document.getElementById('acts-content').innerHTML = acts.length
    ? header + renderActivitiesHTML(acts)
    : header + '<div class="empty"><div class="empty-icon">🏃</div><p>Nenhuma atividade encontrada no período do plano para esta modalidade.</p><p class="text-muted" style="font-size:12px;margin-top:8px">Use strava_sync para sincronizar atividades do Strava.</p></div>';
}

function renderActivitiesHTML(acts) {
  const sportIcon = s => ({Run:'🏃',Ride:'🚴',Swim:'🏊',Walk:'🚶',WeightTraining:'🏋️',Workout:'⚡',Hike:'🥾'}[s]||'🏅');
  const sportColor = s => ({Run:'badge-run',Ride:'badge-ride',Swim:'badge-swim'}[s]||'badge-other');

  const groups = {};
  const groupOrder = [];
  for (const a of acts) {
    const d = (a.start_date_local||a.start_date||'').slice(0,10);
    const monthKey = d.slice(0,7);
    if (!groups[monthKey]) { groups[monthKey] = []; groupOrder.push(monthKey); }
    groups[monthKey].push({...a, _date: d});
  }

  let html = '<div class="activities-list">';
  for (const month of groupOrder) {
    const [y, m] = month.split('-');
    const label = new Date(parseInt(y), parseInt(m)-1, 1).toLocaleDateString('pt-BR',{month:'long',year:'numeric'});
    html += `<div class="act-group-header">${label} <span style="font-weight:400">(${groups[month].length})</span></div>`;
    for (const a of groups[month]) {
      const sp = a.sport_type || a.type || '';
      const dist = a.distance ? fmtDist(a.distance) : '—';
      const time = a.moving_time ? fmtTime(a.moving_time) : '—';
      const pace = (a.average_speed && (sp==='Run'||sp==='Walk')) ? fmtPace(a.average_speed) : '';
      const speed = (a.average_speed && sp==='Ride') ? (a.average_speed*3.6).toFixed(1)+'km/h' : '';
      const hr = a.average_heartrate ? `❤️ ${Math.round(a.average_heartrate)}` : '';
      html += `
        <div class="activity-row" onclick="openActivityDetail(${a.id})">
          <span class="activity-sport-icon">${sportIcon(sp)}</span>
          <span class="activity-meta">${formatDate(a._date)}</span>
          <span class="activity-name" title="${a.name||''}">${a.name||'—'}</span>
          <span class="metric-pill">${dist}</span>
          <span class="metric-pill">${time}</span>
          ${pace ? `<span class="metric-pill pace">${pace}</span>` : speed ? `<span class="metric-pill pace">${speed}</span>` : '<span></span>'}
          ${hr ? `<span class="metric-pill zone">${hr}</span>` : '<span></span>'}
        </div>`;
    }
  }
  html += '</div>';
  return html;
}
