async function openConflictsModal() {
  document.getElementById('modal-conflicts-body').innerHTML = '<div class="loading">Carregando conflitos...</div>';
  openModal('modal-conflicts');

  const conflicts = await fetch('/api/conflicts').then(r=>r.json()).catch(()=>[]);
  if (!conflicts.length) {
    document.getElementById('modal-conflicts-body').innerHTML = '<div style="text-align:center;padding:30px;color:var(--muted)">✅ Nenhum conflito detectado!</div>';
    document.getElementById('dash-conflict-alert').classList.add('hidden');
    return;
  }

  const typeIcon = t => ({Easy:'🟢',Long:'🔵',Tempo:'🟠',Interval:'🔴',Race:'🏆',Rest:'⬜',Cross:'🟣',Strength:'⚫'}[t]||'⚪');

  const html = conflicts.map(c => `
    <div class="conflict-row" id="conflict-row-${c.session_id_1}-${c.session_id_2}">
      <div class="conflict-row-date">📅 ${c.date}</div>
      <div class="conflict-sides">
        <div class="conflict-side">
          <div class="conflict-side-goal">${c.goal_name_1}</div>
          <div class="conflict-side-type">${typeIcon(c.session_type_1)} ${c.session_type_1}</div>
          ${c.pace_target_1 ? `<div class="conflict-side-pace">${c.pace_target_1}</div>` : ''}
          <div class="conflict-side-actions">
            <button class="btn-conflict-edit" onclick="openEditConflictSession(${c.session_id_1},'${c.goal_name_1.replace(/'/g,"\\'")}')">✏️ Editar</button>
            <button class="btn-conflict-del" id="del-btn-${c.session_id_1}" onclick="deleteConflictFromModal(${c.session_id_1}, 'conflict-row-${c.session_id_1}-${c.session_id_2}', this)">🗑 Excluir</button>
          </div>
        </div>
        <div class="conflict-vs">VS</div>
        <div class="conflict-side">
          <div class="conflict-side-goal">${c.goal_name_2}</div>
          <div class="conflict-side-type">${typeIcon(c.session_type_2)} ${c.session_type_2}</div>
          ${c.pace_target_2 ? `<div class="conflict-side-pace">${c.pace_target_2}</div>` : ''}
          <div class="conflict-side-actions">
            <button class="btn-conflict-edit" onclick="openEditConflictSession(${c.session_id_2},'${c.goal_name_2.replace(/'/g,"\\'")}')">✏️ Editar</button>
            <button class="btn-conflict-del" id="del-btn-${c.session_id_2}" onclick="deleteConflictFromModal(${c.session_id_2}, 'conflict-row-${c.session_id_1}-${c.session_id_2}', this)">🗑 Excluir</button>
          </div>
        </div>
      </div>
    </div>`).join('');

  document.getElementById('modal-conflicts-body').innerHTML = html;
}

async function deleteConflictFromModal(sessionId, rowId, btn) {
  if (btn.dataset.confirming) {
    btn.disabled = true;
    btn.textContent = '...';
    const res = await fetch(`/api/sessions/${sessionId}`, { method: 'DELETE' })
      .then(r=>r.json()).catch(()=>null);
    if (!res) { toast('Erro ao excluir', 'error'); btn.disabled=false; btn.textContent='🗑 Excluir'; return; }
    const row = document.getElementById(rowId);
    if (row) row.remove();
    toast('Sessão removida!');
    const remaining = document.querySelectorAll('#modal-conflicts-body .conflict-row').length;
    if (remaining === 0) {
      document.getElementById('modal-conflicts-body').innerHTML = '<div style="text-align:center;padding:30px;color:var(--success)">✅ Todos os conflitos foram resolvidos!</div>';
      document.getElementById('dash-conflict-alert').classList.add('hidden');
    } else {
      document.getElementById('dash-conflict-msg').textContent = `${remaining} conflito(s) detectado(s) entre objetivos — clique para resolver`;
    }
    return;
  }
  btn.dataset.confirming = '1';
  btn.textContent = '⚠️ Confirmar?';
  btn.style.background = 'rgba(231,76,60,.4)';
  setTimeout(() => {
    if (btn.dataset.confirming) {
      delete btn.dataset.confirming;
      btn.textContent = '🗑 Excluir';
      btn.style.background = '';
    }
  }, 3000);
}

async function openEditConflictSession(sessionId, goalName) {
  const s = await fetch(`/api/sessions/${sessionId}`).then(r=>r.json()).catch(()=>null);
  if (!s) { toast('Sessão não encontrada', 'error'); return; }

  document.getElementById('edit-session-id').value = sessionId;
  document.getElementById('edit-session-goal-label').textContent = `Objetivo: ${goalName} · ${s.date} (${s.day_of_week})`;
  document.getElementById('edit-session-type').value = s.session_type || 'Easy';
  document.getElementById('edit-session-dist').value = s.distance_km || 0;
  document.getElementById('edit-session-dur').value = s.duration_min || 0;
  document.getElementById('edit-session-pace').value = s.pace_target || '';
  document.getElementById('edit-session-hr').value = s.hr_zone || '';
  document.getElementById('edit-session-desc').value = s.description || '';
  openModal('modal-edit-session');
}

async function saveEditedSession() {
  const id = parseInt(document.getElementById('edit-session-id').value);
  const body = {
    session_type: document.getElementById('edit-session-type').value,
    description:  document.getElementById('edit-session-desc').value,
    pace_target:  document.getElementById('edit-session-pace').value,
    hr_zone:      document.getElementById('edit-session-hr').value,
    notes:        '',
    distance_km:  parseFloat(document.getElementById('edit-session-dist').value) || 0,
    duration_min: parseInt(document.getElementById('edit-session-dur').value) || 0,
  };
  const res = await fetch(`/api/sessions/${id}`, {
    method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body)
  }).then(r=>r.json()).catch(()=>null);

  if (!res) { toast('Erro ao salvar', 'error'); return; }
  closeModal('modal-edit-session');
  toast('Sessão atualizada!');
  if (!document.getElementById('modal-conflicts').classList.contains('hidden')) {
    await openConflictsModal();
  } else if (currentGoalID) {
    await loadPlan(currentGoalID);
    closeDrawer();
  }
}

async function deleteConflictSession(sessionId, btn) {
  if (btn.dataset.confirming) {
    btn.disabled = true;
    btn.textContent = '...';
    const res = await fetch(`/api/sessions/${sessionId}`, { method: 'DELETE' })
      .then(r=>r.json()).catch(()=>null);
    if (!res) { toast('Erro ao excluir', 'error'); btn.disabled=false; btn.textContent='🗑 Excluir'; return; }
    toast('Sessão removida!');
    await loadPlan(currentGoalID);
    closeDrawer();
    return;
  }
  btn.dataset.confirming = '1';
  btn.textContent = '⚠️ Confirmar?';
  btn.style.background = 'rgba(231,76,60,.4)';
  setTimeout(() => {
    if (btn.dataset.confirming) {
      delete btn.dataset.confirming;
      btn.textContent = '🗑 Excluir';
      btn.style.background = '';
    }
  }, 3000);
}
