// Saturday schedule client logic matching preview layout.
(function () {
  const overviewRoot = document.getElementById('overview');
  const groupsRoot = document.getElementById('groups');
  const showdownRoot = document.getElementById('showdown');
  if (!overviewRoot || !groupsRoot || !showdownRoot) return;

  const fightsPayload = loadJSON('#saturday-fights') || [];
  const betMap = loadJSON('#saturday-bets') || {};
  const nowPayload = loadJSON('#saturday-now');
  const now = nowPayload && nowPayload.now ? new Date(nowPayload.now) : new Date();

  const fights = fightsPayload.map(enrichFight).sort((a, b) => a.scheduledTime - b.scheduledTime);
  const partitions = partitionGroups(fights);

  renderOverview(fights, now);
  renderGroups(partitions.groups, betMap);
  renderShowdown(partitions.playoffs);

  pollForUpdates();

  function loadJSON(selector) {
    const el = document.querySelector(selector);
    if (!el) return null;
    try {
      return JSON.parse(el.textContent || 'null');
    } catch (err) {
      console.error('Saturday JSON parse failed for', selector, err);
      return null;
    }
  }

  function enrichFight(fight) {
    const scheduledTime = new Date(fight.scheduled_time || fight.scheduledTime);
    const totalMinutes = scheduledTime.getHours() * 60 + scheduledTime.getMinutes();
    const slot = Math.round((totalMinutes - (10 * 60 + 30)) / 30);
    const timeLabel = scheduledTime.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
    const isPlayoff = timeLabel === '22:30' || timeLabel === '23:00' || timeLabel === '23:30';
    return {
      ...fight,
      scheduledTime,
      slot,
      timeLabel,
      isPlayoff,
    };
  }

  function partitionGroups(fights) {
    const labels = ['A', 'B', 'C', 'D'];
    const groups = { A: [], B: [], C: [], D: [] };
    const playoffs = [];

    fights.forEach((fight) => {
      if (fight.isPlayoff) {
        playoffs.push(fight);
        return;
      }
      const label = labels[Math.floor(fight.slot / 6)];
      if (groups[label]) groups[label].push(fight);
    });

    return { groups, playoffs };
  }

  function renderOverview(fights, now) {
    const fightsScript = document.getElementById('saturday-fights');
    const weekAttr = fightsScript ? fightsScript.dataset.week : '';
    const weekLabel = weekAttr ? `Week ${weekAttr}` : 'Week';
    const liveFight = fights.find((f) => f.status === 'active');
    const nextFight = fights.find((f) => f.status === 'scheduled' && f.scheduledTime >= now);
    const completedCount = fights.filter((f) => f.status === 'completed' || f.status === 'voided').length;
    const progressPct = Math.round((completedCount / 27) * 100);

    const nextURL = nextFight
      ? (nextFight.status === 'active' ? `/watch/${nextFight.id}` : `/fight/${nextFight.id}`)
      : null;

    const cards = [
      {
        label: 'Week',
        value: `${weekLabel} · 27 fights · 4 groups`,
        sub: 'Department of Recreational Violence',
        className: 'primary'
      },
      {
        label: 'Live Now',
        value: liveFight ? `${liveFight.fighter1_name} vs ${liveFight.fighter2_name}` : '—',
        sub: liveFight ? 'Broadcasting live across the AR grid' : 'No active bout',
        className: 'live',
        dataUrl: liveFight ? `/fight/${liveFight.id}` : undefined
      },
      {
        label: 'Next Fight',
        value: nextFight ? `${nextFight.timeLabel} ${nextFight.fighter1_name} vs ${nextFight.fighter2_name}` : 'TBD',
        sub: buildCountdown(nextFight, now),
        cta: true,
        dataUrl: nextURL
      },
      {
        label: 'Progress',
        value: `${progressPct}% complete`,
        sub: 'Completed bouts out of 27'
      }
    ];

    overviewRoot.innerHTML = cards.map((card) => `
      <div class="overview-card ${card.className || ''}" ${card.dataUrl ? `data-url="${card.dataUrl}"` : ''}>
        <div class="card-title">${card.label}</div>
        <div class="card-value">${card.value}</div>
        <div class="card-sub">${card.sub}</div>
        ${card.cta ? '<div class="bet-tag">Place bets before bell</div>' : ''}
      </div>
    `).join('');

    // Make any overview card with a URL clickable (e.g., Live Now, Next Fight)
    const clickableCards = overviewRoot.querySelectorAll('.overview-card[data-url]');
    clickableCards.forEach((cardEl) => {
      cardEl.classList.add('clickable');
      cardEl.tabIndex = 0;
      const target = cardEl.getAttribute('data-url');
      const go = () => { window.location = target; };
      cardEl.addEventListener('click', go);
      cardEl.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); go(); }
      });
    });
  }

  function buildCountdown(nextFight, now) {
    if (!nextFight) return 'Awaiting schedule';
    const diffMs = nextFight.scheduledTime - now;
    if (diffMs <= 0) return 'Betting closed';
    const minutesLeft = Math.floor(diffMs / 60000);
    return `${minutesLeft}m until bell`;
  }

  function renderGroups(groupMap, betMap) {
    const labels = ['A', 'B', 'C', 'D'];
    groupsRoot.innerHTML = '';

    labels.forEach((label) => {
      const fights = groupMap[label];
      if (!fights || !fights.length) return;

      const card = document.createElement('article');
      card.className = 'group-card';

      const header = document.createElement('div');
      header.className = 'group-header';
      header.innerHTML = `
        <div class="group-title">Group ${label}</div>
        <div class="record-pill">Top seed: ${computeStandings(fights)[0].name}</div>
      `;
      card.appendChild(header);

      const standingsEl = document.createElement('section');
      standingsEl.className = 'standings';
      const standings = computeStandings(fights);
      standingsEl.innerHTML = `
        <h3>Standings</h3>
        ${standings.map(row => `
          <div class="standing-row">
            <div class="rank">${row.rank}</div>
            <div class="name ${betMap[row.id] ? 'bet' : ''}"><a href="/fighter/${row.id}" title="View fighter">${row.name}</a></div>
            <div>${row.wins}-${row.losses}</div>
            <div>${row.diff > 0 ? `+${row.diff}` : row.diff}</div>
          </div>
        `).join('')}
      `;
      card.appendChild(standingsEl);

      const matchesEl = document.createElement('section');
      matchesEl.className = 'matches';
      fights.forEach((fight) => {
        matchesEl.appendChild(renderMatchCard(fight, betMap));
      });
      card.appendChild(matchesEl);

      groupsRoot.appendChild(card);
    });
  }

  function computeStandings(fights) {
    const table = new Map();

    fights.forEach((fight) => {
      const f1 = ensure(table, fight.fighter1_id, fight.fighter1_name);
      const f2 = ensure(table, fight.fighter2_id, fight.fighter2_name);

      if (fight.status === 'completed' && fight.winner_id) {
        const winner = Number(fight.winner_id);
        if (winner === fight.fighter1_id) {
          f1.wins += 1;
          f2.losses += 1;
          if (!f1.firstWin && fight.completed_at) {
            f1.firstWin = new Date(fight.completed_at);
          }
        } else if (winner === fight.fighter2_id) {
          f2.wins += 1;
          f1.losses += 1;
          if (!f2.firstWin && fight.completed_at) {
            f2.firstWin = new Date(fight.completed_at);
          }
        }

        const score1 = fight.final_score1 || 0;
        const score2 = fight.final_score2 || 0;
        f1.diff += score1 - score2;
        f2.diff += score2 - score1;
        f1.completed += 1;
        f2.completed += 1;
      }
    });

    const rows = Array.from(table.values());
    rows.sort((a, b) => {
      if (a.wins !== b.wins) return b.wins - a.wins;
      if (a.diff !== b.diff) return b.diff - a.diff;
      if (a.firstWin && b.firstWin) {
        const aTime = a.firstWin.getTime();
        const bTime = b.firstWin.getTime();
        if (aTime !== bTime) return aTime - bTime;
      } else if (a.firstWin && !b.firstWin) {
        return -1;
      } else if (!a.firstWin && b.firstWin) {
        return 1;
      }
      return a.id - b.id;
    });

    return rows.map((row, idx) => ({ ...row, rank: idx + 1 }));

    function ensure(map, id, name) {
      if (!map.has(id)) {
        map.set(id, { id, name, wins: 0, losses: 0, diff: 0, completed: 0, firstWin: null });
      }
      return map.get(id);
    }
  }

  function renderMatchCard(fight, betMap) {
    const el = document.createElement('div');
    el.className = 'match-card';

    el.innerHTML = `
      <div class="match-top">
        <span>${fight.timeLabel}</span>
        ${matchBadge(fight.status)}
      </div>
      <div class="fighter-pair">
        <div class="fighter-name ${winnerClass(fight, fight.fighter1_id)}">
          <span>${fight.fighter1_name}</span>
          ${winnerDetail(fight, fight.fighter1_id)}
        </div>
        <div class="fighter-name ${winnerClass(fight, fight.fighter2_id)}">
          <span>${fight.fighter2_name}</span>
          ${winnerDetail(fight, fight.fighter2_id)}
        </div>
      </div>
    `;

    if (betMap[fight.id]) {
      el.classList.add('bet');
    }

    // Enable navigation to the fight details page
    if (fight && fight.id) {
      el.classList.add('clickable');
      el.addEventListener('click', () => {
        window.location = `/fight/${fight.id}`;
      });
      // Basic keyboard accessibility
      el.tabIndex = 0;
      el.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          window.location = `/fight/${fight.id}`;
        }
      });
    }

    return el;
  }

  function matchBadge(status) {
    if (status === 'completed') return '<span class="tag" style="color:#8bffc7">Final</span>';
    if (status === 'active') return '<span class="tag" style="color:#ff8888">Live</span>';
    return '<span class="tag" style="color:#8f819c">Upcoming</span>';
  }

  function winnerClass(fight, fighterID) {
    if (fight.status === 'completed' && fight.winner_id) {
      return Number(fight.winner_id) === fighterID ? 'win' : 'loss';
    }
    return '';
  }

  function winnerDetail(fight, fighterID) {
    if (!fight.winner_id || fight.status !== 'completed') return '';
    if (Number(fight.winner_id) !== fighterID) return '';
    if (fight.final_score1 && Number(fight.winner_id) === fight.fighter1_id) {
      return `<span>${fight.final_score1}</span>`;
    }
    if (fight.final_score2 && Number(fight.winner_id) === fight.fighter2_id) {
      return `<span>${fight.final_score2}</span>`;
    }
    return '';
  }

  function renderShowdown(playoffs) {
    const descriptions = [
      'Semifinal · Group A Winner vs Group B Winner',
      'Semifinal · Group C Winner vs Group D Winner',
      'Grand Final · Semifinal Winners'
    ];

    showdownRoot.innerHTML = `
      <div class="showdown-header">
        <div>
          <div class="showdown-title">Road to the Legacy Upgrade</div>
          <div class="showdown-sub">Semifinals &amp; Final · Champion gains +1 random stat post-fight · Commissioner approved</div>
        </div>
        <div class="showdown-sub">Matchups reveal automatically when group standings lock</div>
      </div>
      <div class="showdown-grid">
        ${['22:30', '23:00', '23:30'].map((time, idx) => {
          const fight = playoffs.find(p => p.timeLabel === time);
          const revealed = fight && fight.fighter1_name && fight.fighter2_name;
          return `
            <div class="showdown-card ${revealed && fight && fight.id ? 'clickable' : ''}" ${revealed && fight && fight.id ? `onclick="window.location='/fight/${fight.id}'"` : ''}>
              <div class="showdown-time">${time}</div>
              <div class="showdown-match ${revealed ? 'revealed' : 'pending'}">
                ${revealed ? `${fight.fighter1_name} vs ${fight.fighter2_name}` : '▓▓▓▓▓▓▓▓▓ vs ▓▓▓▓▓▓▓▓▓'}
              </div>
              <div class="showdown-sub" style="margin-top:14px; text-align:center;">
                ${descriptions[idx]}
              </div>
            </div>
          `;
        }).join('')}
      </div>
      <div class="showdown-footer">
        <div class="champion-note">⚡ Legacy Infusion: Champion receives +1 random stat at 23:45</div>
        <div>Watch along · Holo-view pods · Streamer co-cast</div>
      </div>
    `;
  }

  function pollForUpdates() {
    setTimeout(async () => {
      try {
        const res = await fetch('/api/schedule/today');
        if (!res.ok) throw new Error('poll failed');
        const payload = await res.json();
        if (!payload || !Array.isArray(payload.fights) || !payload.meta) {
          throw new Error('invalid response');
        }

        const refreshed = payload.fights.map(enrichFight).sort((a, b) => a.scheduledTime - b.scheduledTime);
        const partitions = partitionGroups(refreshed);
        renderOverview(refreshed, new Date(payload.meta.now));
        renderGroups(partitions.groups, betMap);
        renderShowdown(partitions.playoffs);
      } catch (err) {
        console.error('Saturday poll failed', err);
      } finally {
        pollForUpdates();
      }
    }, 45000);
  }
})();

