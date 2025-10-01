// VIOLENCE THEATER - Live Fight Viewer
// WebSocket connection for live updates
let socket;
let fightID;
let connectionAttempts = 0;
const maxConnectionAttempts = 3;

// Clapping system variables
let isUserLoggedIn = false;
let currentRound = 1;
let isClappingEnabled = false;
let clapCount = 0;
let clapResetTime = Date.now();
let originalFighter1AvatarHTML = '';
let originalFighter2AvatarHTML = '';

// Bouncing clap system
let bouncingClaps = [];
let clapAnimationRAF = null;
let lastClapAnimTime = 0;

function spawnBouncingClap(avatarElement) {
    const clap = document.createElement('div');
    clap.className = 'bouncing-clap';
    clap.textContent = '👏';
    // Randomize size slightly for variety
    const sizePx = 22 + Math.floor(Math.random() * 14); // 22–36px
    clap.style.fontSize = sizePx + 'px';
    clap.style.opacity = '1';
    document.body.appendChild(clap);

    // Starting position: center of avatar
    const avatarRect = avatarElement.getBoundingClientRect();
    const clapRect = clap.getBoundingClientRect();
    let x = avatarRect.left + avatarRect.width / 2 - clapRect.width / 2;
    let y = avatarRect.top + avatarRect.height / 2 - clapRect.height / 2;

    // Initial velocity like DVD logo
    const speed = 140 + Math.random() * 120; // px/sec
    // Pick a random direction that's not too axis-aligned
    let angle = Math.random() * Math.PI * 2;
    const minSinCos = 0.2;
    let vx = Math.cos(angle);
    let vy = Math.sin(angle);
    if (Math.abs(vx) < minSinCos) vx = Math.sign(vx || 1) * minSinCos;
    if (Math.abs(vy) < minSinCos) vy = Math.sign(vy || 1) * minSinCos;
    // Normalize
    const mag = Math.hypot(vx, vy) || 1;
    vx = (vx / mag) * speed;
    vy = (vy / mag) * speed;

    // Sync initial transform
    clap.style.transform = `translate(${Math.round(x)}px, ${Math.round(y)}px)`;

    const item = {
        el: clap,
        x,
        y,
        vx,
        vy,
        w: clapRect.width,
        h: clapRect.height,
        fading: false,
        removeTimeoutId: null
    };
    bouncingClaps.push(item);

    // Start animation loop if not running
    if (clapAnimationRAF === null) {
        lastClapAnimTime = performance.now();
        clapAnimationRAF = requestAnimationFrame(animateBouncingClaps);
    }
}

function animateBouncingClaps(now) {
    const dtMs = Math.min(50, now - lastClapAnimTime); // cap to avoid huge jumps
    const dt = dtMs / 1000;
    lastClapAnimTime = now;

    const viewportW = window.innerWidth;
    const viewportH = window.innerHeight;

    for (let i = 0; i < bouncingClaps.length; i++) {
        const c = bouncingClaps[i];
        if (!c || c.fading) continue;

        c.x += c.vx * dt;
        c.y += c.vy * dt;

        // Bounce on edges
        if (c.x <= 0) {
            c.x = 0;
            c.vx = Math.abs(c.vx);
        } else if (c.x + c.w >= viewportW) {
            c.x = viewportW - c.w;
            c.vx = -Math.abs(c.vx);
        }

        if (c.y <= 0) {
            c.y = 0;
            c.vy = Math.abs(c.vy);
        } else if (c.y + c.h >= viewportH) {
            c.y = viewportH - c.h;
            c.vy = -Math.abs(c.vy);
        }

        c.el.style.transform = `translate(${Math.round(c.x)}px, ${Math.round(c.y)}px)`;
    }

    // Cull removed entries
    if (bouncingClaps.length > 0) {
        clapAnimationRAF = requestAnimationFrame(animateBouncingClaps);
    } else {
        clapAnimationRAF = null;
    }
}

function fadeOutAllBouncingClaps() {
    for (let i = 0; i < bouncingClaps.length; i++) {
        const c = bouncingClaps[i];
        if (!c || c.fading) continue;
        c.fading = true;
        c.el.classList.add('fade-out');
        // Remove after transition
        c.removeTimeoutId = setTimeout(() => {
            if (c.el && c.el.parentNode) {
                c.el.parentNode.removeChild(c.el);
            }
            // Mark for removal by filtering
        }, 650);
    }
    // Periodically clean up array
    setTimeout(() => {
        bouncingClaps = bouncingClaps.filter(c => c && c.fading && c.el && document.body.contains(c.el));
        // After another tick, clear completely
        setTimeout(() => {
            for (let i = 0; i < bouncingClaps.length; i++) {
                const c = bouncingClaps[i];
                if (c.el && c.el.parentNode) c.el.parentNode.removeChild(c.el);
                if (c.removeTimeoutId) clearTimeout(c.removeTimeoutId);
            }
            bouncingClaps = [];
        }, 300);
    }, 200);
}

function initializeWatchPage(id, fightData) {
    fightID = id;
    connectionAttempts = 0;
    
    // Check if user is logged in by looking for user-specific elements
    isUserLoggedIn = document.querySelector('.user-bet-status') !== null || 
                     document.querySelector('[data-user-logged-in]') !== null;
    
    console.log('Initializing Violence Theater for fight:', fightID, 'with data:', fightData);
    console.log('User logged in:', isUserLoggedIn);
    
    // If this is a completed fight, show the results immediately
    if (fightData && fightData.status === 'completed') {
        showCompletedFightResults(fightData);
    } else if (fightData && fightData.status === 'voided') {
        showVoidedFightResults(fightData);
    }
    
    // Set up clapping for logged-in users
    if (isUserLoggedIn) {
        setupClapping();
    }
    
    // Initialize betting summary meter if present
    try {
        initializeBetSummary();
    } catch (e) {
        // no-op
    }

    // Always try to connect for live updates (if any)
    connectWebSocket();
}

function setupClapping() {
    const fighter1Avatar = document.getElementById('fighter1-avatar');
    const fighter2Avatar = document.getElementById('fighter2-avatar');
    
    if (fighter1Avatar) {
        // Preserve the full original avatar markup (could be an <img> or emoji)
        originalFighter1AvatarHTML = fighter1Avatar.innerHTML;
        fighter1Avatar.addEventListener('click', () => {
            const fighterID = fighter1Avatar.getAttribute('data-fighter-id');
            const fighterName = fighter1Avatar.getAttribute('data-fighter-name');
            handleClap(fighterID, fighterName, fighter1Avatar);
        });
    }
    
    if (fighter2Avatar) {
        // Preserve the full original avatar markup (could be an <img> or emoji)
        originalFighter2AvatarHTML = fighter2Avatar.innerHTML;
        fighter2Avatar.addEventListener('click', () => {
            const fighterID = fighter2Avatar.getAttribute('data-fighter-id');
            const fighterName = fighter2Avatar.getAttribute('data-fighter-name');
            handleClap(fighterID, fighterName, fighter2Avatar);
        });
    }
}

function handleClap(fighterID, fighterName, avatarElement) {
    // Get the actual current round from the DOM
    const roundElement = document.getElementById('round-number');
    const actualCurrentRound = parseInt(roundElement?.textContent || '1');
    
    // Check if clapping is enabled (round divisible by 5 and user logged in)
    const shouldBeEnabled = (actualCurrentRound % 5 === 0) && isUserLoggedIn;
    
    if (!shouldBeEnabled) {
        return;
    }
    
    // Rate limiting: 10 claps per second
    const now = Date.now();
    if (now - clapResetTime >= 1000) {
        // Reset counter every second
        clapCount = 0;
        clapResetTime = now;
    }
    
    if (clapCount >= 10) {
        // Silently ignore extra claps - backend will handle rate limiting
        return;
    }
    
    clapCount++;
    
    // Send clap via WebSocket
    if (socket && socket.readyState === WebSocket.OPEN) {
        const clapMessage = {
            type: 'clap',
            fighter_id: parseInt(fighterID),
            fighter_name: fighterName,
            round: actualCurrentRound
        };
        socket.send(JSON.stringify(clapMessage));
        
        // Show local feedback
        showClapNotification(avatarElement, '👏 +20');
        
        // Add clap burst animation
        avatarElement.classList.add('clap-burst');
        setTimeout(() => {
            avatarElement.classList.remove('clap-burst');
        }, 300);

        // Spawn a bouncing clap emoji
        spawnBouncingClap(avatarElement);
    }
}

function showClapNotification(avatarElement, text) {
    const notification = document.createElement('div');
    notification.className = 'clap-notification';
    notification.textContent = text;
    
    // Position relative to avatar
    const rect = avatarElement.getBoundingClientRect();
    notification.style.position = 'fixed';
    notification.style.left = rect.left + rect.width / 2 + 'px';
    notification.style.top = rect.top + 'px';
    notification.style.transform = 'translateX(-50%)';
    
    document.body.appendChild(notification);
    
    // Remove after animation
    setTimeout(() => {
        if (notification.parentNode) {
            notification.parentNode.removeChild(notification);
        }
    }, 2000);
}

function updateClappingState(round) {
    currentRound = round;
    const wasEnabled = isClappingEnabled;
    isClappingEnabled = (round % 5 === 0) && isUserLoggedIn;
    
    const fighter1Avatar = document.getElementById('fighter1-avatar');
    const fighter2Avatar = document.getElementById('fighter2-avatar');
    
    if (isClappingEnabled && !wasEnabled) {
        // Enable clapping - change avatars to clap hands
        if (fighter1Avatar) {
            fighter1Avatar.innerHTML = '👏';
            fighter1Avatar.classList.add('clappable');
        }
        if (fighter2Avatar) {
            fighter2Avatar.innerHTML = '👏';
            fighter2Avatar.classList.add('clappable');
        }
        
        // Show notification about clapping
        addCommentaryMessage({
            action: `🎉 ROUND ${round} - CROWD PARTICIPATION ENABLED! 🎉`,
            commentary: 'Cheer for your fighters! Click their avatars to give them +20 health!',
            announcer: 'THE COMMISSIONER',
            type: 'clap_enabled'
        });
        
    } else if (!isClappingEnabled && wasEnabled) {
        // Disable clapping - restore original avatars
        if (fighter1Avatar) {
            fighter1Avatar.innerHTML = originalFighter1AvatarHTML;
            fighter1Avatar.classList.remove('clappable');
        }
        if (fighter2Avatar) {
            fighter2Avatar.innerHTML = originalFighter2AvatarHTML;
            fighter2Avatar.classList.remove('clappable');
        }

        // Fade out any active bouncing claps
        fadeOutAllBouncingClaps();
    }
}

function showCompletedFightResults(fightData) {
    console.log('Showing completed fight results:', fightData);
    
    // Update status displays
    document.getElementById('commentary-status').textContent = '🏛️ VIOLENCE ARCHIVE';
    document.getElementById('fight-status').textContent = '✅ VIOLENCE COMPLETE';
    document.getElementById('round-number').textContent = 'FINAL';
    
    // Update health bars with final values
    updateHealthBar('health1', fightData.finalHealth1);
    updateHealthBar('health2', fightData.finalHealth2);
    
    // Determine winner message
    let winnerMessage = "Violence has concluded. The chaos gods are satisfied.";
    if (fightData.winnerID) {
        if (fightData.winnerID === fightData.fighter1ID) {
            winnerMessage = `🏆 ${fightData.fighter1Name} emerged victorious from the carnage! 🏆`;
        } else if (fightData.winnerID === fightData.fighter2ID) {
            winnerMessage = `🏆 ${fightData.fighter2Name} emerged victorious from the carnage! 🏆`;
        }
    } else {
        winnerMessage = "💀 BOTH FIGHTERS DIED IN MUTUAL DESTRUCTION! 💀";
    }
    
    // Add completion message to commentary
    const feed = document.getElementById('commentary-feed');
    const messageDiv = document.createElement('div');
    messageDiv.className = 'commentary-message completion-message';
    
    const actionDiv = document.createElement('div');
    actionDiv.className = 'action-text';
    actionDiv.textContent = winnerMessage;
    
    const commentDiv = document.createElement('div');
    commentDiv.className = 'announcer-comment';
    commentDiv.innerHTML = '<span class="announcer-name">"Screaming" Sally Bloodworth:</span> "The blood has been spilled! The violence debt has been paid!"';
    
    messageDiv.appendChild(actionDiv);
    messageDiv.appendChild(commentDiv);
    
    // Replace the initial message
    feed.innerHTML = '';
    feed.appendChild(messageDiv);
}

function showVoidedFightResults(fightData) {
    console.log('Showing voided fight results:', fightData);
    
    // Update status displays
    document.getElementById('commentary-status').textContent = '⚰️ ABSORBED BY THE VOID';
    document.getElementById('fight-status').textContent = '❌ VOIDED';
    document.getElementById('round-number').textContent = 'VOID';
    
    // Add void message to commentary
    const feed = document.getElementById('commentary-feed');
    const messageDiv = document.createElement('div');
    messageDiv.className = 'commentary-message void-message';
    
    const actionDiv = document.createElement('div');
    actionDiv.className = 'action-text';
    actionDiv.textContent = "⚰️ This violence was absorbed by the chaos void. ⚰️";
    
    const commentDiv = document.createElement('div');
    commentDiv.className = 'announcer-comment';
    commentDiv.innerHTML = '<span class="announcer-name">THE COMMISSIONER:</span> "This violence never occurred. It has been erased from the Department records."';
    
    messageDiv.appendChild(actionDiv);
    messageDiv.appendChild(commentDiv);
    
    // Replace the initial message
    feed.innerHTML = '';
    feed.appendChild(messageDiv);
}

function connectWebSocket() {
    if (!fightID) {
        console.error('Fight ID not set');
        return;
    }
    
    connectionAttempts++;
    console.log(`Attempting WebSocket connection #${connectionAttempts} for fight ${fightID}`);
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsURL = `${protocol}//${window.location.host}/ws/fight/${fightID}`;
    console.log('WebSocket URL:', wsURL);
    
    socket = new WebSocket(wsURL);
    
    socket.onopen = function(event) {
        console.log('✅ Connected to violence feed');
        document.getElementById('commentary-status').textContent = 'Connected to violence feed';
        connectionAttempts = 0; // Reset on successful connection
    };
    
    socket.onmessage = function(event) {
        console.log('📡 Received WebSocket message:', event.data);
        const data = JSON.parse(event.data);
        handleLiveUpdate(data);
    };
    
    socket.onclose = function(event) {
        console.log('❌ Disconnected from violence feed, code:', event.code, 'reason:', event.reason);
        
        if (connectionAttempts < maxConnectionAttempts) {
            document.getElementById('commentary-status').textContent = `Reconnecting... (${connectionAttempts}/${maxConnectionAttempts})`;
            // Attempt to reconnect after 3 seconds
            setTimeout(connectWebSocket, 3000);
        } else {
            console.log('Max connection attempts reached, giving up');
            document.getElementById('commentary-status').textContent = '⚠️ Connection failed - showing archived data';
        }
    };
    
    socket.onerror = function(error) {
        console.error('WebSocket error:', error);
        document.getElementById('commentary-status').textContent = '⚠️ Connection error';
    };
}

function handleLiveUpdate(data) {
    switch(data.type) {
        case 'initial':
            handleInitialState(data);
            break;
        case 'action':
            handleLiveAction(data.action);
            break;
        case 'viewer_count':
            updateViewerCount(data.viewer_count);
            break;
        case 'clap':
            updateClappingState(data.round);
            break;
    }
}

function handleInitialState(data) {
    if (data.status === 'scheduled') {
        document.getElementById('commentary-status').textContent = 'Violence begins soon!';
    } else if (data.status === 'active') {
        document.getElementById('commentary-status').textContent = '🔴 LIVE VIOLENCE IN PROGRESS';
        document.getElementById('fight-status').textContent = '🔴 LIVE VIOLENCE';
    } else if (data.status === 'completed') {
        document.getElementById('commentary-status').textContent = '🏛️ VIOLENCE ARCHIVE';
        document.getElementById('fight-status').textContent = '✅ VIOLENCE COMPLETE';
        
        // Update health bars with final values
        if (data.health1 !== undefined) {
            updateHealthBar('health1', data.health1);
        }
        if (data.health2 !== undefined) {
            updateHealthBar('health2', data.health2);
        }
        
        // Update round display
        if (data.round) {
            document.getElementById('round-number').textContent = data.round;
        }
        
        // Add completion message to commentary
        if (data.message) {
            const feed = document.getElementById('commentary-feed');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'commentary-message completion-message';
            
            const actionDiv = document.createElement('div');
            actionDiv.className = 'action-text';
            actionDiv.textContent = data.message;
            
            const commentDiv = document.createElement('div');
            commentDiv.className = 'announcer-comment';
            commentDiv.innerHTML = '<span class="announcer-name">"Screaming" Sally Bloodworth:</span> "The blood has been spilled! The violence debt has been paid!"';
            
            messageDiv.appendChild(actionDiv);
            messageDiv.appendChild(commentDiv);
            
            // Replace the initial message
            feed.innerHTML = '';
            feed.appendChild(messageDiv);
        }
    } else if (data.status === 'voided') {
        document.getElementById('commentary-status').textContent = '⚰️ ABSORBED BY THE VOID';
        document.getElementById('fight-status').textContent = '❌ VOIDED';
        
        // Add void message to commentary
        if (data.message) {
            const feed = document.getElementById('commentary-feed');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'commentary-message void-message';
            
            const actionDiv = document.createElement('div');
            actionDiv.className = 'action-text';
            actionDiv.textContent = data.message;
            
            const commentDiv = document.createElement('div');
            commentDiv.className = 'announcer-comment';
            commentDiv.innerHTML = '<span class="announcer-name">THE COMMISSIONER:</span> "This violence never occurred. It has been erased from the Department records."';
            
            messageDiv.appendChild(actionDiv);
            messageDiv.appendChild(commentDiv);
            
            // Replace the initial message
            feed.innerHTML = '';
            feed.appendChild(messageDiv);
        }
    }
}

// Update center-column betting summary based on existing bet counts if present
function initializeBetSummary() {
    const container = document.getElementById('bet-summary');
    if (!container) return;
    const f1 = parseInt(container.getAttribute('data-f1-bets') || '0', 10);
    const f2 = parseInt(container.getAttribute('data-f2-bets') || '0', 10);
    const total = Math.max(0, f1 + f2);
    const p1 = total > 0 ? Math.round((f1 / total) * 100) : 50;
    const p2 = 100 - p1;
    const fill1 = document.getElementById('bet-fill-1');
    const fill2 = document.getElementById('bet-fill-2');
    const p1El = document.getElementById('bet-p1');
    const p2El = document.getElementById('bet-p2');
    if (fill1) fill1.style.width = p1 + '%';
    if (fill2) fill2.style.width = p2 + '%';
    if (p1El) p1El.textContent = p1 + '%';
    if (p2El) p2El.textContent = p2 + '%';
}

function handleLiveAction(action) {
    // Update health bars
    updateHealthBar('health1', action.health1);
    updateHealthBar('health2', action.health2);
    
    // Update round number
    document.getElementById('round-number').textContent = action.round;
    
    // Always sync clapping state (in case UI got out of sync)
    updateClappingState(action.round);
    
    // Add to commentary feed
    addCommentaryMessage(action);
    
    // Special effects for different action types
    if (action.type === 'critical') {
        flashScreen('#ff4444');
    } else if (action.type === 'death') {
        flashScreen('#000000');
        document.getElementById('fight-status').textContent = '💀 FATALITY';
    }
}

function updateHealthBar(healthId, health) {
    const maxHealth = 100000;
    const percentage = Math.max(0, (health / maxHealth) * 100);
    
    const healthFill = document.getElementById(healthId);
    const healthText = document.getElementById(healthId + '-text');
    
    if (healthFill && healthText) {
        healthFill.style.width = percentage + '%';
        healthText.textContent = health.toLocaleString();
        
        // Color coding
        if (percentage > 60) {
            healthFill.style.backgroundColor = '#00ff00';
        } else if (percentage > 30) {
            healthFill.style.backgroundColor = '#ffaa00';
        } else {
            healthFill.style.backgroundColor = '#ff4444';
        }
    }
}

function addCommentaryMessage(action) {
    const feed = document.getElementById('commentary-feed');
    
    const messageDiv = document.createElement('div');
    
    const actionDiv = document.createElement('div');
    actionDiv.className = 'action-text';
    actionDiv.textContent = action.action;
    
    // Check if announcer has something to say
    if (action.commentary && action.commentary.trim() !== '') {
        // Normal message with announcer commentary
        messageDiv.className = 'commentary-message';
        
        const commentDiv = document.createElement('div');
        commentDiv.className = 'announcer-comment';
        commentDiv.innerHTML = `<span class="announcer-name">${action.announcer}:</span> "${action.commentary}"`;
        
        messageDiv.appendChild(actionDiv);
        messageDiv.appendChild(commentDiv);
    } else {
        // Action-only message when announcer has nothing to say
        messageDiv.className = 'commentary-message action-only-message';
        messageDiv.appendChild(actionDiv);
    }
    
    feed.appendChild(messageDiv);
    
    // Scroll to bottom
    feed.scrollTop = feed.scrollHeight;
    
    // Remove old messages if too many
    while (feed.children.length > 50) {
        feed.removeChild(feed.firstChild);
    }
}

function updateViewerCount(count) {
    document.getElementById('viewer-count').textContent = count;
}

function flashScreen(color) {
    const flash = document.createElement('div');
    flash.style.position = 'fixed';
    flash.style.top = '0';
    flash.style.left = '0';
    flash.style.width = '100%';
    flash.style.height = '100%';
    flash.style.backgroundColor = color;
    flash.style.opacity = '0.3';
    flash.style.pointerEvents = 'none';
    flash.style.zIndex = '9999';
    
    document.body.appendChild(flash);
    
    setTimeout(() => {
        document.body.removeChild(flash);
    }, 200);
} 