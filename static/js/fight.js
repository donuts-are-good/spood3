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
            // Reload page to update inventory and show effects
            window.location.reload();
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