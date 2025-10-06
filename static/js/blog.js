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
    // Highlight Hissy Scale row nearest the pointer
    var scale = document.querySelector('.scale-table');
    if (scale){
      scale.addEventListener('mousemove', function(e){
        var rows = scale.querySelectorAll('.row');
        var best = null; var bestDist = Infinity;
        rows.forEach(function(r){
          var rect = r.getBoundingClientRect();
          var cy = rect.top + rect.height/2; var cx = rect.left + rect.width/2;
          var dx = (e.clientX - cx); var dy = (e.clientY - cy);
          var d = dx*dx + dy*dy;
          if (d < bestDist){ bestDist = d; best = r; }
          r.classList.remove('active');
        });
        if (best) best.classList.add('active');
      });
      scale.addEventListener('mouseleave', function(){
        scale.querySelectorAll('.row').forEach(function(r){ r.classList.remove('active'); });
      });
    }
  });
})();


