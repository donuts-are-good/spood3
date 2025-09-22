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

    // Format the casino header credits display (abbreviated with full amount on hover)
    formatCasinoHeaderCredits();
    
    // Amount adjustment buttons (1/2, 2x, MAX)
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
            } else if (action === 'max') {
                // Use the input's max attribute (bound to user's credits)
                const max = parseInt(input.max);
                if (!isNaN(max)) input.value = max;
            }
            
            // Ensure within bounds
            input.value = Math.max(1, Math.min(parseInt(input.max), parseInt(input.value)));
            updateAbbrevFor(game);
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
    const spin10 = document.getElementById('slots-spin10-btn');
    if (spin10) {
        spin10.addEventListener('click', function() { spinSlotsSeries(10); });
    }

    // Blackjack controls
    const bjStart = document.getElementById('blackjack-start');
    if (bjStart) {
        bjStart.addEventListener('click', blackjackStart);
    }
    const bjHit = document.getElementById('blackjack-hit');
    if (bjHit) {
        bjHit.addEventListener('click', blackjackHit);
    }
    const bjStand = document.getElementById('blackjack-stand');
    if (bjStand) {
        bjStand.addEventListener('click', blackjackStand);
    }

    // Live abbreviated number display for each bet input
    ['moonflip','hilow','slots','blackjack'].forEach(game => {
        const input = document.getElementById(game + '-amount');
        if (input) {
            const handler = () => updateAbbrevFor(game);
            input.addEventListener('input', handler);
            input.addEventListener('change', handler);
            // initial
            updateAbbrevFor(game);
        }
    });

    // Extortion modal wiring
    const extModal = document.getElementById('extortion-modal');
    if (extModal) {
        const payBtn = document.getElementById('extortion-pay');
        const runBtn = document.getElementById('extortion-run');
        // Do NOT close on overlay click for extortion; force explicit choice
        if (payBtn) payBtn.addEventListener('click', () => resolveExtortion('pay'));
        if (runBtn) runBtn.addEventListener('click', () => resolveExtortion('run'));
    }
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

    // Reset Blackjack UI
    const up = document.getElementById('dealer-upcard');
    if (up) { up.className = 'card placeholder'; up.innerHTML = '<span>?</span>'; }
    const p1 = document.getElementById('player-card-1');
    if (p1) { p1.className = 'card placeholder'; p1.innerHTML = '<span>?</span>'; }
    const p2 = document.getElementById('player-card-2');
    if (p2) { p2.className = 'card placeholder'; p2.innerHTML = '<span>?</span>'; }
    const playGroup = document.getElementById('blackjack-play-group');
    const startGroup = document.getElementById('blackjack-start-group');
    if (playGroup) playGroup.classList.add('hidden');
    if (startGroup) startGroup.classList.remove('hidden');
}

function resetBlackjackUI() {
    // Clear the dynamic table render and restore placeholders
    const table = document.getElementById('blackjack-table');
    if (table) {
        table.innerHTML = '';
        const up = document.createElement('div');
        up.id = 'dealer-upcard';
        up.className = 'card placeholder';
        up.innerHTML = '<span>?</span>';
        table.appendChild(up);
        const sep = document.createElement('div');
        sep.className = 'vs-text';
        sep.textContent = 'DEALER';
        table.appendChild(sep);
        const p1 = document.createElement('div');
        p1.id = 'player-card-1';
        p1.className = 'card placeholder';
        p1.innerHTML = '<span>?</span>';
        table.appendChild(p1);
        const p2 = document.createElement('div');
        p2.id = 'player-card-2';
        p2.className = 'card placeholder';
        p2.innerHTML = '<span>?</span>';
        table.appendChild(p2);
    }
    // Do NOT clear history here; it should persist until next snapshot
    const resultDiv = document.getElementById('blackjack-result');
    if (resultDiv) { resultDiv.innerHTML = ''; resultDiv.className = 'game-result'; }
    const startGroup = document.getElementById('blackjack-start-group');
    const playGroup = document.getElementById('blackjack-play-group');
    if (startGroup) startGroup.classList.remove('hidden');
    if (playGroup) playGroup.classList.add('hidden');
    const bjStart = document.getElementById('blackjack-start');
    if (bjStart) bjStart.disabled = false;

    // Clear state for next round
    blackjackState = { amount: 0, dealerUpcard: '', playerHand: [], state: null };

    // Re-enable input and buttons to avoid cursor lockouts
    const amountInput = document.getElementById('blackjack-amount');
    if (amountInput) amountInput.disabled = false;
    const bjHit = document.getElementById('blackjack-hit');
    const bjStand = document.getElementById('blackjack-stand');
    if (bjHit) bjHit.disabled = false;
    if (bjStand) bjStand.disabled = false;
}

// ---- Recent outcomes (last 5) ----
const BJ_RECENT_KEY = 'bj_recent_outcomes';
function pushRecentOutcome(text, kind) {
    try {
        const entry = { text, kind };
        const arr = JSON.parse(localStorage.getItem(BJ_RECENT_KEY) || '[]');
        arr.unshift(entry);
        while (arr.length > 5) arr.pop();
        localStorage.setItem(BJ_RECENT_KEY, JSON.stringify(arr));
        renderRecentOutcomes();
    } catch (_) {}
}
function renderRecentOutcomes() {
    const container = document.getElementById('blackjack-recent');
    if (!container) return;
    const arr = JSON.parse(localStorage.getItem(BJ_RECENT_KEY) || '[]');
    container.innerHTML = '';
    arr.forEach(({ text, kind }) => {
        const div = document.createElement('div');
        div.className = `recent-item ${kind || ''}`;
        div.textContent = text;
        container.appendChild(div);
    });
}
// Render on load
renderRecentOutcomes();

// ---------------- Blackjack (stateless) ----------------
// Longer post-result reset just for Blackjack so players can review results
const BLACKJACK_RESET_DELAY_MS = 4000;
let blackjackState = {
    amount: 0,
    dealerUpcard: '',
    playerHand: [],
    state: null
};

function blackjackStart() {
    // Defensive: ensure history shows last hand only and isn't used as a target
    const hist = document.getElementById('blackjack-history');
    if (hist && hist.children.length > 1) {
        // keep only the newest summary if somehow multiple exist
        while (hist.children.length > 1) hist.removeChild(hist.lastChild);
    }
    // Also ensure the live table exists and is clean before dealing
    const tbl = document.getElementById('blackjack-table');
    if (!tbl || tbl.children.length === 0) {
        resetBlackjackUI();
    }
    const amount = parseInt(document.getElementById('blackjack-amount').value);
    if (!amount || amount <= 0) {
        showResult('blackjack', 'Invalid bet amount', false);
        return;
    }
    // Disable start to prevent double submits
    document.getElementById('blackjack-start').disabled = true;

    fetch('/user/casino/blackjack/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ amount })
    })
    .then(r => r.json())
    .then(data => {
        if (data.extortion_blessed) { if (window.toast && window.toast.info) window.toast.info(data.message || 'A calm wind redirects your steps.', 5000); return; }
        if (data.extortion) { console.log('[Extortion Debug] blackjack start: TRIGGERED'); showExtortionModal(data); return; }
        if (data.natural_blackjack) {
            if (window.toast && window.toast.success) window.toast.success(`Blackjack! +${data.payout.toLocaleString()} credits`, 6000);
            if (typeof data.new_balance === 'number') updateCreditsDisplay(data.new_balance);
            const up = document.getElementById('dealer-upcard');
            up.classList.remove('placeholder'); up.classList.add('revealed'); up.innerHTML = `<span>${data.dealer_upcard}</span>`;
            const p1 = document.getElementById('player-card-1');
            const p2 = document.getElementById('player-card-2');
            p1.classList.remove('placeholder'); p1.classList.add('revealed'); p1.innerHTML = `<span>${data.player_hand[0]}</span>`;
            p2.classList.remove('placeholder'); p2.classList.add('revealed'); p2.innerHTML = `<span>${data.player_hand[1]}</span>`;
            p1.classList.add('win-outline');
            p2.classList.add('win-outline');
            pushRecentOutcome(`Blackjack! +${data.payout.toLocaleString()}`, 'blackjack');
            resetBlackjackUI();
            return;
        }
        console.log('[Extortion Debug] blackjack start: clear');
        if (!data.success) {
            showResult('blackjack', 'Error: ' + data.error, false);
            document.getElementById('blackjack-start').disabled = false;
            return;
        }
        blackjackState.amount = data.amount;
        blackjackState.dealerUpcard = data.dealer_upcard;
        blackjackState.playerHand = data.player_hand;
        blackjackState.state = data.state;

        // Update UI
        const up = document.getElementById('dealer-upcard');
        up.classList.remove('placeholder');
        up.classList.add('revealed');
        up.innerHTML = `<span>${data.dealer_upcard}</span>`;

        const p1 = document.getElementById('player-card-1');
        const p2 = document.getElementById('player-card-2');
        p1.classList.remove('placeholder'); p1.classList.add('revealed'); p1.innerHTML = `<span>${data.player_hand[0]}</span>`;
        p2.classList.remove('placeholder'); p2.classList.add('revealed'); p2.innerHTML = `<span>${data.player_hand[1]}</span>`;

        // Show Hit/Stand
        document.getElementById('blackjack-start-group').classList.add('hidden');
        document.getElementById('blackjack-play-group').classList.remove('hidden');
        if (window.toast && window.toast.info) window.toast.info('Cards dealt. Hit or Stand?', 2500);
    })
    .catch(() => {
        showResult('blackjack', 'Network error', false);
        document.getElementById('blackjack-start').disabled = false;
    });
}

function blackjackHit() {
    // Disable buttons during request
    document.getElementById('blackjack-hit').disabled = true;
    document.getElementById('blackjack-stand').disabled = true;
    let roundEnded = false;

    fetch('/user/casino/blackjack/hit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ state: blackjackState.state })
    })
    .then(r => r.json())
    .then(data => {
        if (data.extortion_blessed) { if (window.toast && window.toast.info) window.toast.info(data.message || 'A calm wind redirects your steps.', 5000); return; }
        if (data.extortion) { console.log('[Extortion Debug] blackjack hit: TRIGGERED'); showExtortionModal(data); return; }
        console.log('[Extortion Debug] blackjack hit: clear');
        if (!data.success) {
            showResult('blackjack', 'Error: ' + (data.error || 'Unknown'), false);
            return;
        }
        blackjackState.playerHand = data.player_hand;
        if (data.state) {
            blackjackState.state = data.state;
        }
        // Render newest card into a new slot element
        const table = document.getElementById('blackjack-table');
        // Append new card to live table only (never to history)
        if (table) {
            const cardDiv = document.createElement('div');
            cardDiv.className = 'card revealed';
            cardDiv.innerHTML = `<span>${data.new_card}</span>`;
            table.appendChild(cardDiv);
        }

        if (data.bust) {
            if (window.toast && window.toast.error) window.toast.error(`Bust at ${data.player_total}. -${blackjackState.amount.toLocaleString()} credits`, 5000);
            pushRecentOutcome(`You lost -${blackjackState.amount.toLocaleString()}`, 'loss');
            // Snapshot losing hand to history
            const history = document.getElementById('blackjack-history');
            if (history) history.innerHTML = '';
            const summary = document.createElement('div');
            summary.className = 'hand-summary';
            const dealerWrap = document.createElement('div');
            const playerWrap = document.createElement('div');
            // Current table cards include dealer upcard + separator + current player cards
            const table = document.getElementById('blackjack-table');
            const cards = Array.from(table.querySelectorAll('.card.revealed'));
            const dealerCard = cards[0]; if (dealerCard) { const c = dealerCard.cloneNode(true); c.classList.add('mini'); dealerWrap.appendChild(c); }
            blackjackState.playerHand.forEach(() => {});
            Array.from(table.querySelectorAll('.card.revealed')).slice(1).forEach(el => { const c = el.cloneNode(true); c.classList.add('mini'); playerWrap.appendChild(c); });
            playerWrap.querySelectorAll('.card').forEach(el => el.classList.add('lose-muted'));
            summary.appendChild(dealerWrap); const sep = document.createElement('div'); sep.className='vs-text'; sep.textContent='DEALER'; summary.appendChild(sep); summary.appendChild(playerWrap);
            if (history) history.prepend(summary);
            resetBlackjackUI();
            roundEnded = true;
        } else {
            if (window.toast && window.toast.info) window.toast.info(`Total: ${data.player_total}. Hit or Stand?`, 2000);
        }
    })
    .catch(() => {
        showResult('blackjack', 'Network error', false);
    })
    .finally(() => {
        if (!roundEnded) {
            document.getElementById('blackjack-hit').disabled = false;
            document.getElementById('blackjack-stand').disabled = false;
        }
    });
}

function blackjackStand() {
    // Disable buttons during request
    document.getElementById('blackjack-hit').disabled = true;
    document.getElementById('blackjack-stand').disabled = true;
    let roundEnded = false;

    fetch('/user/casino/blackjack/stand', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ state: blackjackState.state })
    })
    .then(r => r.json())
    .then(data => {
        if (data.extortion_blessed) { if (window.toast && window.toast.info) window.toast.info(data.message || 'A calm wind redirects your steps.', 5000); return; }
        if (data.extortion) { console.log('[Extortion Debug] blackjack stand: TRIGGERED'); showExtortionModal(data); return; }
        console.log('[Extortion Debug] blackjack stand: clear');
        if (!data.success) {
            showResult('blackjack', 'Error: ' + (data.error || 'Unknown'), false);
            return;
        }
        // Render dealer final hand additions
        const table = document.getElementById('blackjack-table');
        // Remove all existing dealer upcard element and re-render dealer hand in place of first slot
        table.innerHTML = '';
        data.dealer_hand.forEach((c, idx) => {
            const d = document.createElement('div');
            d.className = 'card revealed';
            d.innerHTML = `<span>${c}</span>`;
            table.appendChild(d);
        });
        // append separator
        const sep = document.createElement('div');
        sep.className = 'vs-text';
        sep.textContent = 'DEALER';
        table.appendChild(sep);
        // render player hand
        blackjackState.playerHand.forEach(c => {
            const d = document.createElement('div');
            d.className = 'card revealed';
            d.innerHTML = `<span>${c}</span>`;
            table.appendChild(d);
        });

        const outcome = data.push ? 'PUSH' : (data.won ? 'YOU WIN!' : 'You lose.');
        const deltaNum = data.push ? 0 : (data.won ? data.payout : -blackjackState.amount);
        if (window.toast) {
            if (data.push && window.toast.info) window.toast.info(`Push. Dealer ${data.dealer_total}, You ${data.player_total}.`, 4500);
            else if (data.won && window.toast.success) window.toast.success(`You win! ${deltaNum.toLocaleString()} credits`, 5000);
            else if (!data.won && window.toast.error) window.toast.error(`You lose. ${deltaNum.toLocaleString()} credits`, 5000);
        }
        // Record outcome
        if (data.push) pushRecentOutcome('Push', 'push');
        else if (data.won) pushRecentOutcome(`You won +${deltaNum.toLocaleString()}`, 'win');
        else pushRecentOutcome(`You lost ${deltaNum.toLocaleString()}`, 'loss');

        // Winner highlight and snapshot to history
        const tableCards = Array.from(table.querySelectorAll('.card.revealed'));
        const dealerCount = data.dealer_hand.length;
        const dealerCards = tableCards.slice(0, dealerCount);
        // All remaining revealed cards after dealer belong to player; separator is not a .card
        const playerCards = tableCards.slice(dealerCount);
        const history = document.getElementById('blackjack-history');
        if (history) history.innerHTML = '';
        const summary = document.createElement('div');
        summary.className = 'hand-summary';
        const dealerWrap = document.createElement('div'); dealerWrap.className = 'hand-cards';
        const playerWrap = document.createElement('div'); playerWrap.className = 'hand-cards';
        dealerCards.forEach(el => {
            const c = el.cloneNode(true); c.classList.add('mini'); dealerWrap.appendChild(c);
        });
        const sepMini = document.createElement('div'); sepMini.className = 'vs-text'; sepMini.textContent = 'DEALER';
        playerCards.forEach(el => { const c = el.cloneNode(true); c.classList.add('mini'); playerWrap.appendChild(c); });
        if (!data.push) {
            if (data.won) { playerWrap.querySelectorAll('.card').forEach(el => el.classList.add('win-outline')); dealerWrap.querySelectorAll('.card').forEach(el => el.classList.add('lose-muted')); }
            else { dealerWrap.querySelectorAll('.card').forEach(el => el.classList.add('win-outline')); playerWrap.querySelectorAll('.card').forEach(el => el.classList.add('lose-muted')); }
        }
        summary.appendChild(dealerWrap); summary.appendChild(sepMini); summary.appendChild(playerWrap);
        if (history) history.prepend(summary);

        // Update credits and reset after short delay
        if (typeof data.new_balance === 'number') {
            updateCreditsDisplay(data.new_balance);
        }
        resetBlackjackUI();
        roundEnded = true;
    })
    .catch(() => {
        showResult('blackjack', 'Network error', false);
    })
    .finally(() => {
        if (!roundEnded) {
            document.getElementById('blackjack-hit').disabled = false;
            document.getElementById('blackjack-stand').disabled = false;
        }
    });
}

// Global variables to store Step 1 data for Step 2
let hiLowFirstCard = '';
let hiLowBetAmount = 0;

// Hi-Low Step 1: Place bet and get first card - ALL LOGIC SERVER-SIDE
function placeHiLowBetStep1() {
    const amountInput = document.getElementById('hilow-amount');
    const amount = parseInt(amountInput.value);
    const cap = parseInt(amountInput.max);
    
    if (!amount || amount <= 0) {
        showResult('hilow', 'Invalid bet amount', false);
        return;
    }
    if (cap && amount > cap) {
        showResult('hilow', `Max bet is ${cap.toLocaleString()}`, false);
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
        if (data.extortion_blessed) { if (window.toast && window.toast.info) window.toast.info(data.message || 'A calm wind redirects your steps.', 5000); return; }
        if (data.extortion) { console.log('[Extortion Debug] hilow step1: TRIGGERED'); showExtortionModal(data); return; }
        console.log('[Extortion Debug] hilow step1: clear');
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
    const amountInput = document.getElementById('moonflip-amount');
    const amount = parseInt(amountInput.value);
    const cap = parseInt(amountInput.max);
    
    if (!amount || amount <= 0) {
        showResult('moonflip', 'Invalid bet amount', false);
        return;
    }
    if (cap && amount > cap) {
        showResult('moonflip', `Max bet is ${cap.toLocaleString()}`, false);
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
        if (data.extortion_blessed) { if (window.toast && window.toast.info) window.toast.info(data.message || 'A calm wind redirects your steps.', 5000); return; }
        if (data.extortion) { console.log('[Extortion Debug] moonflip: TRIGGERED'); showExtortionModal(data); return; }
        console.log('[Extortion Debug] moonflip: clear');
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
    const amountInput = document.getElementById('slots-amount');
    const amount = parseInt(amountInput.value);
    const cap = parseInt(amountInput.max);
    
    if (!amount || amount <= 0) {
        showResult('slots', 'Invalid bet amount', false);
        return;
    }
    if (cap && amount > cap) {
        showResult('slots', `Max bet is ${cap.toLocaleString()}`, false);
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
        if (data.extortion_blessed) { if (window.toast && window.toast.info) window.toast.info(data.message || 'A calm wind redirects your steps.', 5000); return; }
        if (data.extortion) { console.log('[Extortion Debug] slots single: TRIGGERED'); showExtortionModal(data); return; }
        console.log('[Extortion Debug] slots single: clear');
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

// Secure multi-spin: performs N sequential spins via backend API.
// Shows only the final result (and updates jackpot/credits incrementally).
function spinSlotsSeries(times) {
    const spinBtn = document.getElementById('slots-spin-btn');
    const spin10Btn = document.getElementById('slots-spin10-btn');
    if (spinBtn) spinBtn.disabled = true;
    if (spin10Btn) spin10Btn.disabled = true;

    const amountInput = document.getElementById('slots-amount');
    const cap = parseInt(amountInput.max);
    const perSpin = parseInt(amountInput.value);
    if (!perSpin || perSpin <= 0 || (cap && perSpin > cap)) {
        showResult('slots', 'Invalid bet amount', false);
        if (spinBtn) spinBtn.disabled = false;
        if (spin10Btn) spin10Btn.disabled = false;
        return;
    }

    let remaining = times;
    let lastData = null;
    let wins = 0;
    let totalLines = 0;
    let maxLines = 0;
    let netCredits = 0;

    // Start continuous spin animation and UI hints
    startSpinning();
    const step1 = document.getElementById('slots-step1-text');
    const step2 = document.getElementById('slots-step2-text');
    if (step1) step1.classList.remove('active');
    if (step2) step2.classList.add('active');
    const updateBtnLabel = () => { if (spinBtn) spinBtn.textContent = `🎰 SPINNING... (${times-remaining+1}/${times})`; };
    updateBtnLabel();

    // Run strictly sequentially using a promise chain to avoid parallel requests
    const notifySuccess = (msg, dur) => { try { if (window.toast && window.toast.success) window.toast.success(msg, dur); } catch (e) {} };
    const notifyWarning = (msg, dur) => { try { if (window.toast && window.toast.warning) window.toast.warning(msg, dur); } catch (e) {} };

    const runSequentially = () => {
        return fetch('/user/casino/slots', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ amount: perSpin })
        })
        .then(r => r.json())
        .then(data => {
            if (data.extortion_blessed) { if (window.toast && window.toast.info) window.toast.info(data.message || 'A calm wind redirects your steps.', 5000); throw new Error('extortion'); }
            if (data.extortion) { console.log(`[Extortion Debug] slots series spin ${times-remaining+1}/${times}: TRIGGERED`); showExtortionModal(data); throw new Error('extortion'); }
            console.log(`[Extortion Debug] slots series spin ${times-remaining+1}/${times}: clear`);
            if (!data.success) throw new Error(data.error || 'Spin failed');
            lastData = data;
            const linesLen = Array.isArray(data.winning_lines) ? data.winning_lines.length : 0;
            if (data.won) wins++;
            totalLines += linesLen;
            if (linesLen > maxLines) maxLines = linesLen;
            netCredits += (data.won ? (data.payout - data.amount) : (-data.amount));
            // Toast per spin outcome (compact)
            if (data.won) {
                notifySuccess(`Slots win: +${data.payout.toLocaleString()} (${linesLen} line${linesLen===1?'':'s'})`, 4000);
            } else {
                notifyWarning(`No win: -${data.amount.toLocaleString()}`, 1500);
            }
            if (typeof data.new_balance === 'number') updateCreditsDisplay(data.new_balance);
            loadProgressiveJackpot();
            remaining--;
            if (spinBtn) updateBtnLabel();
            if (remaining > 0) {
                // slight delay between spins to keep CPU/network tame
                return new Promise(res => setTimeout(res, 120)).then(runSequentially);
            }
        });
    };

    runSequentially().catch(() => {}).finally(() => {
        if (lastData) {
            // Reveal the final grid, then show summary toast
            animateSlotSequences(lastData.sequences, lastData.final_grid, lastData.winning_lines, lastData.won, lastData.payout, lastData.amount, lastData.new_balance);
            const summary = `${wins}/${times} wins • best lines: ${maxLines} • net: ${(netCredits>=0?'+':'')}${netCredits.toLocaleString()} credits`;
            showInfo(`${summary}`, 10000);
        }
        stopSpinning();
        if (spinBtn) spinBtn.disabled = false;
        if (spin10Btn) spin10Btn.disabled = false;
        if (spinBtn) spinBtn.textContent = '🎰 SPIN THE REELS';
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
                
                // Legacy under-button result banner removed in favor of toasts/summary
                
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
        const full = (typeof newBalance === 'number') ? newBalance : parseInt(newBalance, 10) || 0;
        creditsElement.textContent = formatLargeNumber(full);
        creditsElement.title = full.toLocaleString();
    }
}

function showExtortionModal(payload) {
    const modal = document.getElementById('extortion-modal');
    if (!modal) return;
    if (payload) {
        const fmt = (n) => (typeof n === 'number' ? n.toLocaleString() : '—');
        const orig = modal.querySelector('#ext-original');
        const hold = modal.querySelector('#ext-hold');
        const fee = modal.querySelector('#ext-fee');
        const payLabel = modal.querySelector('#ext-pay-label');
        if (orig && typeof payload.original_balance === 'number') orig.textContent = `${fmt(payload.original_balance)} credits`;
        if (hold && typeof payload.hold === 'number') hold.textContent = `${fmt(payload.hold)} credits`;
        if (fee && typeof payload.fee_amount === 'number') fee.textContent = `${fmt(payload.fee_amount)} credits`;
        if (payLabel && typeof payload.fee_amount === 'number') payLabel.textContent = `Pay ${fmt(payload.fee_amount)} credits`;
    }
    // Ensure action buttons are clickable
    const payBtn = document.getElementById('extortion-pay');
    const runBtn = document.getElementById('extortion-run');
    if (payBtn) payBtn.disabled = false;
    if (runBtn) runBtn.disabled = false;
    modal.classList.remove('hidden');
}

function resolveExtortion(choice) {
    // Disable buttons to prevent double submit
    const payBtn = document.getElementById('extortion-pay');
    const runBtn = document.getElementById('extortion-run');
    if (payBtn) payBtn.disabled = true;
    if (runBtn) runBtn.disabled = true;
    fetch('/user/casino/extortion', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ choice })
    })
    .then(r => r.json())
    .then(data => {
        if (data.success) {
            // Compose flavorful outcome message
            const kept = (data.new_balance !== undefined && data.new_balance !== null)
                ? Number(data.new_balance)
                : undefined;
            let msg = '';
            let kind = 'info';
            if (data.outcome === 'paid') {
                msg = `You pay the fee and keep walking. Net -20%. You kept: ${kept?.toLocaleString?.() || kept} credits.`;
                kind = 'warning';
            } else if (data.outcome === 'run_success') {
                msg = `You run like the wind. You slip away with everything. You kept: ${kept?.toLocaleString?.()} credits.`;
                kind = 'success';
            } else if (data.outcome === 'run_fail') {
                msg = `You tried to run. They clipped your wings. Net -50%. You kept: ${kept?.toLocaleString?.()} credits.`;
                kind = 'error';
            }
            try {
                if (window.toast) {
                    const d = 7000;
                    if (kind === 'success' && window.toast.success) window.toast.success(msg, d);
                    else if (kind === 'warning' && window.toast.warning) window.toast.warning(msg, d);
                    else if (kind === 'error' && window.toast.error) window.toast.error(msg, d);
                    else if (window.toast.info) window.toast.info(msg, d);
                }
            } catch (_) {}
            if (data.new_balance !== undefined && data.new_balance !== null) {
                updateCreditsDisplay(Number(data.new_balance));
            }
            // Close modal and restore UI without reloading
            const modal = document.getElementById('extortion-modal');
            if (modal) modal.classList.add('hidden');
            // Attempt to restore common game controls
            try {
                stopSpinning();
                resetSpinButton();
            } catch (_) {}
            const spinBtn = document.getElementById('slots-spin-btn');
            const spin10Btn = document.getElementById('slots-spin10-btn');
            if (spinBtn) spinBtn.disabled = false;
            if (spin10Btn) spin10Btn.disabled = false;
            const bjStart = document.getElementById('blackjack-start');
            const bjHit = document.getElementById('blackjack-hit');
            const bjStand = document.getElementById('blackjack-stand');
            if (bjStart) bjStart.disabled = false;
            if (bjHit) bjHit.disabled = false;
            if (bjStand) bjStand.disabled = false;
            const placeBetButton = document.getElementById('place-hilow-bet');
            if (placeBetButton) placeBetButton.disabled = false;
            document.querySelectorAll('.bet-btn[data-choice]').forEach(btn => btn.disabled = false);
            document.querySelectorAll('.bet-btn[data-guess]').forEach(btn => btn.disabled = false);
        } else {
            try { if (window.toast && window.toast.error) window.toast.error('Extortion resolve failed', 4000); } catch (_) {}
            const modal = document.getElementById('extortion-modal');
            if (modal) modal.classList.add('hidden');
            // Do not reload; user can continue
        }
    })
    .catch(() => {
        window.location.reload();
    });
}

function updateAbbrevFor(game) {
    const input = document.getElementById(game + '-amount');
    const label = document.getElementById(game + '-amount-abbrev');
    if (!input || !label) return;
    const value = parseInt(input.value || '0', 10);
    if (!value || value <= 0) {
        label.textContent = '';
        label.removeAttribute('title');
        return;
    }
    label.textContent = `(${formatLargeNumber(value)})`;
    label.title = value.toLocaleString();
}

function loadProgressiveJackpot() {
    fetch('/user/casino/jackpot')
        .then(response => response.json())
        .then(data => {
            const jackpotDisplay = document.getElementById('jackpot-display');
            if (jackpotDisplay) {
                // Use abbreviated formatting similar to base.html credits
                const formatted = formatLargeNumber(data.jackpot);
                jackpotDisplay.textContent = formatted;
                jackpotDisplay.title = data.jackpot.toLocaleString();
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

// Convert large numbers to abbreviated format with decimal precision
// Mirrors the helper in templates/base.html
function formatLargeNumber(num) {
    if (typeof num !== 'number') {
        num = parseInt(num, 10) || 0;
    }
    if (num < 1000) {
        return num.toString();
    }
    const suffixes = [
        { value: 1e18, suffix: 'Qi' }, // Quintillion
        { value: 1e15, suffix: 'Qa' }, // Quadrillion
        { value: 1e12, suffix: 'T'  }, // Trillion
        { value: 1e9,  suffix: 'B'  }, // Billion
        { value: 1e6,  suffix: 'M'  }, // Million
        { value: 1e3,  suffix: 'K'  }  // Thousand
    ];
    for (let i = 0; i < suffixes.length; i++) {
        if (num >= suffixes[i].value) {
            const result = (num / suffixes[i].value).toFixed(2);
            return (result.endsWith('.00') ? result.slice(0, -3) : result) + suffixes[i].suffix;
        }
    }
    return num.toString();
}

// Abbreviate the displayed player credits in the casino header and add a tooltip with full value
function formatCasinoHeaderCredits() {
    const el = document.querySelector('.credits-amount');
    if (!el) return;
    const raw = el.textContent.replace(/[^0-9]/g, '');
    const value = parseInt(raw, 10);
    if (isNaN(value)) return;
    el.textContent = formatLargeNumber(value);
    el.title = value.toLocaleString();
}

function stopSpinning() {
    document.querySelectorAll('.slot-reel').forEach(reel => {
        reel.classList.remove('spinning');
    });
}

function startSpinning() {
    document.querySelectorAll('.slot-reel').forEach(reel => {
        reel.classList.add('spinning');
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