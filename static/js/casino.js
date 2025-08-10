// Casino JavaScript - UI interactions and backend API calls only
// ALL GAME LOGIC IS SERVER-SIDE - this only handles UI interactions

document.addEventListener('DOMContentLoaded', function() {
    initializeCasino();
});

function initializeCasino() {
    // Initialize tab switching
    initializeTabs();
    
    // Load progressive jackpot on page load
    loadProgressiveJackpot();
    
    // Amount adjustment buttons (1/2 and 2x)
    document.querySelectorAll('.control-btn[data-action]').forEach(btn => {
        btn.addEventListener('click', function() {
            const game = this.dataset.game;
            const action = this.dataset.action;
            const input = document.getElementById(game + '-amount');
            let currentValue = parseInt(input.value) || 0;
            
            if (action === 'half') {
                input.value = Math.floor(currentValue / 2);
            } else if (action === 'double') {
                input.value = currentValue * 2;
            }
            
            // Ensure within bounds
            input.value = Math.max(1, Math.min(parseInt(input.max), parseInt(input.value)));
        });
    });
    
    // Moon flip betting - only UI interaction, all logic is server-side
    // NOTE: Moon flip betting is now handled through moon option selection
    
    // Hi-Low Step 1: Place bet to reveal first card
    const placeBetButton = document.getElementById('place-hilow-bet');
    if (placeBetButton) {
        placeBetButton.addEventListener('click', function() {
            placeHiLowBetStep1();
        });
    }
    
    // Hi-Low Step 2: Make guess to reveal second card
    document.querySelectorAll('.bet-btn[data-guess]').forEach(btn => {
        btn.addEventListener('click', function() {
            const guess = this.dataset.guess;
            placeHiLowBetStep2(guess);
        });
    });

    // Moon option visual selection (UI only)
    document.querySelectorAll('.moon-option').forEach(option => {
        option.addEventListener('click', function() {
            const choice = this.dataset.choice;
            // Visual feedback only - actual betting happens via bet buttons
            document.querySelectorAll('.moon-option').forEach(opt => {
                opt.classList.remove('selected');
                opt.style.borderColor = '';
            });
            this.classList.add('selected');
            
            // Update the bet button to show selected moon and enable it
            const betButton = document.getElementById('moonflip-bet-btn');
            const moonEmoji = choice === 'full' ? '🌕' : '🌑';
            const moonName = choice === 'full' ? 'FULL MOON' : 'NEW MOON';
            betButton.textContent = `🌙 BET ON ${moonName} ${moonEmoji}`;
            betButton.classList.remove('disabled');
            betButton.disabled = false;
            betButton.dataset.choice = choice;
            
            // Move to step 2: Set bet amount
            document.getElementById('moon-step1-text').classList.remove('active');
            document.getElementById('moon-step2-text').classList.add('active');
            
            // Add click listener to the bet button if not already added
            if (!betButton.hasAttribute('data-listener-added')) {
                betButton.addEventListener('click', function() {
                    if (!this.disabled && this.dataset.choice) {
                        placeMoonFlipBet(this.dataset.choice);
                    }
                });
                betButton.setAttribute('data-listener-added', 'true');
            }
        });
    });

    // Slots spinning - all logic server-side
    document.getElementById('slots-spin-btn').addEventListener('click', function() {
        spinSlots();
    });
}

function initializeTabs() {
    // Restore previously active tab from localStorage
    const lastActiveTab = localStorage.getItem('casino-active-tab') || 'moonflip';
    
    // Set initial tab state
    document.querySelectorAll('.tab-btn').forEach(tab => tab.classList.remove('active'));
    document.querySelectorAll('.game-panel').forEach(panel => panel.classList.remove('active'));
    
    // Activate the restored tab
    const activeTabBtn = document.querySelector(`.tab-btn[data-tab="${lastActiveTab}"]`);
    const activePanel = document.getElementById(lastActiveTab + '-panel');
    
    if (activeTabBtn && activePanel) {
        activeTabBtn.classList.add('active');
        activePanel.classList.add('active');
    }
    
    // Tab switching functionality
    document.querySelectorAll('.tab-btn').forEach(tab => {
        tab.addEventListener('click', function() {
            const targetTab = this.dataset.tab;
            
            // Save the active tab to localStorage
            localStorage.setItem('casino-active-tab', targetTab);
            
            // Update tab buttons
            document.querySelectorAll('.tab-btn').forEach(t => t.classList.remove('active'));
            this.classList.add('active');
            
            // Update game panels
            document.querySelectorAll('.game-panel').forEach(panel => {
                panel.classList.remove('active');
            });
            document.getElementById(targetTab + '-panel').classList.add('active');
            
            // Reset any game states when switching tabs
            resetGameStates();
        });
    });
}

function resetGameStates() {
    // Reset Hi-Low game state
    hiLowFirstCard = null;
    hiLowBetAmount = 0;
    
    // Reset Hi-Low UI
    const placeBetGroup = document.getElementById('place-bet-group');
    const guessButtonsGroup = document.getElementById('guess-buttons-group');
    const step1Text = document.getElementById('step1-text');
    const step2Text = document.getElementById('step2-text');
    
    if (placeBetGroup) placeBetGroup.classList.remove('hidden');
    if (guessButtonsGroup) guessButtonsGroup.classList.add('hidden');
    if (step1Text) step1Text.classList.add('active');
    if (step2Text) step2Text.classList.remove('active');
    
    // Reset card displays
    const firstCard = document.getElementById('first-card-display');
    const nextCard = document.getElementById('next-card-display');
    if (firstCard) {
        firstCard.innerHTML = '<span>?</span>';
        firstCard.className = 'card placeholder';
    }
    if (nextCard) {
        nextCard.innerHTML = '<span>?</span>';
        nextCard.className = 'card placeholder';
    }
    
    // Reset Moon Flip state
    document.querySelectorAll('.moon-option').forEach(opt => {
        opt.classList.remove('selected');
    });
    const moonBetBtn = document.getElementById('moonflip-bet-btn');
    if (moonBetBtn) {
        moonBetBtn.textContent = '🌙 SELECT A MOON PHASE FIRST';
        moonBetBtn.classList.add('disabled');
        moonBetBtn.disabled = true;
        moonBetBtn.removeAttribute('data-choice');
    }
    
    // Reset Moon Flip instructions
    const moonStep1 = document.getElementById('moon-step1-text');
    const moonStep2 = document.getElementById('moon-step2-text');
    if (moonStep1) moonStep1.classList.add('active');
    if (moonStep2) moonStep2.classList.remove('active');
    
    // Reset Slots state
    document.querySelectorAll('.slot-reel').forEach(reel => {
        reel.classList.remove('spinning', 'winner');
        reel.textContent = '🎰';
    });
    document.querySelectorAll('.payline-row').forEach(line => {
        line.classList.remove('winning');
    });
    
    // Clear all result displays
    document.querySelectorAll('.game-result').forEach(result => {
        result.innerHTML = '';
        result.className = 'game-result'; // Reset class to remove win/lose/neutral styling
    });
}

// Global variables to store Step 1 data for Step 2
let hiLowFirstCard = '';
let hiLowBetAmount = 0;

// Hi-Low Step 1: Place bet and get first card - ALL LOGIC SERVER-SIDE
function placeHiLowBetStep1() {
    const amount = parseInt(document.getElementById('hilow-amount').value);
    
    if (!amount || amount <= 0) {
        showResult('hilow', 'Invalid bet amount', false);
        return;
    }
    
    // Disable the place bet button
    document.getElementById('place-hilow-bet').disabled = true;
    
    // Send bet to server - NO CLIENT-SIDE LOGIC
    fetch('/user/casino/hilow-step1', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            amount: amount
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            // Store data for Step 2
            hiLowFirstCard = data.first_card;
            hiLowBetAmount = data.amount;
            
            // Show server-generated first card
            const firstCardElement = document.getElementById('first-card-display');
            firstCardElement.innerHTML = `<span>${data.first_card}</span>`;
            firstCardElement.classList.remove('placeholder');
            firstCardElement.classList.add('revealed');
            
            // Move to step 2: show guess buttons, hide bet controls
            document.getElementById('step1-text').classList.remove('active');
            document.getElementById('step2-text').classList.add('active');
            document.getElementById('bet-amount-group').classList.add('hidden');
            document.getElementById('amount-controls-group').classList.add('hidden');
            document.getElementById('place-bet-group').classList.add('hidden');
            document.getElementById('guess-buttons-group').classList.remove('hidden');
            
            showResult('hilow', `First card: ${data.first_card}. Credits charged: ${data.amount}. Now pick higher or lower!`, null);
        } else {
            showResult('hilow', 'Error: ' + data.error, false);
            document.getElementById('place-hilow-bet').disabled = false;
        }
    })
    .catch(error => {
        showResult('hilow', 'Network error', false);
        document.getElementById('place-hilow-bet').disabled = false;
    });
}

// Hi-Low Step 2: Make guess and get result - ALL LOGIC SERVER-SIDE
function placeHiLowBetStep2(guess) {
    // Disable guess buttons
    document.querySelectorAll('.bet-btn[data-guess]').forEach(btn => btn.disabled = true);
    
    // Send guess to server - NO CLIENT-SIDE LOGIC
    fetch('/user/casino/hilow-step2', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            guess: guess,
            first_card: hiLowFirstCard,
            amount: hiLowBetAmount
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            // Show server-generated second card
            const nextCardElement = document.getElementById('next-card-display');
            nextCardElement.innerHTML = `<span>${data.second_card}</span>`;
            nextCardElement.classList.remove('placeholder');
            nextCardElement.classList.add('revealed');
            
            const resultText = `${data.second_card} vs ${data.first_card}! ${data.won ? 'YOU WIN!' : 'You lose.'} ${data.won ? `+${data.payout}` : `-${data.amount}`} credits`;
            showResult('hilow', resultText, data.won);
            
            // Reset for next round after 3 seconds
            setTimeout(() => {
                window.location.reload();
            }, 3000);
        } else {
            showResult('hilow', 'Error: ' + data.error, false);
        }
    })
    .catch(error => {
        showResult('hilow', 'Network error', false);
    })
    .finally(() => {
        // Re-enable guess buttons
        document.querySelectorAll('.bet-btn[data-guess]').forEach(btn => btn.disabled = false);
    });
}

// Moon Flip - ALL GAME LOGIC IS SERVER-SIDE
function placeMoonFlipBet(choice) {
    const amount = parseInt(document.getElementById('moonflip-amount').value);
    
    if (!amount || amount <= 0) {
        showResult('moonflip', 'Invalid bet amount', false);
        return;
    }
    
    // Disable betting buttons during request
    document.querySelectorAll('.bet-btn[data-choice]').forEach(btn => btn.disabled = true);
    
    // Send bet to server - NO CLIENT-SIDE RNG OR GAME LOGIC
    fetch('/user/casino/moonflip', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            amount: amount,
            choice: choice
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            // Display server-generated result
            const resultText = `${data.result === 'full' ? '🌕' : '🌑'} Result: ${data.result === 'full' ? 'Full Moon' : 'New Moon'}! ${data.won ? 'YOU WIN!' : 'You lose.'} ${data.won ? `+${data.payout}` : `-${data.amount}`} credits`;
            showResult('moonflip', resultText, data.won);
            
            // Refresh page to update credits from server
            setTimeout(() => window.location.reload(), 2000);
        } else {
            showResult('moonflip', 'Error: ' + data.error, false);
        }
    })
    .catch(error => {
        showResult('moonflip', 'Network error', false);
    })
    .finally(() => {
        // Re-enable betting buttons
        document.querySelectorAll('.bet-btn[data-choice]').forEach(btn => btn.disabled = false);
    });
}

// Slots - ALL GAME LOGIC IS SERVER-SIDE
function spinSlots() {
    const amount = parseInt(document.getElementById('slots-amount').value);
    
    if (!amount || amount <= 0) {
        showResult('slots', 'Invalid bet amount', false);
        return;
    }
    
    // Disable spin button during request
    const spinButton = document.getElementById('slots-spin-btn');
    spinButton.disabled = true;
    spinButton.textContent = '🎰 SPINNING...';
    
    // Start visual spinning animation
    const reels = document.querySelectorAll('.slot-reel');
    reels.forEach(reel => {
        reel.classList.add('spinning');
    });
    
    // Move to step 2 instructions
    document.getElementById('slots-step1-text').classList.remove('active');
    document.getElementById('slots-step2-text').classList.add('active');
    
    // Emergency fallback - force reset after 15 seconds if something goes wrong
    const emergencyTimeout = setTimeout(() => {
        console.log('Emergency reset triggered after 15 seconds');
        stopSpinning();
        resetSpinButton();
        showResult('slots', 'Game reset due to timeout', false);
    }, 15000);
    
    // Send bet to server - NO CLIENT-SIDE RNG OR GAME LOGIC
    fetch('/user/casino/slots', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            amount: amount
        })
    })
    .then(response => response.json())
    .then(data => {
        clearTimeout(emergencyTimeout); // Cancel emergency timeout
        console.log('Slots response:', data); // Debug logging
        if (data.success) {
            // Animate through the spin sequences from server
            animateSlotSequences(data.sequences, data.final_grid, data.winning_lines, data.won, data.payout, data.amount, data.new_balance);
        } else {
            console.log('Slots error:', data.error); // Debug logging
            stopSpinning();
            showResult('slots', 'Error: ' + data.error, false);
            resetSpinButton();
        }
    })
    .catch(error => {
        clearTimeout(emergencyTimeout); // Cancel emergency timeout
        console.error('Slots network error:', error); // Debug logging
        stopSpinning();
        showResult('slots', 'Network error', false);
        resetSpinButton();
    });
}

function animateSlotSequences(sequences, finalGrid, winningLines, won, payout, amount, newBalance) {
    const reels = document.querySelectorAll('.slot-reel');
    let sequenceIndex = 0;
    
    function showNextSequence() {
        if (sequenceIndex < sequences.length) {
            // Show current sequence
            const sequence = sequences[sequenceIndex];
            sequence.forEach((emoji, index) => {
                if (reels[index]) {
                    reels[index].textContent = emoji;
                }
            });
            
            sequenceIndex++;
            setTimeout(showNextSequence, 300); // 300ms between sequences
        } else {
            // Show final result
            finalGrid.forEach((emoji, index) => {
                if (reels[index]) {
                    reels[index].textContent = emoji;
                }
            });
            
            // Stop spinning animation
            stopSpinning();
            
            // Highlight winning lines
            setTimeout(() => {
                highlightWinningLines(winningLines);
                
                // Show result
                const resultText = won ? 
                    `🎰 JACKPOT! +${payout} credits! ${getWinningLinesText(winningLines)}` : 
                    `🎰 No winning lines. -${amount} credits.`;
                showResult('slots', resultText, won);
                
                // Simple timer-based reset - guaranteed to work
                setTimeout(() => {
                    resetSpinButton();
                    clearWinningHighlights();
                    updateCreditsDisplay(newBalance);
                    // Clear the result notification
                    const resultDiv = document.getElementById('slots-result');
                    if (resultDiv) {
                        resultDiv.innerHTML = '';
                        resultDiv.className = 'game-result'; // Reset styling
                    }
                    // Refresh the progressive jackpot
                    loadProgressiveJackpot();
                }, 1500);
            }, 500);
        }
    }
    
    // Start the sequence animation
    showNextSequence();
}

function clearWinningHighlights() {
    // Clear winner highlighting
    document.querySelectorAll('.slot-reel').forEach(reel => {
        reel.classList.remove('winner');
    });
    document.querySelectorAll('.payline-row').forEach(line => {
        line.classList.remove('winning');
    });
}

function updateCreditsDisplay(newBalance) {
    // Update the credits amount in the header
    const creditsElement = document.querySelector('.credits-amount');
    if (creditsElement) {
        creditsElement.textContent = newBalance.toLocaleString();
    }
}

function loadProgressiveJackpot() {
    fetch('/user/casino/jackpot')
        .then(response => response.json())
        .then(data => {
            const jackpotDisplay = document.getElementById('jackpot-display');
            if (jackpotDisplay) {
                jackpotDisplay.textContent = data.jackpot.toLocaleString();
            }
        })
        .catch(error => {
            console.error('Failed to load progressive jackpot:', error);
            const jackpotDisplay = document.getElementById('jackpot-display');
            if (jackpotDisplay) {
                jackpotDisplay.textContent = '1,000';
            }
        });
}

function stopSpinning() {
    document.querySelectorAll('.slot-reel').forEach(reel => {
        reel.classList.remove('spinning');
    });
}

function highlightWinningLines(winningLines) {
    // Clear previous winners
    document.querySelectorAll('.slot-reel').forEach(reel => reel.classList.remove('winner'));
    document.querySelectorAll('.payline-row').forEach(line => line.classList.remove('winning'));
    
    // Safety check - handle null or undefined winningLines
    if (!winningLines || !Array.isArray(winningLines)) {
        return;
    }
    
    winningLines.forEach(lineIndex => {
        // Highlight the payline indicator
        const paylineRow = document.querySelector(`[data-line="${lineIndex}"]`);
        if (paylineRow) {
            paylineRow.classList.add('winning');
        }
        
        // Highlight the winning reels (row 0 = positions 0,1,2; row 1 = positions 3,4,5; row 2 = positions 6,7,8)
        const startPos = lineIndex * 3;
        for (let i = 0; i < 3; i++) {
            const reelIndex = startPos + i;
            const reel = document.querySelector(`[data-row="${lineIndex}"][data-col="${i}"]`);
            if (reel) {
                reel.classList.add('winner');
            }
        }
    });
}

function getWinningLinesText(winningLines) {
    if (winningLines.length === 0) return '';
    const lineNames = winningLines.map(i => `Row ${i + 1}`);
    return `Winning lines: ${lineNames.join(', ')}`;
}

function resetSpinButton() {
    const spinButton = document.getElementById('slots-spin-btn');
    spinButton.disabled = false;
    spinButton.textContent = '🎰 SPIN THE REELS';
    
    // Reset instructions
    document.getElementById('slots-step1-text').classList.add('active');
    document.getElementById('slots-step2-text').classList.remove('active');
}

function showResult(game, message, won) {
    const resultDiv = document.getElementById(game + '-result');
    resultDiv.textContent = message;
    
    // Handle different result types
    if (won === true) {
        resultDiv.className = 'game-result win';
    } else if (won === false) {
        resultDiv.className = 'game-result lose';
    } else {
        // Neutral message (like Hi-Low step 1 info)
        resultDiv.className = 'game-result neutral';
    }
} 