// Live preview of username changes
document.getElementById('custom_username').addEventListener('input', function(e) {
    const preview = document.getElementById('namePreview');
    const usernameText = preview.querySelector('.username-text');
    const value = e.target.value.trim();
    
    if (value) {
        usernameText.textContent = value;
    } else {
        usernameText.textContent = document.querySelector('.username-text').getAttribute('data-discord-username');
    }
}); 