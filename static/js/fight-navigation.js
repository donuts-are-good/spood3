// Fight Navigation - Keyboard Controls
document.addEventListener('DOMContentLoaded', function() {
    // Get current fight ID from URL
    const pathParts = window.location.pathname.split('/');
    const currentFightId = parseInt(pathParts[pathParts.length - 1]);
    
    if (isNaN(currentFightId)) {
        return; // Not on a fight page
    }
    
    // Handle keyboard navigation
    document.addEventListener('keydown', function(event) {
        // Only handle arrow keys if no input is focused
        if (document.activeElement.tagName === 'INPUT' || 
            document.activeElement.tagName === 'TEXTAREA') {
            return;
        }
        
        let newFightId = null;
        
        if (event.key === 'ArrowLeft') {
            // Previous fight
            newFightId = currentFightId - 1;
            event.preventDefault();
        } else if (event.key === 'ArrowRight') {
            // Next fight
            newFightId = currentFightId + 1;
            event.preventDefault();
        }
        
        if (newFightId !== null && newFightId > 0) {
            window.location.href = `/fight/${newFightId}`;
        }
    });
    
    // Add visual feedback for arrow buttons
    const leftArrow = document.querySelector('.fight-nav-left .nav-arrow');
    const rightArrow = document.querySelector('.fight-nav-right .nav-arrow');
    
    if (leftArrow) {
        leftArrow.addEventListener('click', function(event) {
            event.preventDefault();
            const newFightId = currentFightId - 1;
            if (newFightId > 0) {
                window.location.href = `/fight/${newFightId}`;
            }
        });
    }
    
    if (rightArrow) {
        rightArrow.addEventListener('click', function(event) {
            event.preventDefault();
            const newFightId = currentFightId + 1;
            window.location.href = `/fight/${newFightId}`;
        });
    }
}); 