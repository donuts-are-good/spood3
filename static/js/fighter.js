document.addEventListener('DOMContentLoaded', function () {
    try {
        // Genome grid render
        (function renderGenome() {
            const grid = document.getElementById('genome-grid');
            if (!grid) return;
            let genome = (grid.getAttribute('data-genome') || '').trim();
            if (!genome || genome.toLowerCase() === 'unknown') {
                grid.innerHTML = '<div class="genome-empty">Genome pending filing…</div>';
                return;
            }
            if (genome.startsWith('0x')) genome = genome.slice(2);
            const chunks = [];
            for (let i = 0; i + 8 <= genome.length && chunks.length < 32; i += 8) {
                chunks.push(genome.slice(i, i + 8));
            }
            while (chunks.length < 32) chunks.push('000000ff');
            grid.innerHTML = '';
            chunks.forEach((code, idx) => {
                const r = parseInt(code.slice(0, 2), 16) || 0;
                const g = parseInt(code.slice(2, 4), 16) || 0;
                const b = parseInt(code.slice(4, 6), 16) || 0;
                const a = (parseInt(code.slice(6, 8), 16) || 255) / 255;
                const cell = document.createElement('div');
                cell.className = 'gene-cell';
                cell.style.backgroundColor = `rgba(${r}, ${g}, ${b}, ${a.toFixed(3)})`;
                cell.innerHTML = `<span class="gene-idx">${idx + 1}.</span><span class="gene-code">${code.toUpperCase()}</span>`;
                grid.appendChild(cell);
            });
        })();

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
