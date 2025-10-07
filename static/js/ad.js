// Ad interstitials: purchase wiring and obnoxious effects
(function(){
    function onBuyClick(e){
        e.stopPropagation();
        var btn = e.currentTarget;
        if (btn.classList.contains('disabled') || btn.hasAttribute('disabled')) return;
        var itemId = parseInt(btn.getAttribute('data-item-id'), 10);
        var itemName = btn.getAttribute('data-item-name') || 'Product';
        if (!itemId) return;
        fetch('/user/shop/purchase', {
            method:'POST', headers:{ 'Content-Type':'application/json' },
            body: JSON.stringify({ item_id: itemId, quantity: 1 })
        })
        .then(r=>r.json())
        .then(function(data){
            if (data && data.success){
                try { if (window.toast && window.toast.success) window.toast.success('Purchased ' + itemName + '. Check inventory.', 4000); } catch(_){}
                if (typeof data.new_balance === 'number' && window.updateGlobalCreditsDisplay) {
                    window.updateGlobalCreditsDisplay(data.new_balance);
                }
                // Disable button after purchase
                btn.classList.add('disabled');
                btn.setAttribute('disabled','disabled');
                btn.textContent = 'OWNED';
            } else {
                var msg = (data && data.error) ? data.error : 'Purchase failed';
                try { if (window.toast && window.toast.error) window.toast.error(msg, 5000); } catch(_){}
            }
        })
        .catch(function(){ try { if (window.toast && window.toast.error) window.toast.error('Network error', 4000); } catch(_){} });
    }

    function initAdButtons(){
        var buttons = document.querySelectorAll('.ad-card .ad-buy');
        buttons.forEach(function(b){ b.addEventListener('click', onBuyClick); });
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

