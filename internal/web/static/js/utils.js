const SPORT_COLORS = { Run:'#fc4c02', Ride:'#3498db', Swim:'#00bcd4', Walk:'#2ecc71' };
const ECHART_GRID_COLOR = '#2e3158';
const ECHART_TEXT_COLOR = '#8892b0';

function toast(msg, type='success') {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast ' + type;
  el.classList.remove('hidden');
  setTimeout(() => el.classList.add('hidden'), 3000);
}

function openModal(id) { document.getElementById(id).classList.remove('hidden'); }
function closeModal(id) { document.getElementById(id).classList.add('hidden'); }

function sportBadge(sport) {
  const cls = {Run:'badge-run',Ride:'badge-ride',Swim:'badge-swim'}[sport] || 'badge-other';
  const icon = {Run:'🏃',Ride:'🚴',Swim:'🏊',Walk:'🚶'}[sport] || '🏋️';
  return `<span class="goal-badge ${cls}">${icon} ${sport}</span>`;
}

function sessionIcon(type) {
  return {Easy:'🟢',Long:'🔵',Tempo:'🟡',Interval:'🔴',Rest:'⚪',Race:'🏆',Cross:'🟣',Strength:'💪'}[type] || '🔵';
}

function fmtDist(m) { return m >= 1000 ? (m/1000).toFixed(1)+'km' : m+'m'; }
function fmtPace(ms) {
  if (!ms) return '';
  const sPerKm = 1000/ms;
  const m = Math.floor(sPerKm/60), s = Math.floor(sPerKm%60);
  return m+':'+(s<10?'0':'')+s+'/km';
}
function fmtTime(s) {
  const h = Math.floor(s/3600), m = Math.floor((s%3600)/60);
  return h > 0 ? `${h}h${m}m` : `${m}min`;
}

function formatDate(d) {
  if (!d) return '';
  const [y,m,day] = d.split('-');
  const days = ['Dom','Seg','Ter','Qua','Qui','Sex','Sáb'];
  const dt = new Date(parseInt(y), parseInt(m)-1, parseInt(day));
  return `${days[dt.getDay()]} ${day}/${m}`;
}
