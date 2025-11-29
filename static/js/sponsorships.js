document.addEventListener('DOMContentLoaded', () => {
  const search = document.getElementById('fighter-search');
  if (search) {
    search.addEventListener('input', () => filterFighters(search.value));
  }

  document.querySelectorAll('.assign-btn').forEach(btn => {
    btn.addEventListener('click', () => assignSponsorship(btn));
  });
});

function filterFighters(query) {
  const cards = document.querySelectorAll('.eligible-card');
  const q = (query || '').toLowerCase();
  cards.forEach(card => {
    const name = (card.dataset.name || '').toLowerCase();
    card.style.display = name.includes(q) ? 'flex' : 'none';
  });
}

function assignSponsorship(button) {
  if (button.disabled) return;
  const fighterId = parseInt(button.dataset.fighterId, 10);
  const fighterName = button.dataset.fighterName || 'fighter';
  button.disabled = true;
  const originalText = button.textContent;
  button.textContent = 'Assigning...';

  fetch('/user/sponsorships/assign', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ fighter_id: fighterId })
  })
    .then(res => res.json().catch(() => null))
    .then(data => {
      if (!data || !data.success) {
        const msg = (data && data.error) ? data.error : 'Failed to assign sponsorship.';
        toastError(msg);
        button.disabled = false;
        button.textContent = originalText;
        return;
      }
      toastSuccess(`Sponsored ${fighterName}. Genome archived.`);
      updatePendingCount(data.pending_count);
      moveFighterToLicensed(button.closest('.eligible-card'), data.licensed);
    })
    .catch(() => {
      toastError('Network error.');
      button.disabled = false;
      button.textContent = originalText;
    });
}

function updatePendingCount(count) {
  const panel = document.querySelector('.permit-status');
  if (!panel) return;
  const copy = panel.querySelector('.status-copy p');
  if (count > 0) {
    panel.classList.add('active');
    if (copy) {
      copy.innerHTML = `You currently hold <strong>${count}</strong> unused Fighter Sponsorship${count > 1 ? 's' : ''}. Assign it below to activate genome rights.`;
    }
  } else {
    panel.classList.remove('active');
    if (copy) {
      copy.innerHTML = 'Purchase a Fighter Sponsorship from the shop to unlock genome keepsakes and hybrid privileges.';
    }
    document.querySelectorAll('.assign-btn').forEach(btn => btn.disabled = true);
  }
}

function moveFighterToLicensed(card, licensedInfo) {
  if (card) card.remove();
  const grid = document.querySelector('.licensed-grid');
  if (!grid) return;
  const div = document.createElement('div');
  const record = licensedInfo.record || '';
  div.className = 'licensed-card';
  div.innerHTML = `
    <img src="${licensedInfo.avatar || '/img-cdn/default.png'}" alt="${licensedInfo.name}" width="24" height="24">
    <div class="licensed-body">
      <a class="name" href="/fighter/${licensedInfo.fighter_id}">${licensedInfo.name}</a>
      <div class="meta">${record} · Licensed just now</div>
    </div>
  `;
  grid.prepend(div);
  const count = document.querySelector('.licensed-panel .count');
  if (count) {
    const current = parseInt(count.textContent, 10) || 0;
    count.textContent = `${current + 1} total`;
  }
}

function toastSuccess(msg) {
  try {
    if (window.toast && window.toast.success) {
      window.toast.success(msg, 4000);
      return;
    }
  } catch (_) {}
  alert(msg);
}

function toastError(msg) {
  try {
    if (window.toast && window.toast.error) {
      window.toast.error(msg, 4000);
      return;
    }
  } catch (_) {}
  alert(msg);
}

