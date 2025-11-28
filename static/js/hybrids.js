let selectedFighters = [];

document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.licensed-card').forEach(card => {
    card.addEventListener('click', () => toggleSelection(card));
  });
  const mixButton = document.getElementById('mix-button');
  if (mixButton) {
    mixButton.addEventListener('click', submitHybrid);
  }
  const nameInput = document.getElementById('hybrid-name');
  if (nameInput) {
    nameInput.addEventListener('input', validateName);
  }
});

function toggleSelection(card) {
  const id = parseInt(card.dataset.id, 10);
  if (selectedFighters.includes(id)) {
    selectedFighters = selectedFighters.filter(fid => fid !== id);
    card.classList.remove('selected');
  } else {
    if (selectedFighters.length >= 2) {
      const removed = selectedFighters.shift();
      const oldCard = document.querySelector(`.licensed-card[data-id="${removed}"]`);
      if (oldCard) oldCard.classList.remove('selected');
    }
    selectedFighters.push(id);
    card.classList.add('selected');
  }
  updateSummary();
  updateMixButtonState();
}

function validateName() {
  const nameInput = document.getElementById('hybrid-name');
  const hint = document.getElementById('hybrid-name-hint');
  const name = (nameInput.value || '').trim();
  if (name.length < 3) {
    hint.textContent = 'Name must be at least 3 characters.';
    nameInput.dataset.valid = 'false';
  } else if (name.length > 50) {
    hint.textContent = 'Name must be 50 characters or less.';
    nameInput.dataset.valid = 'false';
  } else if (!/^[a-zA-Z0-9\s\-_'\.]+$/.test(name)) {
    hint.textContent = 'Only letters, numbers, spaces, and - _ \'. . allowed.';
    nameInput.dataset.valid = 'false';
  } else {
    hint.textContent = '✓ Name looks sufficiently menacing.';
    nameInput.dataset.valid = 'true';
  }
  updateMixButtonState();
  return nameInput.dataset.valid === 'true';
}

function updateSummary() {
  const summary = document.getElementById('selection-summary');
  if (!summary) return;
  if (selectedFighters.length === 0) {
    summary.textContent = 'Select two fighters to generate the lab report.';
    return;
  }
  const names = selectedFighters.map(id => {
    const card = document.querySelector(`.licensed-card[data-id="${id}"]`);
    return card ? card.dataset.name : `#${id}`;
  });
  if (names.length === 1) {
    summary.textContent = `Selected ancestor: ${names[0]}. Choose one more.`;
  } else {
    summary.textContent = `Hybrid mix: ${names[0]} + ${names[1]}.`;
  }
}

function updateMixButtonState() {
  const mixButton = document.getElementById('mix-button');
  if (!mixButton) return;
  const nameInput = document.getElementById('hybrid-name');
  const validName = nameInput ? nameInput.dataset.valid === 'true' : false;
  mixButton.disabled = !(selectedFighters.length === 2 && validName);
}

function submitHybrid() {
  const mixButton = document.getElementById('mix-button');
  if (mixButton.disabled) return;
  const name = document.getElementById('hybrid-name').value.trim();
  mixButton.disabled = true;
  mixButton.textContent = 'Mixing...';

  fetch('/user/hybrids', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: name,
      ancestor1_id: selectedFighters[0],
      ancestor2_id: selectedFighters[1],
    })
  })
    .then(res => res.json().catch(() => null))
    .then(data => {
      if (data && data.success) {
        toastSuccess('Hybrid minted! Redirecting...');
        window.location.href = data.redirect || `/fighter/${data.fighter_id}`;
      } else {
        const msg = (data && data.error) ? data.error : 'Failed to create hybrid.';
        toastError(msg);
        mixButton.disabled = false;
        mixButton.textContent = '⚗️ Run Hybridization';
      }
    })
    .catch(() => {
      toastError('Network error.');
      mixButton.disabled = false;
      mixButton.textContent = '⚗️ Run Hybridization';
    });
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

