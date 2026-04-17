const ACT_CHARTS = {};

function disposeActCharts() {
  Object.values(ACT_CHARTS).forEach(c => c && c.dispose());
  Object.keys(ACT_CHARTS).forEach(k => delete ACT_CHARTS[k]);
}

function closeActivityDetail() {
  disposeActCharts();
  document.querySelectorAll('[id^="page-"]').forEach(el => el.classList.add('hidden'));
  document.getElementById('page-activities').classList.remove('hidden');
  document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
  document.getElementById('nav-activities').classList.add('active');
}

async function openActivityDetail(id) {
  document.querySelectorAll('[id^="page-"]').forEach(el => el.classList.add('hidden'));
  document.getElementById('page-activity-detail').classList.remove('hidden');
  document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
  document.getElementById('nav-activities').classList.add('active');

  disposeActCharts();
  document.getElementById('act-hero-name').textContent = 'Carregando...';
  document.getElementById('act-hero-meta').innerHTML = '';
  document.getElementById('act-kpi-grid').innerHTML = '<div class="loading" style="grid-column:1/-1">Buscando dados do Estrova...</div>';
  document.getElementById('act-charts-grid').innerHTML = '';
  document.getElementById('act-tables-grid').innerHTML = '';
  document.getElementById('act-segments-wrap').innerHTML = '';
  document.getElementById('act-desc-card').classList.add('hidden');

  try {
    const detail = await fetch(`${API}/api/activities/${id}/detail`).then(r => r.json());
    if (detail.error) throw new Error(detail.error);
    renderActivityDetail(detail);
  } catch(e) {
    document.getElementById('act-kpi-grid').innerHTML = `<div style="grid-column:1/-1;padding:40px;text-align:center;color:var(--danger)">Erro ao carregar atividade: ${e.message}</div>`;
  }
}

function renderActivityDetail(d) {
  const sp = d.sport_type || d.type || '';
  const isRun  = sp==='Run'||sp==='Walk';
  const isRide = sp==='Ride';
  const sportIcon = s => ({Run:'🏃',Ride:'🚴',Swim:'🏊',Walk:'🚶',WeightTraining:'🏋️',Workout:'⚡',Hike:'🥾'}[s]||'🏅');
  const sportMainColor = s => ({Run:'#fc4c02',Ride:'#3498db',Swim:'#00bcd4',Walk:'#2ecc71'}[s]||'#fc4c02');
  const COLOR       = sportMainColor(sp);
  const COLOR_HR    = '#e74c3c';
  const COLOR_WATTS = '#9b59b6';
  const COLOR_DIST  = '#2ecc71';
  const dateStr = (d.start_date_local||d.start_date||'').slice(0,10);

  document.getElementById('act-hero-icon').textContent = sportIcon(sp);
  document.getElementById('act-hero-name').textContent = d.name || 'Atividade';
  const metaParts = [];
  if (dateStr) metaParts.push(`📅 ${formatDate(dateStr)}`);
  if (sp) metaParts.push(`🏷 ${sp}`);
  if (d.device_name) metaParts.push(`📱 ${d.device_name}`);
  if (d.pr_count > 0) metaParts.push(`🏅 ${d.pr_count} PR(s)`);
  document.getElementById('act-hero-meta').innerHTML = metaParts.map(p=>`<span>${p}</span>`).join('');

  if (d.description) {
    const dc = document.getElementById('act-desc-card');
    dc.textContent = d.description;
    dc.classList.remove('hidden');
  }

  const pace  = (d.average_speed && isRun)  ? fmtPace(d.average_speed) : null;
  const speed = (d.average_speed && isRide) ? (d.average_speed*3.6).toFixed(1)+'km/h' : null;
  const kpiDefs = [
    { label:'Distância',              val: d.distance         ? fmtDist(d.distance)                  : null, color: COLOR },
    { label:'Tempo',                  val: d.moving_time      ? fmtTime(d.moving_time)               : null, color: '#8892b0' },
    { label: isRide?'Velocidade':'Pace', val: pace||speed||null,                                             color: COLOR },
    { label:'FC Média',               val: d.average_heartrate ? `${Math.round(d.average_heartrate)} bpm` : null, color: COLOR_HR },
    { label:'FC Máx',                 val: d.max_heartrate    ? `${Math.round(d.max_heartrate)} bpm` : null, color: COLOR_HR },
    { label:'Elevação',               val: d.total_elevation_gain ? `${Math.round(d.total_elevation_gain)}m` : null, color: COLOR_DIST },
    { label:'Calorias',               val: d.calories         ? `${Math.round(d.calories)} kcal`    : null, color: '#f39c12' },
    { label:'Suffer Score',           val: d.suffer_score     ? `${Math.round(d.suffer_score)}`     : null, color: '#e74c3c' },
    { label:'Watts Médios',           val: d.average_watts    ? `${Math.round(d.average_watts)}W`   : null, color: COLOR_WATTS },
    { label:'PRs',                    val: d.pr_count > 0     ? `${d.pr_count}`                     : null, color: '#ffd700' },
    { label:'Kudos',                  val: d.kudos_count > 0  ? `${d.kudos_count}`                  : null, color: '#8892b0' },
  ].filter(k => k.val !== null);

  document.getElementById('act-kpi-grid').innerHTML = kpiDefs.map(k => `
    <div class="act-kpi-card" style="--kc:${k.color}">
      <div class="act-kpi-label">${k.label}</div>
      <div class="act-kpi-val">${k.val}</div>
    </div>`).join('');

  const laps = d.laps || [];
  const chartsGrid = document.getElementById('act-charts-grid');
  chartsGrid.innerHTML = '';

  if (laps.length > 0) {
    const labels = laps.map((l, i) => l.distance >= 900 ? `${((i+1))}km` : `V${i+1}`);
    const gridOpts = { top:28, bottom:36, left:56, right:16 };
    const xAxis = { type:'category', data:labels, axisLabel:{color:ECHART_TEXT_COLOR,fontSize:11}, axisLine:{lineStyle:{color:ECHART_GRID_COLOR}}, boundaryGap: true };

    if (laps.some(l => l.average_speed)) {
      const chartId = 'act-lap-pace';
      const title = isRide ? 'Velocidade por km' : 'Pace por km';
      chartsGrid.insertAdjacentHTML('beforeend', `
        <div class="act-chart-card">
          <div class="act-chart-title"><span class="act-chart-dot" style="background:${COLOR}"></span>${title}</div>
          <div id="${chartId}" class="act-chart-el"></div>
        </div>`);
      requestAnimationFrame(() => {
        const el = document.getElementById(chartId);
        if (!el) return;
        const c = echarts.init(el, 'dark');
        ACT_CHARTS[chartId] = c;
        if (isRide) {
          const vals = laps.map(l => l.average_speed ? parseFloat((l.average_speed*3.6).toFixed(1)) : null);
          c.setOption({
            backgroundColor:'transparent', grid:gridOpts,
            tooltip:{ trigger:'axis', formatter: p => `${p[0].axisValue}<br/>${p[0].marker}${p[0].value} km/h` },
            xAxis, yAxis:{ type:'value', axisLabel:{color:ECHART_TEXT_COLOR,fontSize:10,formatter:v=>v+'km/h'}, splitLine:{lineStyle:{color:ECHART_GRID_COLOR}} },
            series:[{ type:'bar', data:vals, itemStyle:{color:COLOR, borderRadius:[3,3,0,0]}, barMaxWidth:52 }]
          });
        } else {
          const vals = laps.map(l => l.average_speed ? parseFloat((1000/l.average_speed).toFixed(1)) : null);
          const avgPace = d.average_speed ? parseFloat((1000/d.average_speed).toFixed(1)) : null;
          c.setOption({
            backgroundColor:'transparent', grid:gridOpts,
            tooltip:{ trigger:'axis', formatter: p => {
              if (!p[0].value) return p[0].axisValue;
              const sec=p[0].value, m=Math.floor(sec/60), s=Math.floor(sec%60);
              return `${p[0].axisValue}<br/>${p[0].marker}${m}:${s<10?'0':''}${s}/km`;
            }},
            xAxis,
            yAxis:{ type:'value', inverse:true, axisLabel:{color:ECHART_TEXT_COLOR,fontSize:10,formatter:v=>{const m=Math.floor(v/60),s=Math.floor(v%60);return m+':'+(s<10?'0':'')+s;}}, splitLine:{lineStyle:{color:ECHART_GRID_COLOR}} },
            series:[
              { type:'bar', data:vals, itemStyle:{color:COLOR, borderRadius:[3,3,0,0]}, barMaxWidth:52 },
              ...(avgPace ? [{ type:'line', data:vals.map(()=>avgPace), symbol:'none', lineStyle:{color:'#fff4',width:1.5,type:'dashed'}, tooltip:{show:false} }] : [])
            ]
          });
        }
      });
    }

    if (laps.some(l => l.average_heartrate)) {
      const chartId = 'act-lap-hr';
      chartsGrid.insertAdjacentHTML('beforeend', `
        <div class="act-chart-card">
          <div class="act-chart-title"><span class="act-chart-dot" style="background:${COLOR_HR}"></span>Frequência Cardíaca por km</div>
          <div id="${chartId}" class="act-chart-el"></div>
        </div>`);
      requestAnimationFrame(() => {
        const el = document.getElementById(chartId);
        if (!el) return;
        const c = echarts.init(el, 'dark');
        ACT_CHARTS[chartId] = c;
        const avg = laps.map(l => l.average_heartrate ? Math.round(l.average_heartrate) : null);
        const max = laps.map(l => l.max_heartrate    ? Math.round(l.max_heartrate)    : null);
        const hasMax = max.some(v => v !== null);
        const avgHR = d.average_heartrate ? Math.round(d.average_heartrate) : null;
        c.setOption({
          backgroundColor:'transparent', grid:gridOpts,
          tooltip:{ trigger:'axis', formatter: params => {
            let s = `<b>${params[0].axisValue}</b><br/>`;
            for (const p of params) s += `${p.marker}${p.seriesName}: ${p.value} bpm<br/>`;
            return s;
          }},
          xAxis,
          yAxis:{ type:'value', axisLabel:{color:ECHART_TEXT_COLOR,fontSize:10,formatter:v=>v+' bpm'}, splitLine:{lineStyle:{color:ECHART_GRID_COLOR}} },
          series:[
            { name:'FC Média', type:'line', data:avg, lineStyle:{color:COLOR_HR,width:2}, itemStyle:{color:COLOR_HR}, areaStyle:{color:COLOR_HR+'22'}, symbol:'circle', symbolSize:5, smooth:false },
            ...(hasMax ? [{ name:'FC Máx', type:'line', data:max, lineStyle:{color:COLOR_HR+'66',width:1.5,type:'dashed'}, itemStyle:{color:COLOR_HR+'66'}, symbol:'none', smooth:false }] : []),
            ...(avgHR   ? [{ type:'line', data:avg.map(()=>avgHR), symbol:'none', lineStyle:{color:'#fff4',width:1.5,type:'dashed'}, tooltip:{show:false} }] : [])
          ]
        });
      });
    }

    if (laps.some(l => l.average_watts)) {
      const chartId = 'act-lap-watts';
      chartsGrid.insertAdjacentHTML('beforeend', `
        <div class="act-chart-card">
          <div class="act-chart-title"><span class="act-chart-dot" style="background:${COLOR_WATTS}"></span>Potência (Watts) por km</div>
          <div id="${chartId}" class="act-chart-el"></div>
        </div>`);
      requestAnimationFrame(() => {
        const el = document.getElementById(chartId);
        if (!el) return;
        const c = echarts.init(el, 'dark');
        ACT_CHARTS[chartId] = c;
        const vals = laps.map(l => l.average_watts ? Math.round(l.average_watts) : null);
        const avgW = d.average_watts ? Math.round(d.average_watts) : null;
        c.setOption({
          backgroundColor:'transparent', grid:gridOpts,
          tooltip:{ trigger:'axis', formatter: p => `${p[0].axisValue}<br/>${p[0].marker}${p[0].value}W` },
          xAxis,
          yAxis:{ type:'value', axisLabel:{color:ECHART_TEXT_COLOR,fontSize:10,formatter:v=>v+'W'}, splitLine:{lineStyle:{color:ECHART_GRID_COLOR}} },
          series:[
            { type:'bar', data:vals, itemStyle:{color:COLOR_WATTS, borderRadius:[3,3,0,0]}, barMaxWidth:52 },
            ...(avgW ? [{ type:'line', data:vals.map(()=>avgW), symbol:'none', lineStyle:{color:'#fff4',width:1.5,type:'dashed'}, tooltip:{show:false} }] : [])
          ]
        });
      });
    }
  }

  const tablesGrid = document.getElementById('act-tables-grid');
  tablesGrid.innerHTML = '';

  if (laps.length > 0) {
    const hasPower = laps.some(l => l.average_watts);
    const rows = laps.map((l, i) => {
      const lapPace = isRun  && l.average_speed ? fmtPace(l.average_speed)
                   : isRide && l.average_speed ? (l.average_speed*3.6).toFixed(1)+'km/h' : '—';
      return `<tr>
        <td style="color:var(--muted)">${i+1}</td>
        <td>${l.distance ? fmtDist(l.distance) : '—'}</td>
        <td>${l.moving_time ? fmtTime(l.moving_time) : '—'}</td>
        <td style="color:${COLOR};font-weight:600">${lapPace}</td>
        <td style="color:${COLOR_HR}">${l.average_heartrate ? Math.round(l.average_heartrate) : '—'}</td>
        ${hasPower ? `<td style="color:${COLOR_WATTS}">${l.average_watts ? Math.round(l.average_watts)+'W' : '—'}</td>` : ''}
      </tr>`;
    }).join('');
    tablesGrid.insertAdjacentHTML('beforeend', `
      <div class="act-table-card">
        <div class="act-table-title">Voltas (${laps.length})</div>
        <table class="act-table">
          <thead><tr><th>#</th><th>Dist</th><th>Tempo</th><th>${isRide?'Vel':'Pace'}</th><th>FC</th>${hasPower?'<th>Watts</th>':''}</tr></thead>
          <tbody>${rows}</tbody>
        </table>
      </div>`);
  }

  const best = d.best_efforts || [];
  if (best.length > 0) {
    const rows = best.map(b => {
      const rank = b.pr_rank === 1 ? '<span class="act-pr-badge">PR #1</span>'
                 : b.pr_rank  > 1 ? `<span class="act-pr-badge" style="background:var(--surface2);color:var(--muted)">#${b.pr_rank}</span>` : '—';
      return `<tr><td>${b.name}</td><td style="font-weight:600">${fmtTime(b.elapsed_time)}</td><td>${rank}</td></tr>`;
    }).join('');
    tablesGrid.insertAdjacentHTML('beforeend', `
      <div class="act-table-card">
        <div class="act-table-title">Melhores Esforços</div>
        <table class="act-table">
          <thead><tr><th>Esforço</th><th>Tempo</th><th>Rank</th></tr></thead>
          <tbody>${rows}</tbody>
        </table>
      </div>`);
  }

  const segs = d.segment_efforts || [];
  const segsWrap = document.getElementById('act-segments-wrap');
  segsWrap.innerHTML = '';
  if (segs.length > 0) {
    const hasWatts = segs.some(s => s.average_watts);
    const rows = segs.map(s => {
      const pr  = s.pr_rank  === 1 ? '<span class="act-pr-badge">PR #1</span>'
                : s.pr_rank  >  1 ? `<span class="act-pr-badge" style="background:var(--surface2);color:var(--muted)">#${s.pr_rank}</span>` : '';
      const kom = s.kom_rank === 1 ? '<span class="act-kom-badge">KOM</span>' : '';
      const segVal = s.distance && s.moving_time
        ? (isRide ? (s.distance/s.moving_time*3.6).toFixed(1)+'km/h' : fmtPace(s.distance/s.moving_time))
        : '—';
      return `<tr>
        <td style="font-weight:500">${s.name}</td>
        <td>${s.distance ? fmtDist(s.distance) : '—'}</td>
        <td>${fmtTime(s.elapsed_time)}</td>
        <td style="color:${COLOR};font-weight:600">${segVal}</td>
        <td>${kom||pr||'—'}</td>
        ${hasWatts ? `<td style="color:${COLOR_WATTS}">${s.average_watts ? Math.round(s.average_watts)+'W' : '—'}</td>` : ''}
      </tr>`;
    }).join('');
    segsWrap.innerHTML = `
      <div class="act-table-card">
        <div class="act-table-title">Segmentos (${segs.length})</div>
        <table class="act-table">
          <thead><tr><th>Segmento</th><th>Dist</th><th>Tempo</th><th>${isRide?'Vel':'Pace'}</th><th>Rank</th>${hasWatts?'<th>Watts</th>':''}</tr></thead>
          <tbody>${rows}</tbody>
        </table>
      </div>`;
  }
}
