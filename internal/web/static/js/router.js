function showPage(name) {
  document.querySelectorAll('[id^="page-"]').forEach(el => el.classList.add('hidden'));
  document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
  document.getElementById('page-' + name)?.classList.remove('hidden');
  document.getElementById('nav-' + name)?.classList.add('active');
  window.location.hash = name;
  if (name === 'dashboard') loadDashboard();
  if (name === 'goals') loadGoals('active');
  if (name === 'activities') loadActivities('');
}

window.addEventListener('popstate', () => {
  const page = (window.location.hash || '#dashboard').replace('#','').split('/')[0];
  showPage(page);
});
