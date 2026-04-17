function disposeCharts() {
  [chartVolume, chartSportDist, chartPace, chartHR].forEach(c => { if (c) c.dispose(); });
  chartVolume = chartSportDist = chartPace = chartHR = null;
}

async function loadDashboard() {
  disposeCharts();
  document.getElementById('kpi-grid').innerHTML = '<div class="loading" style="grid-column:1/-1">Carregando...</div>';
  document.getElementById('dashboard-goals').innerHTML = '<div class="loading" style="grid-column:1/-1">Carregando...</div>';
  document.getElementById('insights-grid').innerHTML = '<div class="loading" style="padding:20px">Analisando...</div>';

  const data = await fetch('/api/dashboard').then(r=>r.json()).catch(()=>null);
  if (!data) return;

  allGoals = (data.goals || []).map(g => g);
  const badge = document.getElementById('goals-badge');
  if (data.active_goals > 0) { badge.textContent = data.active_goals; badge.style.display=''; }

  const alertEl = document.getElementById('dash-conflict-alert');
  if (data.conflicts_count > 0) {
    document.getElementById('dash-conflict-msg').textContent = `${data.conflicts_count} conflito(s) detectado(s) entre objetivos — clique para resolver`;
    alertEl.classList.remove('hidden');
  } else {
    alertEl.classList.add('hidden');
  }

  document.getElementById('dash-date-sub').textContent =
    `Atualizado em ${new Date().toLocaleDateString('pt-BR',{weekday:'long',day:'numeric',month:'long'})}`;

  renderKPIs(data);
  renderDashboardGoals(data.goals || []);
  renderInsights(data);

  requestAnimationFrame(() => {
    renderVolumeChart(data.weekly_volume || []);
    renderSportDistChart(data.summary_by_sport || []);
    renderPaceChart(data.pace_trend || []);
    renderHRChart(data.pace_trend || []);
  });
}

function renderKPIs(data) {
  const sports = data.summary_by_sport || [];
  const weekVol = data.weekly_volume || [];
  const goals = data.goals || [];

  const totalKm = sports.reduce((s,x) => s + (x.total_km||0), 0).toFixed(0);
  const totalHours = sports.reduce((s,x) => s + (x.total_hours||0), 0);

  const weeksAgg = {};
  weekVol.forEach(r => { weeksAgg[r.week] = (weeksAgg[r.week]||0) + r.km; });
  const sortedWeeks = Object.keys(weeksAgg).sort();
  const lastWeekKm = sortedWeeks.length >= 1 ? weeksAgg[sortedWeeks[sortedWeeks.length-1]] : 0;
  const prevWeekKm = sortedWeeks.length >= 2 ? weeksAgg[sortedWeeks[sortedWeeks.length-2]] : 0;
  const weekDiff = prevWeekKm > 0 ? Math.round((lastWeekKm - prevWeekKm) / prevWeekKm * 100) : 0;
  const weekTrendClass = weekDiff > 0 ? 'up' : weekDiff < 0 ? 'down' : 'neutral';
  const weekTrendText = weekDiff > 0 ? `▲ ${weekDiff}% vs semana anterior` : weekDiff < 0 ? `▼ ${Math.abs(weekDiff)}% vs semana anterior` : '— igual à semana anterior';

  const totalSessions = goals.reduce((s,g) => s + (g.sessions_total||0), 0);
  const doneSessions  = goals.reduce((s,g) => s + (g.sessions_done||0), 0);
  const consistency   = totalSessions > 0 ? Math.round(doneSessions/totalSessions*100) : 0;

  let nextRaceText = '—';
  goals.forEach(g => {
    if (g.target_date) {
      const days = Math.ceil((new Date(g.target_date) - new Date()) / 86400000);
      if (days > 0) nextRaceText = `${days}d → ${g.name.substring(0,20)}`;
    }
  });

  document.getElementById('kpi-grid').innerHTML = `
    <div class="kpi-card" style="--kpi-color:var(--accent)">
      <div class="kpi-icon">🏃</div>
      <div class="kpi-value">${totalKm}<small style="font-size:18px;font-weight:500"> km</small></div>
      <div class="kpi-label">Distância total</div>
      <div class="kpi-sub">${data.total_activities} atividades · ${Math.round(totalHours)}h</div>
    </div>
    <div class="kpi-card" style="--kpi-color:#3498db">
      <div class="kpi-icon">📆</div>
      <div class="kpi-value">${lastWeekKm.toFixed(1)}<small style="font-size:18px;font-weight:500"> km</small></div>
      <div class="kpi-label">Esta semana</div>
      <div class="kpi-trend ${weekTrendClass}">${weekTrendText}</div>
    </div>
    <div class="kpi-card" style="--kpi-color:#2ecc71">
      <div class="kpi-icon">✅</div>
      <div class="kpi-value">${consistency}<small style="font-size:18px;font-weight:500">%</small></div>
      <div class="kpi-label">Sessões concluídas</div>
      <div class="kpi-sub">${doneSessions} de ${totalSessions} sessões planejadas</div>
    </div>
    <div class="kpi-card" style="--kpi-color:#f39c12">
      <div class="kpi-icon">🏆</div>
      <div class="kpi-value" style="font-size:22px;padding-top:4px">${nextRaceText}</div>
      <div class="kpi-label">Próximo objetivo</div>
      <div class="kpi-sub">${data.active_goals} objetivo(s) ativo(s)${data.conflicts_count ? ` · ⚠️ ${data.conflicts_count} conflito(s)`:''}</div>
    </div>`;
}

function renderDashboardGoals(goals) {
  if (!goals.length) {
    document.getElementById('dashboard-goals').innerHTML = `
      <div class="empty" style="grid-column:1/-1">
        <div class="empty-icon">🎯</div>
        <p>Nenhum objetivo ativo. Crie seu primeiro objetivo!</p>
        <button class="btn btn-primary" onclick="openCreateGoal()">+ Criar Objetivo</button>
      </div>`;
    return;
  }
  document.getElementById('dashboard-goals').innerHTML = goals.map(g => {
    const pct = g.sessions_total > 0 ? Math.round(g.sessions_done/g.sessions_total*100) : 0;
    const daysLeft = g.target_date ? Math.ceil((new Date(g.target_date)-new Date())/86400000) : null;
    const sport = g.sport_type || '';
    const icon = {Run:'🏃',Ride:'🚴',Swim:'🏊',Walk:'🚶'}[sport] || '🏋️';
    return `
    <div class="dash-goal-card" onclick="openGoalFromDashboard(${g.id})">
      <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:8px">
        <div>
          <div class="dash-goal-name">${icon} ${g.name}</div>
          <div class="dash-goal-meta">${g.target_value}${daysLeft!=null ? ` · <span style="color:${daysLeft<14?'var(--warning)':'var(--muted)'}">${daysLeft > 0 ? daysLeft+'d restantes':'Hoje!'}</span>`:''}</div>
        </div>
        <span style="font-size:11px;padding:3px 8px;border-radius:12px;background:rgba(252,76,2,.12);color:var(--accent);font-weight:700">${sport}</span>
      </div>
      ${g.description ? `<div style="font-size:12px;color:var(--muted);margin-bottom:10px">${g.description}</div>` : ''}
      <div class="dash-goal-progress">
        <div class="dash-goal-pct">${pct}%</div>
        <div class="dash-goal-bar-wrap">
          <div class="progress-bar" style="height:8px"><div class="progress-fill" style="width:${pct}%"></div></div>
          <div class="dash-goal-sessions">${g.sessions_done}/${g.sessions_total} sessões concluídas</div>
        </div>
      </div>
    </div>`;
  }).join('');
}

function renderInsights(data) {
  const insights = [];
  const sports = data.summary_by_sport || [];
  const weekVol = data.weekly_volume || [];
  const paceTrend = data.pace_trend || [];
  const goals = data.goals || [];

  const weeksAgg = {};
  weekVol.forEach(r => { weeksAgg[r.week] = (weeksAgg[r.week]||0) + r.km; });
  const sortedWeeks = Object.keys(weeksAgg).sort();
  if (sortedWeeks.length >= 2) {
    const cur = weeksAgg[sortedWeeks[sortedWeeks.length-1]];
    const prev = weeksAgg[sortedWeeks[sortedWeeks.length-2]];
    const diff = cur - prev;
    if (Math.abs(diff) > 0.5) {
      insights.push({ icon:'📈', color: diff > 0 ? 'var(--success)' : 'var(--warning)',
        title: diff > 0 ? `Volume em alta` : `Volume em baixa`,
        text: `${diff>0?'+':''}${diff.toFixed(1)}km esta semana vs semana passada (${cur.toFixed(1)}km vs ${prev.toFixed(1)}km)` });
    }
  }

  if (paceTrend.length >= 3) {
    const recent = paceTrend.slice(-3).map(p => p.avg_speed_ms);
    const avg1 = recent.slice(0,Math.floor(recent.length/2)).reduce((a,b)=>a+b,0)/(Math.floor(recent.length/2)||1);
    const avg2 = recent.slice(Math.ceil(recent.length/2)).reduce((a,b)=>a+b,0)/(recent.length - Math.ceil(recent.length/2)||1);
    const paceImproved = avg2 > avg1 * 1.01;
    const paceWorse = avg2 < avg1 * 0.99;
    if (paceImproved) {
      insights.push({ icon:'⚡', color:'var(--success)', title:'Pace melhorando',
        text:`Seu pace médio nas últimas semanas está mais rápido. Continue o trabalho de qualidade!` });
    } else if (paceWorse) {
      insights.push({ icon:'💤', color:'var(--warning)', title:'Pace mais lento',
        text:`Pace levemente mais lento recentemente — pode ser carga acumulada ou semana de recuperação.` });
    }
  }

  const hrPoints = paceTrend.filter(p => p.avg_hr > 0);
  if (hrPoints.length >= 3) {
    const recent = hrPoints.slice(-3);
    const hrAvg = recent.reduce((s,p)=>s+p.avg_hr,0)/recent.length;
    if (hrAvg > 160) {
      insights.push({ icon:'❤️', color:'var(--danger)', title:'FC elevada', text:`FC média das últimas corridas: ${hrAvg.toFixed(0)} bpm — considere treinos Z1-Z2 para recuperação.` });
    } else if (hrAvg < 140 && hrAvg > 0) {
      insights.push({ icon:'💚', color:'var(--success)', title:'FC controlada', text:`FC média ${hrAvg.toFixed(0)} bpm — boa base aeróbica. Você pode incrementar o volume.` });
    }
  }

  const totalSessions = goals.reduce((s,g) => s+(g.sessions_total||0), 0);
  const doneSessions  = goals.reduce((s,g) => s+(g.sessions_done||0), 0);
  if (totalSessions > 0) {
    const pct = Math.round(doneSessions/totalSessions*100);
    if (pct >= 80) {
      insights.push({ icon:'🔥', color:'var(--accent)', title:'Alta consistência!', text:`${pct}% das sessões planejadas concluídas — você está no caminho certo!` });
    } else if (pct < 50 && totalSessions > 4) {
      insights.push({ icon:'📋', color:'var(--warning)', title:'Plano em atraso', text:`${pct}% das sessões concluídas. Revise o plano ou use /estrova-resolve-conflicts.` });
    }
  }

  goals.forEach(g => {
    if (g.target_date) {
      const days = Math.ceil((new Date(g.target_date)-new Date())/86400000);
      if (days > 0 && days <= 14) {
        insights.push({ icon:'🏁', color:'gold', title:`${g.name} em ${days} dias!`,
          text:`Hora do tapering! Reduza 20-30% do volume, mantenha intensidade moderada.` });
      } else if (days > 14 && days <= 30) {
        insights.push({ icon:'🎯', color:'var(--info)', title:`${days} dias para ${g.name}`,
          text:`Fase de pico. Inclua simulações de pace da prova nos treinos Tempo.` });
      }
    }
  });

  if (sports.length > 0) {
    const top = sports[0];
    insights.push({ icon:{Run:'🏃',Ride:'🚴',Swim:'🏊'}[top.sport_type]||'🏋️', color:'var(--muted)',
      title:`Esporte principal: ${top.sport_type}`,
      text:`${top.total_km}km em ${top.count} atividades · ${top.total_hours}h no total` });
  }

  if (!insights.length) {
    insights.push({ icon:'🔄', color:'var(--muted)', title:'Sincronize seus treinos', text:'Use strava_sync para importar atividades e gerar insights.' });
  }

  document.getElementById('insights-grid').innerHTML = insights.map(ins => `
    <div class="insight-card" style="--insight-color:${ins.color}">
      <div class="insight-icon">${ins.icon}</div>
      <div class="insight-text"><strong>${ins.title}</strong><span>${ins.text}</span></div>
    </div>`).join('');
}

function renderVolumeChart(weekVol) {
  const el = document.getElementById('chart-volume');
  if (!el) return;
  chartVolume = echarts.init(el, null, { renderer: 'canvas' });

  const weeks = [...new Set(weekVol.map(r=>r.week))].sort();
  const sports = [...new Set(weekVol.map(r=>r.sport_type))];
  const wkLabel = w => { const [y,wn] = w.split('-W'); return `S${parseInt(wn)}`; };

  const series = sports.map(sp => ({
    name: sp,
    type: 'bar', stack: 'total', barMaxWidth: 32,
    itemStyle: { color: SPORT_COLORS[sp] || '#888', borderRadius: sp === sports[sports.length-1] ? [3,3,0,0] : 0 },
    data: weeks.map(w => { const r = weekVol.find(x=>x.week===w&&x.sport_type===sp); return r ? r.km : 0; }),
    label: { show: false }
  }));

  chartVolume.setOption({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' },
      formatter: params => {
        const w = params[0]?.axisValue || '';
        let s = `<b>${w}</b><br/>`;
        params.forEach(p => { if (p.value > 0) s += `${p.marker}${p.seriesName}: <b>${p.value}km</b><br/>`; });
        return s;
      }},
    legend: { data: sports, bottom: 0, textStyle: { color: ECHART_TEXT_COLOR, fontSize: 11 }, itemWidth: 12, itemHeight: 8 },
    grid: { top: 10, left: 40, right: 10, bottom: 36 },
    xAxis: { type: 'category', data: weeks.map(wkLabel), axisLabel: { color: ECHART_TEXT_COLOR, fontSize: 10 }, axisLine: { lineStyle: { color: ECHART_GRID_COLOR } }, splitLine: { show: false } },
    yAxis: { type: 'value', axisLabel: { color: ECHART_TEXT_COLOR, fontSize: 10, formatter: v => v+'km' }, splitLine: { lineStyle: { color: ECHART_GRID_COLOR, type: 'dashed' } } },
    series
  });
}

function renderSportDistChart(sports) {
  const el = document.getElementById('chart-sport-dist');
  if (!el) return;
  chartSportDist = echarts.init(el, null, { renderer: 'canvas' });
  const data = sports.filter(s=>s.total_km>0).map(s => ({
    name: s.sport_type, value: parseFloat(s.total_km),
    itemStyle: { color: SPORT_COLORS[s.sport_type] || '#8892b0' }
  }));
  chartSportDist.setOption({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'item', formatter: p => `${p.marker}${p.name}<br/><b>${p.value}km</b> (${p.percent}%)` },
    legend: { orient: 'vertical', right: 10, top: 'center', textStyle: { color: ECHART_TEXT_COLOR, fontSize: 11 } },
    series: [{ type: 'pie', radius: ['42%','72%'], center: ['38%','50%'],
      label: { show: false }, emphasis: { label: { show: true, fontSize: 13, fontWeight: 700, color: '#e8eaf6' } },
      data
    }]
  });
}

function speedToPace(ms) {
  if (!ms || ms <= 0) return null;
  const sPerKm = 1000 / ms;
  return sPerKm;
}

function renderPaceChart(paceTrend) {
  const el = document.getElementById('chart-pace');
  if (!el) return;
  chartPace = echarts.init(el, null, { renderer: 'canvas' });
  const valid = paceTrend.filter(p => p.avg_speed_ms > 0);
  if (!valid.length) { chartPace.setOption({ backgroundColor:'transparent', graphic:[{type:'text',left:'center',top:'middle',style:{text:'Sem dados de corrida',fill:ECHART_TEXT_COLOR}}] }); return; }
  chartPace.setOption({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis', formatter: params => {
      const p = params[0]; if (!p) return '';
      const sec = p.value; const m = Math.floor(sec/60), s = Math.floor(sec%60);
      return `<b>${p.axisValue}</b><br/>${p.marker}Pace: <b>${m}:${s<10?'0':''}${s}/km</b>`;
    }},
    grid: { top: 10, left: 52, right: 14, bottom: 28 },
    xAxis: { type: 'category', data: valid.map(p=>{ const[,wn]=p.week.split('-W'); return `S${parseInt(wn)}`; }),
      axisLabel: { color: ECHART_TEXT_COLOR, fontSize: 10 }, axisLine: { lineStyle: { color: ECHART_GRID_COLOR } }, splitLine: { show: false } },
    yAxis: { type: 'value', inverse: true, min: v => Math.floor(v.min)-10, max: v => Math.ceil(v.max)+10,
      axisLabel: { color: ECHART_TEXT_COLOR, fontSize: 10, formatter: v => { const m=Math.floor(v/60),s=Math.floor(v%60); return `${m}:${s<10?'0':''}${s}`; } },
      splitLine: { lineStyle: { color: ECHART_GRID_COLOR, type: 'dashed' } } },
    series: [{ type: 'line', smooth: true, data: valid.map(p => speedToPace(p.avg_speed_ms)),
      symbol: 'circle', symbolSize: 5,
      lineStyle: { color: 'var(--accent)', width: 2 },
      itemStyle: { color: 'var(--accent)' },
      areaStyle: { color: { type:'linear', x:0,y:0,x2:0,y2:1, colorStops:[{offset:0,color:'rgba(252,76,2,.25)'},{offset:1,color:'rgba(252,76,2,0)'}] } }
    }]
  });
}

function renderHRChart(paceTrend) {
  const el = document.getElementById('chart-hr');
  if (!el) return;
  chartHR = echarts.init(el, null, { renderer: 'canvas' });
  const valid = paceTrend.filter(p => p.avg_hr > 0);
  if (!valid.length) { chartHR.setOption({ backgroundColor:'transparent', graphic:[{type:'text',left:'center',top:'middle',style:{text:'Sem dados de FC',fill:ECHART_TEXT_COLOR}}] }); return; }
  chartHR.setOption({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis', formatter: params => {
      const p = params[0]; if (!p) return '';
      return `<b>${p.axisValue}</b><br/>${p.marker}FC média: <b>${p.value} bpm</b>`;
    }},
    grid: { top: 10, left: 46, right: 14, bottom: 28 },
    xAxis: { type: 'category', data: valid.map(p=>{ const[,wn]=p.week.split('-W'); return `S${parseInt(wn)}`; }),
      axisLabel: { color: ECHART_TEXT_COLOR, fontSize: 10 }, axisLine: { lineStyle: { color: ECHART_GRID_COLOR } }, splitLine: { show: false } },
    yAxis: { type: 'value', min: v => Math.floor(v.min)-5,
      axisLabel: { color: ECHART_TEXT_COLOR, fontSize: 10, formatter: v => v+' bpm' },
      splitLine: { lineStyle: { color: ECHART_GRID_COLOR, type: 'dashed' } } },
    series: [{ type: 'line', smooth: true, data: valid.map(p => p.avg_hr),
      symbol: 'circle', symbolSize: 5,
      lineStyle: { color: '#e74c3c', width: 2 },
      itemStyle: { color: '#e74c3c' },
      areaStyle: { color: { type:'linear', x:0,y:0,x2:0,y2:1, colorStops:[{offset:0,color:'rgba(231,76,60,.25)'},{offset:1,color:'rgba(231,76,60,0)'}] } }
    }]
  });
}

window.addEventListener('resize', () => {
  [chartVolume, chartSportDist, chartPace, chartHR].forEach(c => c?.resize());
});
