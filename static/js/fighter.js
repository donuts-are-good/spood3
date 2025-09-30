document.addEventListener('DOMContentLoaded', function () {
    try {
        // Lore editor toggle
        const loreToggle = document.querySelector('.lore-edit-toggle');
        const loreArea = document.querySelector('.lore-edit');
        const loreCancel = document.querySelector('.lore-cancel');
        if (loreToggle && loreArea) {
            // start hidden unless template renders open
            if (!loreArea.classList.contains('open')) loreArea.style.display = 'none';

            loreToggle.addEventListener('click', function () {
                const isOpen = loreArea.style.display !== 'none';
                loreArea.style.display = isOpen ? 'none' : 'block';
                loreArea.classList.toggle('open', !isOpen);
                loreToggle.setAttribute('aria-expanded', String(!isOpen));
            });

            if (loreCancel) {
                loreCancel.addEventListener('click', function () {
                    loreArea.style.display = 'none';
                    loreArea.classList.remove('open');
                    loreToggle.setAttribute('aria-expanded', 'false');
                });
            }
        }

        // Avatar editor toggle
        const avatarToggle = document.querySelector('.avatar-edit-toggle');
        const avatarArea = document.querySelector('.avatar-edit');
        const avatarCancel = document.querySelector('.avatar-cancel');
        if (avatarToggle && avatarArea) {
            // start hidden unless template renders open
            if (!avatarArea.classList.contains('open')) avatarArea.style.display = 'none';

            avatarToggle.addEventListener('click', function () {
                const isOpen = avatarArea.style.display !== 'none';
                avatarArea.style.display = isOpen ? 'none' : 'block';
                avatarArea.classList.toggle('open', !isOpen);
                avatarToggle.setAttribute('aria-expanded', String(!isOpen));
            });

            if (avatarCancel) {
                avatarCancel.addEventListener('click', function () {
                    avatarArea.style.display = 'none';
                    avatarArea.classList.remove('open');
                    avatarToggle.setAttribute('aria-expanded', 'false');
                });
            }
        }
    } catch (e) {
        console.warn('fighter.js init error', e);
    }
});
