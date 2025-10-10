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

  // Regime ribbon (placeholder colors to distinguish segments)
  const rib=document.getElementById('regime-ribbon'); if (rib) {
    const cols=['#2e7d32','#8d6e63','#b71c1c','#5c6bc0','#ff8f00'];
    for(let i=0;i<Math.max(1,days);i++){ const s=document.createElement('div'); s.className='seg'; s.style.background=cols[i%cols.length]; rib.appendChild(s); }
  }

  // Weekly composite chart: temperature (left axis), viscosity (right axis), precipitation bars
  function drawWeeklyChart(){
    const svg=document.getElementById('svg-visc-temp'); if(!svg) return; const W=860,H=320,PL=40,PR=56,PT=28,PB=36; svg.innerHTML='';
    const n = Math.max(1, data.series.length);
    const xs = (i)=> PL + (i*(W-PL-PR)/(n-1));
    // Extract series
    const temp = data.series.map(d => d.TemperatureF||0);
    const visc = data.series.map(d => d.Viscosity||0);
    const rain = data.series.map(d => d.PrecipitationMM||0);
    // Scales
    const tMin=-20, tMax=120;
    const vMin=0, vMax=100000;
    const pMin=0, pMax=100;
    const yTemp = (v)=> PT + (1-((v - tMin)/(tMax - tMin))) * (H-PT-PB);
    const yVisc = (v)=> PT + (1-((v - vMin)/(vMax - vMin))) * (H-PT-PB);
    const yPrec = (v)=> (H-PB) - ((v - pMin)/(pMax - pMin)) * (H-PT-PB);

    // Grid and x labels
    for(let i=0;i<n;i++){
      const x=xs(i); const g=document.createElementNS('http://www.w3.org/2000/svg','line');
      g.setAttribute('x1',x); g.setAttribute('y1',PT); g.setAttribute('x2',x); g.setAttribute('y2',H-PB); g.setAttribute('stroke','#1e1e1e'); svg.appendChild(g);
      const t=document.createElementNS('http://www.w3.org/2000/svg','text'); t.setAttribute('x',x); t.setAttribute('y',H-10); t.setAttribute('fill','#888'); t.setAttribute('font-size','12'); t.setAttribute('text-anchor','middle'); t.textContent='D'+(i+1); svg.appendChild(t);
    }
    // Left y (temp)
    for(let j=0;j<=5;j++){ const y=PT + j*((H-PT-PB)/5); const gl=document.createElementNS('http://www.w3.org/2000/svg','line'); gl.setAttribute('x1',PL); gl.setAttribute('y1',y); gl.setAttribute('x2',W-PR); gl.setAttribute('y2',y); gl.setAttribute('stroke','#1e1e1e'); svg.appendChild(gl); const v=tMax - j*((tMax-tMin)/5); const tt=document.createElementNS('http://www.w3.org/2000/svg','text'); tt.setAttribute('x',8); tt.setAttribute('y',y+4); tt.setAttribute('fill','#888'); tt.setAttribute('font-size','12'); tt.textContent=Math.round(v); svg.appendChild(tt); }
    // Right y (viscosity)
    for(let j=0;j<=5;j++){ const y=PT + j*((H-PT-PB)/5); const v=vMax - j*((vMax-vMin)/5); const tt=document.createElementNS('http://www.w3.org/2000/svg','text'); tt.setAttribute('x',W-PR+8); tt.setAttribute('y',y+4); tt.setAttribute('fill','#78d7ff'); tt.setAttribute('font-size','12'); tt.textContent=Math.round(v/1000)+'k'; svg.appendChild(tt); }

    // Legend
    const lg=document.getElementById('legend-main'); if(lg){ lg.innerHTML=''; const mk=(c,t)=>{const d=document.createElement('div'); d.className='lg'; const sw=document.createElement('span'); sw.className='sw'; sw.style.background=c; const tx=document.createElement('span'); tx.textContent=t; d.appendChild(sw); d.appendChild(tx); lg.appendChild(d)}; mk('#ff6fb3','Temperature (°F)'); mk('#5ad1ff','Viscosity (cP)'); mk('rgba(64,224,208,0.6)','Precipitation (mm)'); }

    // Precip bars
    for(let i=0;i<n;i++){ const x=xs(i); const y=yPrec(rain[i]); const h=(H-PB)-y; const r=document.createElementNS('http://www.w3.org/2000/svg','rect'); r.setAttribute('x',x-6); r.setAttribute('y',y); r.setAttribute('width',12); r.setAttribute('height',Math.max(0,h)); r.setAttribute('fill','rgba(64,224,208,0.28)'); r.setAttribute('stroke','rgba(64,224,208,0.7)'); svg.appendChild(r); }

    // Series: temperature
    let pT=''; for(let i=0;i<n;i++){ pT += `${i?'L':'M'}${xs(i)},${yTemp(temp[i])} `; }
    const pathT=document.createElementNS('http://www.w3.org/2000/svg','path'); pathT.setAttribute('d',pT); pathT.setAttribute('fill','none'); pathT.setAttribute('stroke','#ff6fb3'); pathT.setAttribute('stroke-width','2.5'); svg.appendChild(pathT);
    for(let i=0;i<n;i++){ const c=document.createElementNS('http://www.w3.org/2000/svg','circle'); c.setAttribute('cx',xs(i)); c.setAttribute('cy',yTemp(temp[i])); c.setAttribute('r',3); c.setAttribute('fill','#ff6fb3'); c.setAttribute('opacity','.9'); c.setAttribute('title',`Temp: ${temp[i]}°F`); svg.appendChild(c); }

    // Series: viscosity
    let pV=''; for(let i=0;i<n;i++){ pV += `${i?'L':'M'}${xs(i)},${yVisc(visc[i])} `; }
    const pathV=document.createElementNS('http://www.w3.org/2000/svg','path'); pathV.setAttribute('d',pV); pathV.setAttribute('fill','none'); pathV.setAttribute('stroke','#5ad1ff'); pathV.setAttribute('stroke-width','2.5'); svg.appendChild(pathV);
    for(let i=0;i<n;i++){ const c=document.createElementNS('http://www.w3.org/2000/svg','circle'); c.setAttribute('cx',xs(i)); c.setAttribute('cy',yVisc(visc[i])); c.setAttribute('r',3); c.setAttribute('fill','#5ad1ff'); c.setAttribute('opacity','.9'); c.setAttribute('title',`Viscosity: ${visc[i]} cP`); svg.appendChild(c); }
  }
  drawWeeklyChart();

  // Wind rose — compact with guides and titles
  (function(){const svg=document.getElementById('svg-rose'); if(!svg) return; const W=320,H=320,cx=W/2,cy=H/2,r=120; svg.innerHTML='';
    const circle=(rad)=>{const c=document.createElementNS('http://www.w3.org/2000/svg','circle'); c.setAttribute('cx',cx);c.setAttribute('cy',cy);c.setAttribute('r',rad);c.setAttribute('fill','none');c.setAttribute('stroke','#222'); svg.appendChild(c)}; circle(r*0.33); circle(r*0.66); circle(r);
    const lab=(tx,x,y)=>{const t=document.createElementNS('http://www.w3.org/2000/svg','text'); t.setAttribute('x',x); t.setAttribute('y',y); t.setAttribute('fill','#666'); t.setAttribute('font-size','11'); t.setAttribute('text-anchor','middle'); t.textContent=tx; svg.appendChild(t)}; lab('N',cx,12); lab('S',cx,H-6); lab('E',W-8,cy+4); lab('W',8,cy+4);
    for(const d of data.series){const a=((d.WindDirDeg||0)-90)*Math.PI/180; const len=Math.min(1,(d.WindSpeedMPH||0)/35)*r; const x=cx+Math.cos(a)*len; const y=cy+Math.sin(a)*len; const l=document.createElementNS('http://www.w3.org/2000/svg','line'); l.setAttribute('x1',cx);l.setAttribute('y1',cy);l.setAttribute('x2',x);l.setAttribute('y2',y); l.setAttribute('stroke','#40e0d0'); l.setAttribute('stroke-width','3'); l.setAttribute('opacity','.8'); svg.appendChild(l); const dot=document.createElementNS('http://www.w3.org/2000/svg','circle'); dot.setAttribute('cx',x); dot.setAttribute('cy',y); dot.setAttribute('r',3.5); dot.setAttribute('fill','#40e0d0'); dot.setAttribute('title',`Wind ${d.WindSpeedMPH||0} mph @ ${d.WindDirDeg||0}°`); svg.appendChild(dot); }
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


