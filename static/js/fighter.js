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

        // Paste-to-upload for avatar
        const avatarEdit = document.querySelector('.avatar-edit');
        const fileInput = document.getElementById('avatar-file');
        const avatarImg = document.getElementById('fighter-avatar-img');
        if (avatarEdit && fileInput) {
            // Listen at the document level so paste works anywhere while the editor is open
            document.addEventListener('paste', async function (e) {
                // Only handle when the avatar editor is open/visible
                if (!avatarEdit.classList.contains('open')) return;
                try {
                    const cd = e.clipboardData || window.clipboardData;
                    if (!cd) return;
                    const items = cd.items || [];
                    // Prefer the first image file in clipboard
                    let blob = null;
                    for (let i = 0; i < items.length; i++) {
                        const it = items[i];
                        if (it && it.kind === 'file') {
                            const f = it.getAsFile();
                            if (f && /^image\//.test(f.type)) { blob = f; break; }
                        }
                    }
                    // Some browsers expose files directly
                    if (!blob && cd.files && cd.files.length) {
                        const f = cd.files[0];
                        if (f && /^image\//.test(f.type)) blob = f;
                    }
                    if (!blob) return;
                    const dt = new DataTransfer();
                    const ext = (blob.type.split('/')[1] || 'png');
                    const file = new File([blob], 'pasted-image.' + ext, { type: blob.type });
                    dt.items.add(file);
                    fileInput.files = dt.files;
                    // Preview
                    if (avatarImg) {
                        const url = URL.createObjectURL(file);
                        avatarImg.src = url;
                    }
                    const submitBtn = avatarEdit.querySelector('button[type="submit"]');
                    if (submitBtn) submitBtn.focus();
                } catch (err) {
                    console.warn('avatar paste failed', err);
                }
            });
        }
    } catch (e) {
        console.warn('fighter.js init error', e);
    }
});
