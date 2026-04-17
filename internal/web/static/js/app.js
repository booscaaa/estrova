function loadAll() { loadDashboard(); }

window.addEventListener('DOMContentLoaded', () => {
  const hash = window.location.hash || '#dashboard';
  const page = hash.replace('#','').split('/')[0] || 'dashboard';
  showPage(page);
});

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') closeDrawer();
});
