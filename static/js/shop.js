// Shop purchase functionality
function purchaseItem(itemId, itemName, price, quantity = 1) {
    const totalCost = price * quantity;
    const quantityText = quantity > 1 ? ` (${quantity}x)` : '';
    
    // If buying the High Roller Card, show a compact disclosure
    if (itemName && itemName.toLowerCase().includes('high roller')) {
        const receipt = [
            'Patron Disclosure — Itemized Receipt',
            'Weekly Patron Tithe: 20% of account balance (applied Mondays)\n',
            'Estimated allocation:',
            '  • Little Spoodys Endowment for the Arts ............ 11%',
            '  • Youth Enrichment & After‑Naps .................... 3%',
            '  • Administrative Handling Fee ...................... 3%',
            '  • Community Outreach ░░░░░░ ........................ 1%',
            '  • Executive Wellness & Recovery .................... 2%\n',
            'Terms: Non‑refundable patronage; allocations may drift; filings may be decorative;',
            '       access may be revoked for cause or vibes.\n',
            'Proceed with purchase?'
        ].join('\n');
        const ok = confirm(receipt);
        if (!ok) return;
    }

    // Send purchase request directly without confirmation
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