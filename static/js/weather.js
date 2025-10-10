(function(){
  const data = window.__WEATHER__ || {weekly:null,today:null,series:[]};
  const days = data.series && data.series.length ? data.series.length : 0;

  // Header KV
  if (data.weekly) {
    const w = data.weekly; const set=(id,v)=>{const el=document.getElementById(id); if(el) el.textContent=v||'—'};
    set('kv-biome', w.Biome); set('kv-pizza', w.PizzaSelection); set('kv-off', w.CasinoOfficials);
  }
  if (data.today) {
    const t = data.today; const set=(id,v)=>{const el=document.getElementById(id); if(el) el.textContent=v||'—'};
    set('kv-cheese', t.CheeseSmell); set('kv-time', t.TimeMode);
    // Today full readout
    const kv=document.getElementById('today-kv'); if(kv){
      const pairs = [
        ['Date', (t.Date||'').toString().slice(0,10)],
        ['Regime', t.Regime],
        ['Viscosity (cP)', t.Viscosity],
        ['Temperature (°F)', t.TemperatureF],
        ['Temporality', t.Temporality],
        ['Wind Speed (mph)', t.WindSpeedMPH],
        ['Wind Dir (deg)', t.WindDirDeg],
        ['Precipitation (mm)', t.PrecipitationMM],
        ['Drizzle (min)', t.DrizzleMinutes],
      ];
      kv.innerHTML = pairs.map(([k,v])=>`<div class='kv-item'><span class='k'>${k}</span><span class='v'>${v??'—'}</span></div>`).join('');
    }
  }

  // Regime ribbon
  const rib=document.getElementById('regime-ribbon'); if (rib) {
    const cols=['#2e7d32','#8d6e63','#b71c1c','#5c6bc0','#ff8f00'];
    for(let i=0;i<Math.max(1,days);i++){ const s=document.createElement('div'); s.className='seg'; s.style.background=cols[i%cols.length]; rib.appendChild(s); }
  }

  // Draw axes, legend, and series (two y-axes conceptually)
  function drawLine(svgId, arr, color, min, max, precip){
    const svg=document.getElementById(svgId); if(!svg || !arr || !arr.length) return; const W=860,H=320,P=36; svg.innerHTML='';
    // axes
    const axisCol='#3a3a3a';
    const xSteps=arr.length-1; for(let i=0;i<=xSteps;i++){const x=P+i*((W-2*P)/(xSteps)); const g=document.createElementNS('http://www.w3.org/2000/svg','line'); g.setAttribute('x1',x);g.setAttribute('y1',P);g.setAttribute('x2',x);g.setAttribute('y2',H-P);g.setAttribute('stroke','#1e1e1e'); svg.appendChild(g); const t=document.createElementNS('http://www.w3.org/2000/svg','text'); t.setAttribute('x',x); t.setAttribute('y',H-P+18); t.setAttribute('fill','#888'); t.setAttribute('font-size','12'); t.setAttribute('text-anchor','middle'); t.textContent = 'D'+(i+1); svg.appendChild(t); }
    const yTicks=5; for(let j=0;j<=yTicks;j++){const y=P+j*((H-2*P)/yTicks); const g=document.createElementNS('http://www.w3.org/2000/svg','line'); g.setAttribute('x1',P);g.setAttribute('y1',y);g.setAttribute('x2',W-P);g.setAttribute('y2',y);g.setAttribute('stroke','#1e1e1e'); svg.appendChild(g); const v=max - j*((max-min)/yTicks); const t=document.createElementNS('http://www.w3.org/2000/svg','text'); t.setAttribute('x',8); t.setAttribute('y',y+4); t.setAttribute('fill','#888'); t.setAttribute('font-size','12'); t.textContent = Math.round(v); svg.appendChild(t); }
    // legend
    const lg=document.getElementById('legend-main'); if(lg){ lg.innerHTML=''; const mk=(c,t)=>{const d=document.createElement('div'); d.className='lg'; const sw=document.createElement('span'); sw.className='sw'; sw.style.background=c; const tx=document.createElement('span'); tx.textContent=t; d.appendChild(sw); d.appendChild(tx); lg.appendChild(d)}; mk('#5ad1ff','Viscosity (cP)'); mk('#ff6fb3','Temperature (°F)'); mk('rgba(64,224,208,0.6)','Precipitation (mm)'); }
    // lines
    let path=''; for(let i=0;i<arr.length;i++){const x=P+i*((W-2*P)/(arr.length-1)); const y=P+(1-((arr[i]-min)/(max-min)))*(H-2*P); path+=`${i?'L':'M'}${x},${y} `}
    const p=document.createElementNS('http://www.w3.org/2000/svg','path'); p.setAttribute('d',path); p.setAttribute('fill','none'); p.setAttribute('stroke',color); p.setAttribute('stroke-width','2.5'); svg.appendChild(p);
    // precipitation bars on secondary axis (0-100)
    if (precip) { const pmin=0,pmax=100; for(let i=0;i<precip.length;i++){ const x=P+i*((W-2*P)/precip.length); const h=((precip[i]-pmin)/(pmax-pmin))*(H-2*P); const r=document.createElementNS('http://www.w3.org/2000/svg','rect'); r.setAttribute('x',x-6); r.setAttribute('y',H-P-h); r.setAttribute('width',12); r.setAttribute('height',h); r.setAttribute('fill','rgba(64,224,208,0.28)'); r.setAttribute('stroke','rgba(64,224,208,0.7)'); svg.appendChild(r) } }
  }

  const visc = data.series.map(d => d.Viscosity||0);
  const temp = data.series.map(d => d.TemperatureF||0);
  const rain = data.series.map(d => d.PrecipitationMM||0);
  drawLine('svg-visc-temp', visc, '#5ad1ff', 0, 100000, rain);
  drawLine('svg-visc-temp', temp, '#ff6fb3', -20, 120, null);

  // Wind rose — larger
  (function(){const svg=document.getElementById('svg-rose'); if(!svg) return; const W=420,H=420,cx=W/2,cy=H/2,r=160; svg.innerHTML='';
    const circle=(rad)=>{const c=document.createElementNS('http://www.w3.org/2000/svg','circle'); c.setAttribute('cx',cx);c.setAttribute('cy',cy);c.setAttribute('r',rad);c.setAttribute('fill','none');c.setAttribute('stroke','#222'); svg.appendChild(c)}; circle(r); circle(r*0.66); circle(r*0.33);
    for(const d of data.series){const a=((d.WindDirDeg||0)-90)*Math.PI/180; const len=((d.WindSpeedMPH||0)/40)*r; const x=cx+Math.cos(a)*len; const y=cy+Math.sin(a)*len; const l=document.createElementNS('http://www.w3.org/2000/svg','line'); l.setAttribute('x1',cx);l.setAttribute('y1',cy);l.setAttribute('x2',x);l.setAttribute('y2',y); l.setAttribute('stroke','#40e0d0'); l.setAttribute('stroke-width','3'); l.setAttribute('opacity','.7'); svg.appendChild(l)}
  })();

  // Swarm counts
  (function(){const svg=document.getElementById('svg-swarm'); if(!svg) return; const W=420,H=240; svg.innerHTML=''; const today=data.today||{}; const counts=today.CountsJSON?JSON.parse(today.CountsJSON):{}; const items=Object.entries(counts); if(!items.length){const t=document.createElementNS('http://www.w3.org/2000/svg','text'); t.setAttribute('x',12); t.setAttribute('y',H/2); t.setAttribute('fill','#888'); t.textContent='No counts filed.'; svg.appendChild(t); return;} let x=40; for(const [k,v] of items){ const n=v||0; for(let i=0;i<n;i++){ const cx=x + Math.random()*20; const cy=H-20 - (Math.random()* (H-60)); const dot=document.createElementNS('http://www.w3.org/2000/svg','circle'); dot.setAttribute('cx',cx); dot.setAttribute('cy',cy); dot.setAttribute('r',4); dot.setAttribute('fill','#ffcc66'); svg.appendChild(dot);} const label=document.createElementNS('http://www.w3.org/2000/svg','text'); label.setAttribute('x',x); label.setAttribute('y',H-4); label.setAttribute('fill','#fff'); label.setAttribute('font-size','12'); label.textContent=k+` (${n})`; svg.appendChild(label); x+=Math.max(60, (W-80)/items.length); } })();

  // Events
  (function(){const svg=document.getElementById('svg-events'); if(!svg) return; const W=420,H=160,P=24; svg.innerHTML=''; const today=data.today||{}; const events=today.EventsJSON?JSON.parse(today.EventsJSON):[]; if(!events.length){const t=document.createElementNS('http://www.w3.org/2000/svg','text'); t.setAttribute('x',P); t.setAttribute('y',H/2); t.setAttribute('fill','#888'); t.textContent='No events recorded.'; svg.appendChild(t); return;} let x=P; const step=(W-2*P)/Math.max(1,events.length-1); for(const ev of events){ const y = H/2 + ((ev.intensity||0)-50); const c=document.createElementNS('http://www.w3.org/2000/svg','circle'); c.setAttribute('cx',x); c.setAttribute('cy',y); c.setAttribute('r',5); c.setAttribute('fill','#ff6fb3'); svg.appendChild(c); const label=document.createElementNS('http://www.w3.org/2000/svg','text'); label.setAttribute('x',x+8); label.setAttribute('y',y-8); label.setAttribute('fill','#fff'); label.setAttribute('font-size','11'); label.textContent=ev.type||'event'; svg.appendChild(label); x+=step; } })();

  // Indices and weekly JSON dumps as readable KV
  (function(){const el=document.getElementById('indices-list'); if(!el) return; const today=data.today||{}; const idx=today.IndicesJSON?JSON.parse(today.IndicesJSON):{}; const items=Object.entries(idx); el.innerHTML = items.length? items.map(([k,v])=>`<div class='kv-item'><span class='k'>${k}</span><span class='v'>${v}</span></div>`).join('') : '<div class="kv-item"><span class="k">No indices</span><span class="v">—</span></div>'; })();
  (function(){const el=document.getElementById('weekly-kv'); if(!el) return; const w=data.weekly||{}; const traits=w.WeeklyTraitsJSON?JSON.parse(w.WeeklyTraitsJSON):{}; const pairs=[['Seed', w.SeedHash||'—'], ['Algo', w.AlgoVersion||'—']]; const extra=Object.entries(traits); el.innerHTML = pairs.concat(extra).map(([k,v])=>`<div class='kv-item'><span class='k'>${k}</span><span class='v'>${v}</span></div>`).join(''); })();

  // Notes
  (function(){const el=document.getElementById('notes'); if(!el) return; for(const d of data.series){const p=document.createElement('div'); const dt=(d.Date||'').toString().slice(0,10); p.textContent=`${dt||'Day'} — temp ${d.TemperatureF||0}°F; visc ${d.Viscosity||0} cP; rain ${d.PrecipitationMM||0}mm.`; el.appendChild(p)} })();
})();


