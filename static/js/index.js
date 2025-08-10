// Index page functionality - countdown timers, Sunday redirects, and weather advisory

// Chaos Weather Advisory dismissal
function dismissWeatherAdvisory() {
    const advisory = document.getElementById('chaosWeatherAdvisory');
    const indicator = document.getElementById('weatherIndicator');
    
    if (advisory) {
        // Hide the advisory
        advisory.style.display = 'none';
        // Store dismissal in localStorage so it stays dismissed for the session
        localStorage.setItem('weatherAdvisoryDismissed', 'true');
        
        // Show the weather indicator in the header
        if (indicator) {
            indicator.style.display = 'block';
        }
    }
}

// Restore weather advisory from header indicator
function restoreWeatherAdvisory() {
    const advisory = document.getElementById('chaosWeatherAdvisory');
    const indicator = document.getElementById('weatherIndicator');
    
    if (advisory) {
        // Show the advisory
        advisory.style.display = 'block';
        // Remove dismissal from localStorage
        localStorage.removeItem('weatherAdvisoryDismissed');
        
        // Hide the weather indicator in the header
        if (indicator) {
            indicator.style.display = 'none';
        }
    }
}

// Check if weather advisory should be shown and manage indicator visibility
function checkWeatherAdvisoryDismissal() {
    const advisory = document.getElementById('chaosWeatherAdvisory');
    const indicator = document.getElementById('weatherIndicator');
    
    if (advisory) {
        if (localStorage.getItem('weatherAdvisoryDismissed') === 'true') {
            // Advisory should be hidden
            advisory.style.display = 'none';
            
            // Show the weather indicator in the header
            if (indicator) {
                indicator.style.display = 'block';
            }
        } else {
            // Advisory should be showing
            advisory.style.display = 'block';
            
            // Hide the weather indicator in the header
            if (indicator) {
                indicator.style.display = 'none';
            }
        }
    }
}

// Countdown timer functionality
function initCountdown() {
    const countdownTimer = document.querySelector('.countdown-timer');
    if (!countdownTimer) return;
    
    const targetTime = new Date(countdownTimer.dataset.targetTime);
    const fightId = countdownTimer.dataset.fightId;
    const fighter1 = countdownTimer.dataset.fighter1;
    const fighter2 = countdownTimer.dataset.fighter2;
    const display = countdownTimer.querySelector('.countdown-display');
    
    function updateCountdown() {
        const now = new Date();
        const timeLeft = targetTime - now;
        
        if (timeLeft <= 0) {
            display.innerHTML = '🔥 <strong>VIOLENCE IS LIVE!</strong> 🔥';
            display.style.color = '#ff4444';
            display.style.fontSize = '1.2em';
            display.style.fontWeight = 'bold';
            
            // Check if fight is actually active and redirect to watch
            setTimeout(() => {
                window.location.href = `/watch/${fightId}`;
            }, 3000);
            
            return;
        }
        
        // Calculate time components
        const days = Math.floor(timeLeft / (1000 * 60 * 60 * 24));
        const hours = Math.floor((timeLeft % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        const minutes = Math.floor((timeLeft % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((timeLeft % (1000 * 60)) / 1000);
        
        // Format countdown display
        let countdownText = '';
        if (days > 0) {
            countdownText = `${days}d ${hours}h ${minutes}m ${seconds}s`;
        } else if (hours > 0) {
            countdownText = `${hours}h ${minutes}m ${seconds}s`;
        } else if (minutes > 0) {
            countdownText = `${minutes}m ${seconds}s`;
        } else {
            countdownText = `${seconds}s`;
            // Add urgency styling for final countdown
            if (seconds <= 30) {
                display.style.color = '#ff4444';
                display.style.fontSize = '1.1em';
                display.style.fontWeight = 'bold';
            }
        }
        
        display.textContent = countdownText;
    }
    
    // Update immediately and then every second
    updateCountdown();
    setInterval(updateCountdown, 1000);
}

// Check for Sunday closure and redirect at midnight CST6CDT
function checkSundayClosure() {
    const now = new Date();
    
    // Convert to CST/CDT - simpler approach
    // Create a date in Central Time
    const centralTime = new Date(now.toLocaleString("en-US", {timeZone: "America/Chicago"}));
    
    console.log('Current Central Time:', centralTime.toString());
    console.log('Day of week (0=Sunday):', centralTime.getDay());
    console.log('Hours:', centralTime.getHours(), 'Minutes:', centralTime.getMinutes());
    
    // Check if it's Sunday (0 = Sunday)
    if (centralTime.getDay() === 0) {
        // It's Sunday - redirect to closed page
        console.log('Sunday detected, redirecting to /closed');
        window.location.href = '/closed';
        return;
    }
    
    // Check if it's Saturday night approaching midnight
    if (centralTime.getDay() === 6) { // Saturday
        const timeUntilMidnight = new Date(centralTime);
        timeUntilMidnight.setDate(centralTime.getDate() + 1);
        timeUntilMidnight.setHours(0, 0, 0, 0);
        
        const msUntilMidnight = timeUntilMidnight.getTime() - centralTime.getTime();
        
        console.log('Saturday night - ms until midnight:', msUntilMidnight);
        
        // If less than 5 seconds until midnight, redirect
        if (msUntilMidnight <= 5000) {
            console.log('Less than 5 seconds until midnight, redirecting');
            window.location.href = '/closed';
            return;
        }
        
        // Set a timeout to redirect at midnight
        if (msUntilMidnight > 0) {
            console.log('Setting timeout for', msUntilMidnight, 'ms');
            setTimeout(() => {
                window.location.href = '/closed';
            }, msUntilMidnight);
        }
    }
}

// Initialize everything when page loads
document.addEventListener('DOMContentLoaded', function() {
    checkWeatherAdvisoryDismissal();
    initCountdown();
    checkSundayClosure();
    
    // Also check every 30 seconds in case user stays on page
    setInterval(checkSundayClosure, 30000);
}); 