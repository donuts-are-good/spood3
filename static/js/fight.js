// Format numbers like 1200 -> 1.2k, 1500000 -> 1.5M
function formatShort(n) {
    const num = typeof n === 'number' ? n : parseInt(n, 10);
    if (isNaN(num)) return '';
    if (num >= 1e9) return (num / 1e9).toFixed(num % 1e9 === 0 ? 0 : 1) + 'B';
    if (num >= 1e6) return (num / 1e6).toFixed(num % 1e6 === 0 ? 0 : 1) + 'M';
    if (num >= 1e3) return (num / 1e3).toFixed(num % 1e3 === 0 ? 0 : 1) + 'k';
    return String(num);
}

// Initialize bet stamps on load
document.addEventListener('DOMContentLoaded', function() {
    const stamps = document.querySelectorAll('.bet-stamp-text[data-amount]');
    stamps.forEach(el => {
        const amount = el.getAttribute('data-amount');
        el.textContent = formatShort(amount);
    });
});
// Apply effect (bless/curse) to a fighter
function applyEffect(itemId, fightId, fighterId, fighterName, effectName) {
    // Send effect application request directly without confirmation
    fetch('/user/fight/apply-effect', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            item_id: itemId,
            fight_id: fightId,
            fighter_id: fighterId,
            target_type: 'fighter'
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            showSuccess(`Successfully applied ${effectName} to ${fighterName}!`);

            try {
                // Update effect badges counts
                const f1Cur = document.querySelector('.fighter-card:first-of-type .effect-badge.curse-badge .effect-count');
                const f1Ble = document.querySelector('.fighter-card:first-of-type .effect-badge.blessing-badge .effect-count');
                const f2Cur = document.querySelector('.fighter-card:last-of-type .effect-badge.curse-badge .effect-count');
                const f2Ble = document.querySelector('.fighter-card:last-of-type .effect-badge.blessing-badge .effect-count');
                if (f1Cur) f1Cur.textContent = `${data.fighter1_curses}x`;
                if (f1Ble) f1Ble.textContent = `${data.fighter1_blessings}x`;
                if (f2Cur) f2Cur.textContent = `${data.fighter2_curses}x`;
                if (f2Ble) f2Ble.textContent = `${data.fighter2_blessings}x`;

                // Ensure indicator containers are visible if counts > 0
                const f1Indicators = document.querySelectorAll('.fighter-card')[0]?.querySelector('.effect-indicators');
                const f2Indicators = document.querySelectorAll('.fighter-card')[1]?.querySelector('.effect-indicators');
                if (f1Indicators && (data.fighter1_curses > 0 || data.fighter1_blessings > 0)) {
                    f1Indicators.style.display = '';
                }
                if (f2Indicators && (data.fighter2_curses > 0 || data.fighter2_blessings > 0)) {
                    f2Indicators.style.display = '';
                }

                // Update Tale of the Tape numbers for the target fighter
                const side = data.target_side; // 'f1' or 'f2'
                const map = {
                    strength: document.getElementById(`tot-${side}-strength`),
                    speed: document.getElementById(`tot-${side}-speed`),
                    endurance: document.getElementById(`tot-${side}-endurance`),
                    technique: document.getElementById(`tot-${side}-technique`),
                };
                if (data.updated_stats) {
                    if (map.strength) map.strength.textContent = data.updated_stats.strength;
                    if (map.speed) map.speed.textContent = data.updated_stats.speed;
                    if (map.endurance) map.endurance.textContent = data.updated_stats.endurance;
                    if (map.technique) map.technique.textContent = data.updated_stats.technique;
                }

                // Update the win highlight classes based on updated values
                const f1Vals = {
                    strength: document.getElementById('tot-f1-strength'),
                    speed: document.getElementById('tot-f1-speed'),
                    endurance: document.getElementById('tot-f1-endurance'),
                    technique: document.getElementById('tot-f1-technique'),
                };
                const f2Vals = {
                    strength: document.getElementById('tot-f2-strength'),
                    speed: document.getElementById('tot-f2-speed'),
                    endurance: document.getElementById('tot-f2-endurance'),
                    technique: document.getElementById('tot-f2-technique'),
                };
                function updateWinClass(stat) {
                    const v1 = parseInt(f1Vals[stat]?.textContent || '0', 10);
                    const v2 = parseInt(f2Vals[stat]?.textContent || '0', 10);
                    if (f1Vals[stat]) f1Vals[stat].classList.toggle('win', v1 > v2);
                    if (f2Vals[stat]) f2Vals[stat].classList.toggle('win', v2 > v1);
                }
                ['strength','speed','endurance','technique'].forEach(updateWinClass);

                // Update inventory quantity for the used item (if shown on page)
                const invButtons = document.querySelectorAll(`.effect-button[data-item-id="${itemId}"]`);
                if (invButtons && data.updated_inventory && typeof data.updated_inventory.quantity === 'number') {
                    // Find the quantity pill in the same effect-item card
                    invButtons.forEach(btn => {
                        const header = btn.closest('.effect-item')?.querySelector('.effect-header .effect-quantity');
                        if (header) header.textContent = `×${data.updated_inventory.quantity}`;
                    });
                }
            } catch (e) {
                console.warn('Live update failed gracefully:', e);
            }

            return;
        } else {
            if (data.limit_reached) {
                if (window.toast && window.toast.info) {
                    window.toast.info('You\'ve hit the max of 10 effects for this fight.', 5000);
                } else {
                    showError('You\'ve hit the max of 10 effects for this fight.');
                }
                return;
            }
            showError(`Failed to apply ${effectName}: ${data.error}`);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showError('Woah slow down, try again.');
    });
} 

// Convenience: read arguments from a button's data-* attributes
function applyEffectFromButton(btn) {
    try {
        const itemId = parseInt(btn.dataset.itemId, 10);
        const fightId = parseInt(btn.dataset.fightId, 10);
        const fighterId = parseInt(btn.dataset.fighterId, 10);
        const fighterName = btn.dataset.fighterName || '';
        const effectName = btn.dataset.effectName || '';
        return applyEffect(itemId, fightId, fighterId, fighterName, effectName);
    } catch (e) {
        console.error('applyEffectFromButton error', e);
        if (window.toast && window.toast.error) window.toast.error('Failed to apply effect.');
    }
}