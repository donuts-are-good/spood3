// Apply effect (bless/curse) to a fighter
function applyEffect(itemId, fighterId, fighterName, effectName) {
    // Confirm effect application
    if (!confirm(`Apply ${effectName} to ${fighterName}?`)) {
        return;
    }

    // Send effect application request
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
            alert(`Successfully applied ${effectName} to ${fighterName}!`);
            // Reload page to update inventory and show effects
            window.location.reload();
        } else {
            alert(`Failed to apply ${effectName}: ${data.error}`);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        alert('Failed to apply effect. Please try again.');
    });
} 