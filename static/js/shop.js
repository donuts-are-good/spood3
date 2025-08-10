// Shop purchase functionality
function purchaseItem(itemId, itemName, price, quantity = 1) {
    const totalCost = price * quantity;
    const quantityText = quantity > 1 ? ` (${quantity}x)` : '';
    
    // Confirm purchase
    if (!confirm(`Purchase ${itemName}${quantityText} for ${totalCost.toLocaleString()} credits?`)) {
        return;
    }

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
            alert(`Successfully purchased ${itemName}${quantityText}!`);
            // Reload page to update credits and inventory
            window.location.reload();
        } else {
            alert(`Failed to purchase ${itemName}: ${data.error}`);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        alert('An error occurred while making the purchase.');
    });
} 