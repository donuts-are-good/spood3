/**
 * Leaderboard JavaScript - Format large credit balances
 */

document.addEventListener('DOMContentLoaded', function() {
    formatCreditBalances();
    formatHeaderCredits();
});

/**
 * Format large numbers with abbreviated suffixes (k, m, b, t, q, etc.)
 */
function formatCreditBalances() {
    const creditElements = document.querySelectorAll('.credits-amount');
    
    creditElements.forEach(element => {
        const originalValue = parseInt(element.textContent.trim());
        if (!isNaN(originalValue)) {
            element.textContent = formatLargeNumber(originalValue);
        }
    });
}

/**
 * Convert large numbers to abbreviated format
 * @param {number} num - The number to format
 * @returns {string} - Formatted number with suffix
 */
function formatLargeNumber(num) {
    if (num < 1000) {
        return num.toString();
    }
    
    const suffixes = [
        { value: 1e18, suffix: 'Qi' }, // Quintillion (covers int64 max ~9.22 quintillion)
        { value: 1e15, suffix: 'Qa' }, // Quadrillion  
        { value: 1e12, suffix: 'T' },  // Trillion
        { value: 1e9,  suffix: 'B' },  // Billion
        { value: 1e6,  suffix: 'M' },  // Million
        { value: 1e3,  suffix: 'K' }   // Thousand
    ];
    
    for (let i = 0; i < suffixes.length; i++) {
        if (num >= suffixes[i].value) {
            const formatted = (num / suffixes[i].value).toFixed(2);
            // Remove trailing .00 for cleaner display
            const cleaned = formatted.endsWith('.00') ? formatted.slice(0, -3) : formatted;
            return cleaned + suffixes[i].suffix;
        }
    }
    
    return num.toString();
}

/**
 * Format the user's credit balance in the header navigation
 */
function formatHeaderCredits() {
    // Look for the credits display in the header nav
    const nav = document.querySelector('nav');
    if (nav) {
        const spans = nav.querySelectorAll('span');
        spans.forEach(span => {
            if (span.textContent.includes('Credits:')) {
                const text = span.textContent;
                const match = text.match(/Credits:\s*(\d+)/);
                if (match) {
                    const credits = parseInt(match[1]);
                    const formatted = formatLargeNumber(credits);
                    span.textContent = text.replace(/Credits:\s*\d+/, `Credits: ${formatted}`);
                }
            }
        });
    }
}
