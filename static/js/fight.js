// Apply effect (bless/curse) to a fighter
function applyEffect(itemId, fighterId, fighterName, effectName) {
    // Send effect application request directly without confirmation
    fetch('/user/fight/apply-effect', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            item_id: itemId,
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
            showError(`Failed to apply ${effectName}: ${data.error}`);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showError('Woah slow down, try again.');
    });
} 