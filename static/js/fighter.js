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
            avatarEdit.addEventListener('paste', async function (e) {
                try {
                    const items = e.clipboardData && e.clipboardData.items ? e.clipboardData.items : [];
                    for (let i = 0; i < items.length; i++) {
                        const it = items[i];
                        if (it.kind === 'file') {
                            const blob = it.getAsFile();
                            if (!blob) continue;
                            // Only accept images
                            if (!/^image\//.test(blob.type)) continue;
                            // Put the blob into the file input using DataTransfer
                            const dt = new DataTransfer();
                            const file = new File([blob], 'pasted-image.' + (blob.type.split('/')[1] || 'png'), { type: blob.type });
                            dt.items.add(file);
                            fileInput.files = dt.files;
                            // Preview
                            if (avatarImg) {
                                const url = URL.createObjectURL(file);
                                avatarImg.src = url;
                            }
                            // Focus the Upload button for convenience
                            const submitBtn = avatarEdit.querySelector('button[type="submit"]');
                            if (submitBtn) submitBtn.focus();
                            break;
                        }
                    }
                } catch (err) {
                    console.warn('avatar paste failed', err);
                }
            });
        }
    } catch (e) {
        console.warn('fighter.js init error', e);
    }
});
