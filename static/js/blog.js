// Subtle ticker drift for the serum modal and outside-click close
(function(){
  function drift(){
    var ticks = document.querySelectorAll('.ticker-grid .tick span');
    if (!ticks.length) return requestAnimationFrame(drift);
    ticks.forEach(function(el, i){
      var txt = (el.textContent||'').trim();
      if (txt.indexOf('∞') !== -1) return; // leave as-is
      var sign = txt.charAt(0)==='-'?'-':'+';
      var base = parseFloat(txt.replace(/[^0-9.-]/g,''));
      if (!isFinite(base)) return;
      var delta = (Math.sin(Date.now()/1500 + i)*0.03); // small oscillation
      var newVal = base + delta;
      el.textContent = sign + newVal.toFixed(2) + '%';
    });
    requestAnimationFrame(drift);
  }

  function enableOutsideClickClose(){
    var overlay = document.getElementById('serum-protocol');
    if (!overlay) return;
    overlay.addEventListener('click', function(e){
      if (e.target === overlay) {
        history.pushState('', document.title, window.location.pathname + window.location.search);
      }
    });
  }

  window.addEventListener('DOMContentLoaded', function(){
    enableOutsideClickClose();
    drift();
  });
})();


