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

  // Weekly mini chart (Temp, Visc, Precip)
  (function(){
    const svg=document.getElementById('mini-weekly'); if(!svg) return; const W=600,H=180,P=28; svg.innerHTML='';
    const n=Math.max(1,data.series.length); const x=(i)=>P+i*((W-2*P)/(n-1));
    const dayLabels=['Mon','Tue','Wed','Thu','Fri','Sat','Sun'];
    const temp=data.series.map(d=>d.TemperatureF||0);
    const visc=data.series.map(d=>d.Viscosity||0);
    const rain=data.series.map(d=>d.PrecipitationMM||0);
    const tMin=-20,tMax=120, vMin=0,vMax=100000, pMin=0,pMax=100;
    const yT=v=>P+(1-((v-tMin)/(tMax-tMin)))*(H-2*P);
    const yV=v=>P+(1-((v-vMin)/(vMax-vMin)))*(H-2*P);
    const yP=v=>H-P-((v-pMin)/(pMax-pMin))*(H-2*P);
    // grid
    for(let i=0;i<n;i++){const gx=document.createElementNS('http://www.w3.org/2000/svg','line'); const xx=x(i); gx.setAttribute('x1',xx);gx.setAttribute('y1',P);gx.setAttribute('x2',xx);gx.setAttribute('y2',H-P);gx.setAttribute('stroke','#1e1e1e'); svg.appendChild(gx); const lbl=document.createElementNS('http://www.w3.org/2000/svg','text'); lbl.setAttribute('x',xx); lbl.setAttribute('y',H-6); lbl.setAttribute('fill', (i%7===5)?'#ffcc66':'#888'); lbl.setAttribute('font-size','11'); lbl.setAttribute('text-anchor','middle'); lbl.textContent=dayLabels[i%7]||('D'+(i+1)); svg.appendChild(lbl)}
    for(let j=0;j<=4;j++){ const y=P+j*((H-2*P)/4); const gl=document.createElementNS('http://www.w3.org/2000/svg','line'); gl.setAttribute('x1',P); gl.setAttribute('y1',y); gl.setAttribute('x2',W-P); gl.setAttribute('y2',y); gl.setAttribute('stroke','#151515'); svg.appendChild(gl) }
    // precip bars
    for(let i=0;i<n;i++){const r=document.createElementNS('http://www.w3.org/2000/svg','rect'); r.setAttribute('x',x(i)-5); r.setAttribute('y',yP(rain[i])); r.setAttribute('width',10); r.setAttribute('height',Math.max(0,(H-P)-yP(rain[i]))); r.setAttribute('fill','rgba(64,224,208,0.28)'); r.setAttribute('stroke','rgba(64,224,208,0.7)'); svg.appendChild(r)}
    // temp
    let pT=''; for(let i=0;i<n;i++){pT+=`${i?'L':'M'}${x(i)},${yT(temp[i])} `;} const pathT=document.createElementNS('http://www.w3.org/2000/svg','path'); pathT.setAttribute('d',pT); pathT.setAttribute('fill','none'); pathT.setAttribute('stroke','#ff6fb3'); pathT.setAttribute('stroke-width','2'); svg.appendChild(pathT);
    // visc
    let pV=''; for(let i=0;i<n;i++){pV+=`${i?'L':'M'}${x(i)},${yV(visc[i])} `;} const pathV=document.createElementNS('http://www.w3.org/2000/svg','path'); pathV.setAttribute('d',pV); pathV.setAttribute('fill','none'); pathV.setAttribute('stroke','#5ad1ff'); pathV.setAttribute('stroke-width','2'); svg.appendChild(pathV);
    // legend
    const lg=document.getElementById('legend-mini'); if(lg){ lg.innerHTML=''; const mk=(c,t)=>{const d=document.createElement('div'); d.className='lg'; const sw=document.createElement('span'); sw.className='sw'; sw.style.background=c; const tx=document.createElement('span'); tx.textContent=t; d.appendChild(sw); d.appendChild(tx); lg.appendChild(d)}; mk('#ff6fb3','Temp'); mk('#5ad1ff','Visc'); mk('rgba(64,224,208,0.6)','Precip'); }
    // unified tooltips per day (Temp, Visc, Precip)
    const tip=document.getElementById('wx-tip'); const show=(html,e)=>{if(!tip) return; tip.innerHTML=html; tip.style.display='block'; tip.style.left=(e.clientX+12)+'px'; tip.style.top=(e.clientY+12)+'px'}; const hide=()=>{if(tip) tip.style.display='none'};
    const step = (W-2*P)/Math.max(1,(n-1));
    for(let i=0;i<n;i++){
      const ov=document.createElementNS('http://www.w3.org/2000/svg','rect');
      ov.setAttribute('x', (x(i) - step/2));
      ov.setAttribute('y', P);
      ov.setAttribute('width', Math.max(14, step));
      ov.setAttribute('height', H-2*P);
      ov.setAttribute('fill', 'transparent');
      const html = `${dayLabels[i%7]||('D'+(i+1))}<br/>`+
        `<span style="color:#ff6fb3">Temp</span>: <b>${temp[i]}°F</b><br/>`+
        `<span style="color:#5ad1ff">Visc</span>: <b>${visc[i]}</b> cP<br/>`+
        `<span style="color:turquoise">Precip</span>: <b>${rain[i]} mm</b>`;
      ov.addEventListener('mousemove', e=>show(html,e));
      ov.addEventListener('mouseleave', hide);
      svg.appendChild(ov);
    }
  })();

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

  // Compact wind compass
  (function(){const svg=document.getElementById('wind-compass'); if(!svg) return; const W=160,H=160,cx=W/2,cy=H/2,r=60; svg.innerHTML='';
    const circle=(rad)=>{const c=document.createElementNS('http://www.w3.org/2000/svg','circle'); c.setAttribute('cx',cx);c.setAttribute('cy',cy);c.setAttribute('r',rad);c.setAttribute('fill','none');c.setAttribute('stroke','#222'); svg.appendChild(c)}; circle(r*0.5); circle(r);
    const lab=(tx,x,y)=>{const t=document.createElementNS('http://www.w3.org/2000/svg','text'); t.setAttribute('x',x); t.setAttribute('y',y); t.setAttribute('fill','#666'); t.setAttribute('font-size','10'); t.setAttribute('text-anchor','middle'); t.textContent=tx; svg.appendChild(t)}; lab('N',cx,10); lab('S',cx,H-4); lab('E',W-6,cy+4); lab('W',6,cy+4);
    const d = data.today||{}; const a=((d.WindDirDeg||0)-90)*Math.PI/180; const len=Math.min(1,(d.WindSpeedMPH||0)/35)*r; const x=cx+Math.cos(a)*len; const y=cy+Math.sin(a)*len; const l=document.createElementNS('http://www.w3.org/2000/svg','line'); l.setAttribute('x1',cx);l.setAttribute('y1',cy);l.setAttribute('x2',x);l.setAttribute('y2',y); l.setAttribute('stroke','#40e0d0'); l.setAttribute('stroke-width','3'); svg.appendChild(l); const dot=document.createElementNS('http://www.w3.org/2000/svg','circle'); dot.setAttribute('cx',x); dot.setAttribute('cy',y); dot.setAttribute('r',3.5); dot.setAttribute('fill','#40e0d0'); svg.appendChild(dot);
    // title-like tooltip
    const tip=document.getElementById('wx-tip'); const html=`Wind: <b>${d.WindSpeedMPH||0} mph</b> @ <b>${d.WindDirDeg||0}°</b>`; svg.addEventListener('mousemove',e=>{if(!tip) return; tip.innerHTML=html; tip.style.display='block'; tip.style.left=(e.clientX+12)+'px'; tip.style.top=(e.clientY+12)+'px'}); svg.addEventListener('mouseleave',()=>{if(tip) tip.style.display='none'})
  })();

  // Counts list (compact cards)
  (function(){const el=document.getElementById('counts-list'); if(!el) return; const today=data.today||{}; const counts=today.CountsJSON?JSON.parse(today.CountsJSON):{}; const items=Object.entries(counts); el.innerHTML = items.length? items.map(([k,v])=>`<li><span>${k}</span><b>${v}</b></li>`).join('') : '<li>No counts filed.</li>'; })();

  // Events list
  (function(){const el=document.getElementById('events-list'); if(!el) return; const today=data.today||{}; const events=today.EventsJSON?JSON.parse(today.EventsJSON):[]; el.innerHTML = events.length? events.map((e)=>`<li>${e.type||'event'} — ${e.intensity||0}</li>`).join('') : '<li>No events recorded.</li>'; })();

  // Indices and weekly JSON dumps as readable KV
  (function(){const el=document.getElementById('indices-list'); if(!el) return; const today=data.today||{}; const idx=today.IndicesJSON?JSON.parse(today.IndicesJSON):{}; const items=Object.entries(idx); el.innerHTML = items.length? items.map(([k,v])=>`<div class='kv-item'><span class='k'>${k}</span><span class='v'>${v}</span></div>`).join('') : '<div class="kv-item"><span class="k">No indices</span><span class="v">—</span></div>'; })();
  (function(){const el=document.getElementById('weekly-kv'); if(!el) return; const w=data.weekly||{}; const traits=w.WeeklyTraitsJSON?JSON.parse(w.WeeklyTraitsJSON):{}; const rows=[]; if(w.SeedHash){ rows.push(`<div class='kv-item'><span class='k'>Seed</span><span class='v small'>${w.SeedHash}</span></div>`);} if(w.AlgoVersion){ rows.push(`<div class='kv-item'><span class='k'>Algo</span><span class='v'>${w.AlgoVersion}</span></div>`);} for(const [k,v] of Object.entries(traits)){ rows.push(`<div class='kv-item'><span class='k'>${k}</span><span class='v'>${v}</span></div>`);} el.innerHTML = rows.join(''); })();

  // Forecast table
  (function(){const t=document.getElementById('forecast-table'); if(!t) return; const rows=data.series||[]; const header=['Day','Temp °F','Visc cP','Precip mm']; const dayLabels=['Mon','Tue','Wed','Thu','Fri','Sat','Sun']; t.innerHTML = `<tr>${header.map(h=>`<th>${h}</th>`).join('')}</tr>` + rows.map((d,i)=>`<tr><td>${dayLabels[i%7]||('D'+(i+1))}</td><td>${d.TemperatureF||0}</td><td>${d.Viscosity||0}</td><td>${d.PrecipitationMM||0}</td></tr>`).join(''); })();

  // Keyboard navigation for days
  (function(){
    const url=new URL(window.location.href); const param=url.searchParams.get('date');
    const base = param || (new Date().toISOString().slice(0,10));
    function go(delta){ const d=new Date(base+'T00:00:00'); d.setDate(d.getDate()+delta); const today=new Date(); today.setHours(0,0,0,0); if (d>today) return; window.location='/weather?date='+d.toISOString().slice(0,10); }
    document.addEventListener('keydown', (e)=>{
      if (['INPUT','TEXTAREA'].includes((document.activeElement.tagName||'').toUpperCase())) return;
      if (e.key==='ArrowLeft') { e.preventDefault(); go(-1); }
      else if (e.key==='ArrowRight') { e.preventDefault(); go(1); }
    });
  })();

  // Notes
  (function(){const el=document.getElementById('notes'); if(!el) return; for(const d of data.series){const p=document.createElement('div'); const dt=(d.Date||'').toString().slice(0,10); p.textContent=`${dt||'Day'} — temp ${d.TemperatureF||0}°F; visc ${d.Viscosity||0} cP; rain ${d.PrecipitationMM||0}mm.`; el.appendChild(p)} })();
})();


