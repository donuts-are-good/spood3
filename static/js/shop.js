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

// Format large numbers on the shop page (credits balance and item prices)
document.addEventListener('DOMContentLoaded', function() {
    formatShopNumbers();
});

function formatShopNumbers() {
    // Header credits
    const creditsEl = document.querySelector('.shop-container .credits-amount');
    if (creditsEl) {
        const raw = creditsEl.textContent.replace(/[^0-9]/g, '');
        const value = parseInt(raw, 10);
        if (!isNaN(value)) {
            creditsEl.textContent = `💰 ${formatLargeNumber(value)} Credits`;
            creditsEl.title = `${value.toLocaleString()} Credits`;
        }
    }

    // Item prices
    document.querySelectorAll('.shop-grid .item-price').forEach(el => {
        const raw = el.textContent.replace(/[^0-9]/g, '');
        const value = parseInt(raw, 10);
        if (!isNaN(value)) {
            el.textContent = `💰 ${formatLargeNumber(value)} Credits`;
            el.title = `${value.toLocaleString()} Credits`;
        }
    });
}

// Abbreviated number formatting (shared with base/casino style)
function formatLargeNumber(num) {
    if (typeof num !== 'number') {
        num = parseInt(num, 10) || 0;
    }
    if (num < 1000) return num.toString();
    const suffixes = [
        { value: 1e18, suffix: 'Qi' },
        { value: 1e15, suffix: 'Qa' },
        { value: 1e12, suffix: 'T'  },
        { value: 1e9,  suffix: 'B'  },
        { value: 1e6,  suffix: 'M'  },
        { value: 1e3,  suffix: 'K'  },
    ];
    for (let i = 0; i < suffixes.length; i++) {
        if (num >= suffixes[i].value) {
            const result = (num / suffixes[i].value).toFixed(2);
            return (result.endsWith('.00') ? result.slice(0, -3) : result) + suffixes[i].suffix;
        }
    }
    return num.toString();
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
            if (window.toast && window.toast.success) window.toast.success(`Purchased ${itemName}${quantityText}`, 4000);
            // Live update header wallet and shop balance display
            if (typeof data.new_balance === 'number') {
                updateGlobalCreditsDisplay(data.new_balance);
                updateShopHeaderCredits(data.new_balance);
            }
            // Live update inventory grid if server returned updated item
            if (data.updated_inventory_item) {
                updateShopInventoryItem(data.updated_inventory_item);
            }
            // Optionally disable unique item button immediately
            disableUniqueIfNeeded(itemId);
        } else {
            if (window.toast && window.toast.error) window.toast.error(`Purchase failed: ${data.error}`, 5000);
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

function updateShopInventoryItem(inv) {
    try {
        const grid = document.querySelector('.inventory-grid');
        if (!grid) return;
        const sel = `.inventory-item[data-shop-item-id="${inv.shop_item_id}"]`;
        let itemEl = grid.querySelector(sel);
        if (!itemEl) {
            // Create a new inventory card if not present
            itemEl = document.createElement('div');
            itemEl.className = 'inventory-item';
            itemEl.setAttribute('data-shop-item-id', String(inv.shop_item_id));
            itemEl.innerHTML = `
                <div class="inventory-emoji"></div>
                <div class="inventory-name"></div>
                <div class="inventory-quantity"></div>
            `;
            grid.prepend(itemEl);
        }
        const emojiEl = itemEl.querySelector('.inventory-emoji');
        const nameEl = itemEl.querySelector('.inventory-name');
        const qtyEl = itemEl.querySelector('.inventory-quantity');
        if (emojiEl) emojiEl.textContent = inv.emoji || '';
        if (nameEl) nameEl.textContent = inv.name || '';
        if (qtyEl) qtyEl.textContent = `×${inv.quantity || 0}`;
    } catch (e) {
        console.warn('Failed to live-update inventory item', e);
    }
}

// Update base layout header credits link
function updateGlobalCreditsDisplay(newBalance) {
    try {
        const nav = document.querySelector('nav');
        if (!nav) return;
        const link = nav.querySelector('a[href="/shop"]');
        if (!link) return;
        const formatted = formatLargeNumber(newBalance);
        link.textContent = `Credits: ${formatted}`;
        link.title = `${newBalance.toLocaleString()} Violence Credits`;
    } catch (_) {}
}

// Update the shop page header credit display
function updateShopHeaderCredits(newBalance) {
    const creditsEl = document.querySelector('.shop-container .credits-amount');
    if (!creditsEl) return;
    creditsEl.textContent = `💰 ${formatLargeNumber(newBalance)} Credits`;
    creditsEl.title = `${newBalance.toLocaleString()} Credits`;
}

function disableUniqueIfNeeded(itemId) {
    const card = document.querySelector(`.shop-item[data-item-id="${itemId}"]`);
    if (!card) return;
    // If this was a unique item, disable its buy button
    const btn = card.querySelector('.buy-button');
    if (btn) {
        btn.disabled = true;
        btn.textContent = 'ALREADY OWNED';
    }
}