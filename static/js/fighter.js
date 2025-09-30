document.addEventListener('DOMContentLoaded', function () {
    try {
        // Lore editor toggle
        const loreToggle = document.querySelector('.lore-edit-toggle');
        const loreArea = document.querySelector('.lore-edit');
        const loreCancel = document.querySelector('.lore-cancel');
        if (loreToggle && loreArea) {
            loreToggle.addEventListener('click', function () {
                loreArea.classList.toggle('open');
                loreToggle.setAttribute('aria-expanded', String(loreArea.classList.contains('open')));
            });

            if (loreCancel) {
                loreCancel.addEventListener('click', function () {
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
            avatarToggle.addEventListener('click', function () {
                avatarArea.classList.toggle('open');
                avatarToggle.setAttribute('aria-expanded', String(avatarArea.classList.contains('open')));
            });

            if (avatarCancel) {
                avatarCancel.addEventListener('click', function () {
                    avatarArea.classList.remove('open');
                    avatarToggle.setAttribute('aria-expanded', 'false');
                });
            }
        }
    } catch (e) {
        console.warn('fighter.js init error', e);
    }
});
