(function(){
  const data = window.__WEATHER__ || {weekly:null,today:null,series:[]};
  const days = data.series && data.series.length ? data.series.length : 0;

  // Ribbon
  const rib=document.getElementById('regime-ribbon'); if (rib) {
    const cols=['#2e7d32','#8d6e63','#b71c1c','#5c6bc0','#ff8f00'];
    for(let i=0;i<Math.max(1,days);i++){ const s=document.createElement('div'); s.className='seg'; s.style.background=cols[i%cols.length]; rib.appendChild(s); }
  }

  function drawLine(svgId, arr, color, min, max, precip){
    const svg=document.getElementById(svgId); if(!svg || !arr || !arr.length) return; const W=420,H=160,P=18; svg.innerHTML='';
    for(let i=0;i<=6;i++){const x=P+i*((W-2*P)/6); const g=document.createElementNS('http://www.w3.org/2000/svg','line'); g.setAttribute('x1',x);g.setAttribute('y1',P);g.setAttribute('x2',x);g.setAttribute('y2',H-P);g.setAttribute('stroke','#1e1e1e'); svg.appendChild(g)}
    for(let j=0;j<3;j++){const y=P+j*((H-2*P)/3); const g=document.createElementNS('http://www.w3.org/2000/svg','line'); g.setAttribute('x1',P);g.setAttribute('y1',y);g.setAttribute('x2',W-P);g.setAttribute('y2',y);g.setAttribute('stroke','#1e1e1e'); svg.appendChild(g)}
    let path=''; for(let i=0;i<arr.length;i++){const x=P+i*((W-2*P)/(arr.length-1)); const y=P+(1-((arr[i]-min)/(max-min)))*(H-2*P); path+=`${i?'L':'M'}${x},${y} `}
    const p=document.createElementNS('http://www.w3.org/2000/svg','path'); p.setAttribute('d',path); p.setAttribute('fill','none'); p.setAttribute('stroke',color); p.setAttribute('stroke-width','2.5'); svg.appendChild(p);
    if (precip) { for(let i=0;i<precip.length;i++){ const x=P+i*((W-2*P)/precip.length); const h=(precip[i]/100)*(H-2*P); const r=document.createElementNS('http://www.w3.org/2000/svg','rect'); r.setAttribute('x',x-5); r.setAttribute('y',H-P-h); r.setAttribute('width',10); r.setAttribute('height',h); r.setAttribute('fill','rgba(64,224,208,0.28)'); r.setAttribute('stroke','rgba(64,224,208,0.7)'); svg.appendChild(r) } }
  }

  // Build arrays from daily series
  const visc = data.series.map(d => d.viscosity||0);
  const temp = data.series.map(d => d.temperature_f||0);
  const rain = data.series.map(d => d.precipitation_mm||0);
  drawLine('v-chart', visc, '#5ad1ff', 0, 100000, rain);
  drawLine('t-chart', temp, '#ff6fb3', -20, 120, null);

  // Wind rose (mini)
  (function(){const svg=document.getElementById('wind-rose'); if(!svg) return; const W=420,H=160,cx=W/2,cy=H/2,r=60; svg.innerHTML='';
    const circle=(rad)=>{const c=document.createElementNS('http://www.w3.org/2000/svg','circle'); c.setAttribute('cx',cx);c.setAttribute('cy',cy);c.setAttribute('r',rad);c.setAttribute('fill','none');c.setAttribute('stroke','#222'); svg.appendChild(c)}; circle(r); circle(r*0.66); circle(r*0.33);
    for(const d of data.series){const a=((d.wind_dir_deg||0)-90)*Math.PI/180; const len=((d.wind_speed_mph||0)/40)*r; const x=cx+Math.cos(a)*len; const y=cy+Math.sin(a)*len; const l=document.createElementNS('http://www.w3.org/2000/svg','line'); l.setAttribute('x1',cx);l.setAttribute('y1',cy);l.setAttribute('x2',x);l.setAttribute('y2',y); l.setAttribute('stroke','#40e0d0'); l.setAttribute('stroke-width','3'); l.setAttribute('opacity','.7'); svg.appendChild(l)}
  })();

  // Counts
  (function(){const el=document.getElementById('counts'); if(!el) return; const today=data.today||{}; const counts=today.counts_json?JSON.parse(today.counts_json):{}; const items=Object.entries(counts); if(!items.length){el.textContent='No counts filed.'; return;} for(const [k,v] of items){const c=document.createElement('div'); c.className='chip'; c.innerHTML=`<span class="k">${k}</span><span class="v">${v}</span>`; el.appendChild(c)} })();

  // Events
  (function(){const el=document.getElementById('events'); if(!el) return; const today=data.today||{}; const events=today.events_json?JSON.parse(today.events_json):[]; if(!events.length){el.textContent='No events recorded.'; return;} for(const ev of events){const d=document.createElement('div'); d.textContent = `${ev.time||''} · ${ev.type||'event'} (${ev.intensity||''}) ${ev.note||''}`; el.appendChild(d)} })();

  // Notes
  (function(){const el=document.getElementById('notes'); if(!el) return; for(const d of data.series){const p=document.createElement('div'); const dt=(d.date||'').toString().slice(0,10); p.textContent=`${dt||'Day'} — temp ${d.temperature_f||0}°F; visc ${d.viscosity||0} cP; rain ${d.precipitation_mm||0}mm.`; el.appendChild(p)} })();
})();


