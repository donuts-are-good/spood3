// Fighter Creation Wizard JavaScript
let currentStep = 1;
const totalSteps = 4;

// Fighter data storage
let fighterData = {
    name: '',
    stats: {
        strength: 20,
        speed: 20,
        endurance: 20,
        technique: 20
    },
    chaosStats: {}
};

// Initialize the wizard when page loads
document.addEventListener('DOMContentLoaded', function() {
    initializeWizard();
});

function initializeWizard() {
    // Set up name input validation
    const nameInput = document.getElementById('fighter-name');
    if (nameInput) {
        nameInput.addEventListener('input', validateName);
    }
    
    // Initialize stat allocation
    updateStatDisplay();
    updateStatsNextButton();
}

// Navigation functions
function nextStep() {
    if (currentStep < totalSteps) {
        // Hide current step
        document.getElementById(`step-${getStepName(currentStep)}`).classList.remove('active');
        
        currentStep++;
        
        // Show next step
        document.getElementById(`step-${getStepName(currentStep)}`).classList.add('active');
        
        // Handle step-specific logic
        if (currentStep === 4) {
            populatePreview();
        }
    }
}

function prevStep() {
    if (currentStep > 1) {
        // Hide current step
        document.getElementById(`step-${getStepName(currentStep)}`).classList.remove('active');
        
        currentStep--;
        
        // Show previous step
        document.getElementById(`step-${getStepName(currentStep)}`).classList.add('active');
    }
}

function getStepName(step) {
    const stepNames = ['name', 'stats', 'chaos', 'preview'];
    return stepNames[step - 1];
}

// Name validation
function validateName() {
    const nameInput = document.getElementById('fighter-name');
    const validation = document.getElementById('name-validation');
    const nextBtn = document.getElementById('name-next-btn');
    
    const name = nameInput.value.trim();
    
    if (name.length < 3) {
        validation.textContent = 'Name must be at least 3 characters';
        validation.className = 'name-validation invalid';
        nextBtn.disabled = true;
        return false;
    } else if (name.length > 50) {
        validation.textContent = 'Name must be 50 characters or less';
        validation.className = 'name-validation invalid';
        nextBtn.disabled = true;
        return false;
    } else if (!/^[a-zA-Z0-9\s\-_'\.]+$/.test(name)) {
        validation.textContent = 'Name contains invalid characters';
        validation.className = 'name-validation invalid';
        nextBtn.disabled = true;
        return false;
    } else {
        validation.textContent = '✓ Name looks good!';
        validation.className = 'name-validation valid';
        fighterData.name = name;
        nextBtn.disabled = false;
        return true;
    }
}

// Stat allocation functions
function adjustStat(statName, delta) {
    const newValue = fighterData.stats[statName] + delta;
    const totalPoints = getTotalAllocatedPoints();
    const remainingPoints = 300 - totalPoints;
    
    // Check bounds
    if (newValue < 20 || newValue > 120) {
        return;
    }
    
    // Check if we have enough points to increase
    if (delta > 0 && remainingPoints < delta) {
        return;
    }
    
    // Apply the change
    fighterData.stats[statName] = newValue;
    updateStatDisplay();
    updateStatsNextButton();
}

function getTotalAllocatedPoints() {
    return Object.values(fighterData.stats).reduce((sum, value) => sum + value, 0);
}

function updateStatDisplay() {
    const totalPoints = getTotalAllocatedPoints();
    const remainingPoints = 300 - totalPoints;
    
    // Update remaining points display
    document.getElementById('remaining-points').textContent = remainingPoints;
    
    // Update each stat bar
    Object.keys(fighterData.stats).forEach(statName => {
        const value = fighterData.stats[statName];
        const percentage = (value / 120) * 100;
        
        const fillElement = document.getElementById(`${statName}-fill`);
        const valueElement = document.getElementById(`${statName}-value`);
        
        if (fillElement && valueElement) {
            fillElement.style.width = percentage + '%';
            valueElement.textContent = `${value}/120`;
        }
    });
}

function updateStatsNextButton() {
    const totalPoints = getTotalAllocatedPoints();
    const nextBtn = document.getElementById('stats-next-btn');
    
    if (nextBtn) {
        nextBtn.disabled = totalPoints !== 300;
    }
}

// Quick build presets
function applyPreset(presetName) {
    const presets = {
        balanced: { strength: 75, speed: 75, endurance: 75, technique: 75 },
        glass_cannon: { strength: 120, speed: 100, endurance: 20, technique: 60 },
        tank: { strength: 60, speed: 20, endurance: 120, technique: 100 },
        speedster: { strength: 40, speed: 120, endurance: 60, technique: 80 },
        technical: { strength: 50, speed: 70, endurance: 60, technique: 120 },
        bruiser: { strength: 110, speed: 60, endurance: 90, technique: 40 },
        agile: { strength: 60, speed: 110, endurance: 50, technique: 80 },
        sentinel: { strength: 80, speed: 40, endurance: 120, technique: 60 },
        scholar: { strength: 40, speed: 70, endurance: 70, technique: 120 }
    };
    
    if (presets[presetName]) {
        fighterData.stats = { ...presets[presetName] };
        updateStatDisplay();
        updateStatsNextButton();
    }
}

// Randomize a valid 300-point distribution within [20,120] for each stat
function randomizeStats() {
    const btn = document.getElementById('randomize-button');
    if (btn) {
        btn.disabled = true;
        btn.classList.add('spinning');
    }

    showRandomizeOverlay();

    const statNames = ["strength", "speed", "endurance", "technique"];
    const minPerStat = 20;
    const maxPerStat = 120;
    const total = 300;

    // Start each stat at minimum
    const base = { strength: minPerStat, speed: minPerStat, endurance: minPerStat, technique: minPerStat };
    let remaining = total - (minPerStat * statNames.length); // 220

    // Capacities left per stat
    const capacity = { strength: maxPerStat - minPerStat, speed: maxPerStat - minPerStat, endurance: maxPerStat - minPerStat, technique: maxPerStat - minPerStat };

    // Distribute remaining points randomly while respecting per-stat caps
    while (remaining > 0) {
        const name = statNames[Math.floor(Math.random() * statNames.length)];
        if (capacity[name] === 0) continue;
        const give = Math.min(1 + Math.floor(Math.random() * 5), capacity[name], remaining); // 1..5 at a time
        base[name] += give;
        capacity[name] -= give;
        remaining -= give;
    }

    fighterData.stats = base;
    // Animate numbers/sliders by updating after a short delay to simulate rolling
    setTimeout(() => {
        updateStatDisplay();
        updateStatsNextButton();
        hideRandomizeOverlay();
        if (btn) {
            btn.disabled = false;
            btn.classList.remove('spinning');
        }
    }, 600);
}

// simple overlay spinner for randomizing UX
function showRandomizeOverlay() {
    let overlay = document.getElementById('randomize-overlay');
    if (!overlay) {
        overlay = document.createElement('div');
        overlay.id = 'randomize-overlay';
        overlay.innerHTML = '<div class="spinner"></div>';
        document.body.appendChild(overlay);
    }
    overlay.style.display = 'flex';
}

function hideRandomizeOverlay() {
    const overlay = document.getElementById('randomize-overlay');
    if (overlay) overlay.style.display = 'none';
}

// Chaos stats generation
function generateChaosStats() {
    // Don't actually generate or show the chaos stats yet
    // Just indicate that the user has "rolled" and can proceed
    
    // Enable next button
    document.getElementById('chaos-next-btn').disabled = false;
    
    // Update UI to show that chaos stats are "ready" but not revealed
    document.querySelectorAll('.chaos-stat').forEach(stat => {
        stat.setAttribute('data-rarity', 'pending');
        const slot = stat.querySelector('.gacha-slot');
        slot.classList.add('generated');
        slot.textContent = '✅ READY';
    });
    
    // Disable generate button
    document.getElementById('generate-chaos').disabled = true;
    document.getElementById('generate-chaos').textContent = '✅ Chaos Stats Locked In!';
    
    // Show message about gacha reveal
    const rarityIndicator = document.getElementById('overall-rarity');
    rarityIndicator.style.display = 'block';
    rarityIndicator.innerHTML = '<span class="rarity-badge pending">🎲 STATS WILL BE REVEALED AFTER CREATION! 🎲</span>';
    
    // Store that chaos stats are ready (but don't generate actual values yet)
    fighterData.chaosStats = { ready: true };
}

function getRandomFromRarity(categories, targetRarity) {
    // Try to get from target rarity first, fall back if not available
    const rarities = ['legendary', 'rare', 'uncommon', 'common'];
    
    for (let rarity of rarities) {
        if (rarity === targetRarity && categories[rarity] && categories[rarity].length > 0) {
            return categories[rarity][Math.floor(Math.random() * categories[rarity].length)];
        }
    }
    
    // Fallback to common
    return categories.common[Math.floor(Math.random() * categories.common.length)];
}

function generateFingers(rarity) {
    switch (rarity) {
        case 'legendary':
            return [0, 1, 25, 30, 50, 100][Math.floor(Math.random() * 6)];
        case 'rare':
            return Math.floor(Math.random() * 21); // 0-20
        case 'uncommon':
            return Math.random() < 0.5 ? Math.floor(Math.random() * 2) + 6 : Math.floor(Math.random() * 3) + 13;
        default:
            return Math.floor(Math.random() * 5) + 8; // 8-12
    }
}

function generateToes(rarity) {
    switch (rarity) {
        case 'legendary':
            return [0, 1, 25, 30, 50, 100][Math.floor(Math.random() * 6)];
        case 'rare':
            return Math.floor(Math.random() * 21); // 0-20
        case 'uncommon':
            return Math.random() < 0.5 ? Math.floor(Math.random() * 2) + 6 : Math.floor(Math.random() * 3) + 13;
        default:
            return Math.floor(Math.random() * 5) + 8; // 8-12
    }
}

function generateMolecularDensity(rarity) {
    switch (rarity) {
        case 'legendary':
            return Math.random() < 0.5 ? 0.1 : 99.9;
        case 'rare':
            return Math.random() < 0.5 ? Math.random() * 10 : 90 + Math.random() * 9.9;
        case 'uncommon':
            return Math.random() < 0.5 ? 10 + Math.random() * 20 : 70 + Math.random() * 20;
        default:
            return 10 + Math.random() * 80; // 10-90
    }
}

function displayChaosStats() {
    const chaos = fighterData.chaosStats;
    
    // Update each chaos stat display
    document.getElementById('blood-type').textContent = chaos.bloodType;
    document.getElementById('horoscope').textContent = chaos.horoscope;
    document.getElementById('fingers').textContent = chaos.fingers;
    document.getElementById('toes').textContent = chaos.toes;
    document.getElementById('fighter-class').textContent = chaos.fighterClass;
    document.getElementById('molecular-density').textContent = chaos.molecularDensity.toFixed(1);
    
    // Update rarity indicators
    document.querySelectorAll('.chaos-stat').forEach(stat => {
        stat.setAttribute('data-rarity', chaos.rarity);
        stat.querySelector('.gacha-slot').classList.add('generated');
    });
}

// Preview population
function populatePreview() {
    // Update fighter name
    document.getElementById('fighter-name-preview').textContent = fighterData.name;
    
    // Update stats preview
    const statsGrid = document.getElementById('stats-preview');
    statsGrid.innerHTML = '';
    
    Object.entries(fighterData.stats).forEach(([statName, value]) => {
        const statDiv = document.createElement('div');
        statDiv.className = 'preview-stat';
        statDiv.innerHTML = `
            <label>${statName}</label>
            <value>${value}</value>
        `;
        statsGrid.appendChild(statDiv);
    });
    
    // Update chaos preview - don't show actual values yet!
    const chaosGrid = document.getElementById('chaos-preview');
    chaosGrid.innerHTML = '';
    
    if (fighterData.chaosStats && fighterData.chaosStats.ready) {
        const chaosItems = [
            'Blood Type', 'Horoscope', 'Fighter Class', 
            'Molecular Density', 'Fingers', 'Toes'
        ];
        
        chaosItems.forEach(itemName => {
            const chaosDiv = document.createElement('div');
            chaosDiv.className = 'preview-chaos';
            chaosDiv.innerHTML = `
                <label>${itemName.toLowerCase()}</label>
                <value>🎲 READY TO REVEAL</value>
            `;
            chaosGrid.appendChild(chaosDiv);
        });
    }
}

// Final fighter creation
function createFighter() {
    // Show loading state
    const createBtn = document.querySelector('.wizard-btn.primary');
    createBtn.disabled = true;
    createBtn.textContent = 'Creating Fighter...';
    
    // Send data to server
    fetch('/user/create-fighter', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(fighterData)
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            // Redirect to success page or fighter profile
            window.location.href = `/fighter/${data.fighter_id}`;
        } else {
            showError('Error creating fighter: ' + data.error);
            createBtn.disabled = false;
            createBtn.textContent = '🥊 Create Fighter & Use License!';
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showError('Error creating fighter. Please try again.');
        createBtn.disabled = false;
        createBtn.textContent = '🥊 Create Fighter & Use License!';
    });
} 