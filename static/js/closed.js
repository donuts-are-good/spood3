// Closed page functionality - secret casino access

document.addEventListener('DOMContentLoaded', function() {
    const commissionerSignature = document.getElementById('commissioner-signature');
    
    if (commissionerSignature) {
        // Add secret casino access via double-click
        commissionerSignature.addEventListener('dblclick', function() {
            // Add a subtle visual feedback
            commissionerSignature.style.color = '#ffd700';
            commissionerSignature.style.textShadow = '0 0 10px #ffd700';
            
            // Redirect to secret casino after brief delay
            setTimeout(() => {
                window.location.href = '/user/casino';
            }, 200);
        });
        
        // Add hover effect to hint at interactivity (very subtle)
        commissionerSignature.addEventListener('mouseenter', function() {
            commissionerSignature.style.cursor = 'default';
        });
    }
}); 