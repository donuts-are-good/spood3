document.addEventListener('DOMContentLoaded', function () {
    const toggle = document.querySelector('.lore-edit-toggle');
    const area = document.querySelector('.lore-edit');
    const cancel = document.querySelector('.lore-cancel');
    if (toggle && area) {
        toggle.addEventListener('click', () => {
            const open = area.style.display !== 'none' && area.style.display !== '' ? true : area.classList.contains('open');
            const nextState = !(area.style.display !== 'none' && area.style.display !== '' || area.classList.contains('open'));
            if (nextState) {
                area.style.display = 'block';
                area.classList.add('open');
            } else {
                area.style.display = 'none';
                area.classList.remove('open');
            }
            toggle.setAttribute('aria-expanded', String(nextState));
        });
    }
    if (cancel && area && toggle) {
        cancel.addEventListener('click', () => {
            area.style.display = 'none';
            area.classList.remove('open');
            toggle.setAttribute('aria-expanded', 'false');
        });
    }
});
