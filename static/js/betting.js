// Betting confirmation dialog
function confirmBet(event, fighterName) {
    const form = event.target;
    const amountInput = form.querySelector('input[name="amount"]');
    const amount = parseInt(amountInput.value);
    
    if (!amount || amount <= 0) {
        showError('Please enter a valid bet amount.');
        return false;
    }
    
    // Submit bet directly without confirmation
    return true;
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