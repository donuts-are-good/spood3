// Ad interstitials: purchase wiring and obnoxious effects
(function(){
    function purchase(itemId, itemName, markOwned){
        fetch('/user/shop/purchase', {
            method:'POST', headers:{ 'Content-Type':'application/json' },
            body: JSON.stringify({ item_id: itemId, quantity: 1 })
        })
        .then(r=>r.json())
        .then(function(data){
            if (data && data.success){
                try { if (window.toast && window.toast.success) window.toast.success('Purchased ' + itemName + '. Check inventory.', 4000); } catch(_){ }
                if (typeof data.new_balance === 'number' && window.updateGlobalCreditsDisplay) {
                    window.updateGlobalCreditsDisplay(data.new_balance);
                }
                if (typeof markOwned === 'function') markOwned();
            } else {
                var msg = (data && data.error) ? data.error : 'Purchase failed';
                try { if (window.toast && window.toast.error) window.toast.error(msg, 5000); } catch(_){ }
            }
        })
        .catch(function(){ try { if (window.toast && window.toast.error) window.toast.error('Network error', 4000); } catch(_){} });
    }

    function onBuyClick(e){
        e.stopPropagation();
        var btn = e.currentTarget;
        if (btn.classList.contains('disabled') || btn.hasAttribute('disabled')) return;
        var itemId = parseInt(btn.getAttribute('data-item-id'), 10);
        var itemName = btn.getAttribute('data-item-name') || 'Product';
        if (!itemId) return;
        purchase(itemId, itemName, function(){
            btn.classList.add('disabled');
            btn.setAttribute('disabled','disabled');
            btn.textContent = 'OWNED';
        });
    }

    function initAdButtons(){
        var buttons = document.querySelectorAll('.ad-card .ad-buy');
        buttons.forEach(function(b){ b.addEventListener('click', onBuyClick); });
        // Make entire ad card clickable (unless owned is displayed)
        var cards = document.querySelectorAll('.ad-card');
        cards.forEach(function(card){
            card.addEventListener('click', function(){
                var itemId = parseInt(card.getAttribute('data-item-id'), 10);
                var itemName = card.getAttribute('data-item-name') || 'Product';
                if (!itemId) return;
                showConfirmModal(itemId, itemName);
            });
        });
    }

    function showConfirmModal(itemId, itemName){
        // Build modal DOM
        var overlay = document.createElement('div');
        overlay.className = 'ad-modal-overlay';
        var modal = document.createElement('div');
        modal.className = 'ad-modal';
        modal.innerHTML = ''+
            '<div class="ad-modal-header">' +
                '<div class="ad-modal-title">Confirm Purchase</div>' +
            '</div>' +
            '<div class="ad-modal-body">' +
                '<p>Proceeding will immediately debit your account for this experimental product.</p>' +
                '<p><strong>Disclaimer:</strong> Effects may be unpredictable and are not guaranteed. All purchases are final and non-refundable.</p>' +
                '<label class="ad-modal-check"><input type="checkbox" id="adAgree"> I agree to the Department\'s Mandatory Arbitration Agreement and Non-Refundable Purchase Terms.</label>' +
            '</div>' +
            '<div class="ad-modal-footer">' +
                '<button class="ad-modal-btn ad-cancel">Cancel</button>' +
                '<button class="ad-modal-btn ad-confirm" disabled>Confirm Purchase</button>' +
            '</div>';
        overlay.appendChild(modal);
        document.body.appendChild(overlay);

        var chk = modal.querySelector('#adAgree');
        var btnConfirm = modal.querySelector('.ad-confirm');
        var btnCancel = modal.querySelector('.ad-cancel');

        function close(){
            try { document.body.removeChild(overlay); } catch(_){}
        }
        btnCancel.addEventListener('click', function(e){ e.stopPropagation(); close(); });
        overlay.addEventListener('click', function(e){ if (e.target === overlay) close(); });
        document.addEventListener('keydown', function esc(e){ if (e.key === 'Escape'){ close(); document.removeEventListener('keydown', esc);} });
        chk.addEventListener('change', function(){ btnConfirm.disabled = !chk.checked; });
        btnConfirm.addEventListener('click', function(){
            if (btnConfirm.disabled) return;
            btnConfirm.disabled = true;
            purchase(itemId, itemName, function(){ close(); });
        });
    }

    // Flicker effect for skeevy vibes
    function initFlicker(){
        var cards = document.querySelectorAll('.ad-card .ad-title');
        cards.forEach(function(el){
            var t = 0;
            function tick(){ t += 1; var flick = (Math.sin(t/13)+Math.sin(t/7))*0.06; el.style.filter = 'brightness(' + (1.0+flick) + ')'; requestAnimationFrame(tick); }
            requestAnimationFrame(tick);
        });
    }

    document.addEventListener('DOMContentLoaded', function(){
        initAdButtons();
        initFlicker();
    });
})();

