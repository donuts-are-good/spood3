document.addEventListener('DOMContentLoaded', function () {
    try {
        const toggle = document.querySelector('.lore-edit-toggle');
        const area = document.querySelector('.lore-edit');
        const cancel = document.querySelector('.lore-cancel');
        if (!toggle || !area) return;
        // start hidden unless template renders open
        if (!area.classList.contains('open')) area.style.display = 'none';

        toggle.addEventListener('click', function () {
            const isOpen = area.style.display !== 'none';
            area.style.display = isOpen ? 'none' : 'block';
            area.classList.toggle('open', !isOpen);
            toggle.setAttribute('aria-expanded', String(!isOpen));
        });

        if (cancel) {
            cancel.addEventListener('click', function () {
                area.style.display = 'none';
                area.classList.remove('open');
                toggle.setAttribute('aria-expanded', 'false');
            });
        }
    } catch (e) {
        console.warn('fighter.js init error', e);
    }
});
