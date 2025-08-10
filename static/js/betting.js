// Betting confirmation dialog
function confirmBet(event, fighterName) {
    const form = event.target;
    const amountInput = form.querySelector('input[name="amount"]');
    const amount = parseInt(amountInput.value);
    
    if (!amount || amount <= 0) {
        alert('Please enter a valid bet amount.');
        return false;
    }
    
    const confirmation = confirm(
        `Are you sure you want to bet ${amount} credits on ${fighterName}?\n\n` +
        `If ${fighterName} wins, you'll receive ${amount * 2} credits total.\n` +
        `If ${fighterName} loses, you'll lose your ${amount} credits.\n\n` +
        `This action cannot be undone.`
    );
    
    return confirmation;
}

// Add some visual feedback for bet inputs
document.addEventListener('DOMContentLoaded', function() {
    const betInputs = document.querySelectorAll('.bet-input');
    
    betInputs.forEach(input => {
        input.addEventListener('input', function() {
            const value = parseInt(this.value);
            const max = parseInt(this.getAttribute('max'));
            
            // Visual feedback for invalid amounts
            if (value > max) {
                this.style.borderColor = '#ff4444';
                this.style.color = '#ff4444';
            } else if (value > 0) {
                this.style.borderColor = '#4A90E2';
                this.style.color = '#fff';
            } else {
                this.style.borderColor = '#555';
                this.style.color = '#fff';
            }
        });
        
        // Reset styles on blur if valid
        input.addEventListener('blur', function() {
            const value = parseInt(this.value);
            const max = parseInt(this.getAttribute('max'));
            
            if (value <= max && value > 0) {
                this.style.borderColor = '#555';
                this.style.color = '#fff';
            }
        });
    });
}); 