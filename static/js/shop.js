// Shop purchase functionality
function purchaseItem(itemId, itemName, price, quantity = 1) {
    const totalCost = price * quantity;
    const quantityText = quantity > 1 ? ` (${quantity}x)` : '';
    
    // If buying the High Roller Card, open styled modal instead of native confirm
    if (itemName && itemName.toLowerCase().includes('high roller')) {
        return openHighRollerModal(() => doPurchase(itemId, itemName, quantity));
    }

    return doPurchase(itemId, itemName, quantity);
}

function doPurchase(itemId, itemName, quantity) {
    // Send purchase request
    fetch('/user/shop/purchase', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            item_id: itemId,
            quantity: quantity
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            const quantityText = quantity > 1 ? ` (${quantity}x)` : '';
            showSuccess(`Successfully purchased ${itemName}${quantityText}!`);
            // Reload page to update credits and inventory
            window.location.reload();
        } else {
            showError(`Failed to purchase ${itemName}: ${data.error}`);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showError('An error occurred while making the purchase.');
    });
} 

// High Roller modal wiring
function openHighRollerModal(onConfirm) {
    const modal = document.getElementById('highroller-modal');
    if (!modal) return;
    modal.classList.remove('hidden');
    const overlay = modal.querySelector('.modal-overlay');
    const cancel = modal.querySelector('#hr-cancel');
    const confirmBtn = modal.querySelector('#hr-confirm');

    const close = () => modal.classList.add('hidden');
    const cleanup = () => {
        overlay.removeEventListener('click', onOverlay);
        cancel.removeEventListener('click', onCancel);
        confirmBtn.removeEventListener('click', onOk);
    };
    const onOverlay = () => { close(); cleanup(); };
    const onCancel = () => { close(); cleanup(); };
    const onOk = () => { close(); cleanup(); if (typeof onConfirm === 'function') onConfirm(); };

    overlay.addEventListener('click', onOverlay);
    cancel.addEventListener('click', onCancel);
    confirmBtn.addEventListener('click', onOk);
}