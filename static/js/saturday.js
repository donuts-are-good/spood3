// Saturday schedule client logic: renders group standings, timeline,
// playoff cards, and polls for semifinal/final updates.
(function () {
  const groupsRoot = document.querySelector('[data-groups]');
  if (!groupsRoot) return;

  const fightsPayload = loadJSON('#saturday-fights') || [];
  const betMap = loadJSON('#saturday-bets') || {};
  const nowPayload = loadJSON('#saturday-now');
  const now = nowPayload && nowPayload.now ? new Date(nowPayload.now) : new Date();

  const fights = fightsPayload.map(enrichFight).sort((a, b) => a.scheduledTime - b.scheduledTime);

  const partitions = partitionGroups(fights);
  renderGroups(partitions.groups, betMap);
  renderTimeline(fights, betMap);
  renderPlayoffs(partitions.playoffs, betMap);
  renderStatusCards(fights, now);

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

  function renderGroups(groupMap, betMap) {
    const labels = ['A', 'B', 'C', 'D'];
    groupsRoot.innerHTML = '';

    labels.forEach((label) => {
      const fights = groupMap[label];
      if (!fights || !fights.length) return;

      const card = document.createElement('div');
      card.className = 'group-card';

      const title = document.createElement('h3');
      title.textContent = `Group ${label}`;
      card.appendChild(title);

      const standings = computeStandings(fights);
      standings.forEach((row, index) => {
        const standing = document.createElement('div');
        standing.className = 'group-standings';

        const rank = document.createElement('div');
        rank.className = 'rank';
        rank.textContent = index + 1;
        standing.appendChild(rank);

        const fighter = document.createElement('div');
        fighter.className = 'fighter';
        fighter.textContent = row.name;
        if (betMap[row.id]) fighter.classList.add('bet');
        standing.appendChild(fighter);

        const record = document.createElement('div');
        record.className = 'record';
        record.textContent = `${row.wins}-${row.losses}`;
        standing.appendChild(record);

        const diff = document.createElement('div');
        diff.className = 'score';
        diff.textContent = row.diff >= 0 ? `+${row.diff}` : row.diff;
        standing.appendChild(diff);

        card.appendChild(standing);
      });

      fights.forEach((fight) => card.appendChild(renderFightSlot(fight, betMap)));

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
    return rows;

    function ensure(map, id, name) {
      if (!map.has(id)) {
        map.set(id, { id, name, wins: 0, losses: 0, diff: 0, completed: 0, firstWin: null });
      }
      return map.get(id);
    }
  }

  function renderFightSlot(fight, betMap) {
    const slot = document.createElement('div');
    slot.className = 'fight-slot';
    slot.classList.add(classForStatus(fight.status));
    if (betMap[fight.id]) slot.classList.add('bet');

    const time = document.createElement('div');
    time.className = 'time';
    time.textContent = fight.scheduledTime.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
    });
    slot.appendChild(time);

    const names = document.createElement('div');
    names.className = 'names';

    const one = document.createElement('span');
    one.textContent = fight.fighter1_name;
    const two = document.createElement('span');
    two.textContent = fight.fighter2_name;

    if (fight.status === 'completed' && fight.winner_id) {
      const winner = Number(fight.winner_id);
      if (winner === fight.fighter1_id) {
        one.classList.add('winner');
        two.classList.add('loser');
      } else if (winner === fight.fighter2_id) {
        two.classList.add('winner');
        one.classList.add('loser');
      }
    }

    names.appendChild(one);
    names.appendChild(two);
    slot.appendChild(names);

    const cta = document.createElement('div');
    cta.className = 'cta';
    const link = document.createElement('a');
    link.href = fight.status === 'active' ? `/watch/${fight.id}` : `/fight/${fight.id}`;
    link.textContent = fight.status === 'active' ? 'Watch' : 'Details';
    cta.appendChild(link);
    slot.appendChild(cta);

    return slot;
  }

  function classForStatus(status) {
    if (status === 'active') return 'live';
    if (status === 'completed') return 'completed';
    if (status === 'voided') return 'voided';
    return 'upcoming';
  }

  function renderTimeline(fights, betMap) {
    const container = document.getElementById('timeline-slots');
    if (!container) return;
    container.innerHTML = '';

    fights.forEach((fight) => {
      const slot = document.createElement('div');
      slot.className = 'slot';
      slot.classList.add(classForStatus(fight.status));
      if (betMap[fight.id]) slot.classList.add('bet');

      const time = document.createElement('div');
      time.className = 'slot-time';
      time.textContent = fight.scheduledTime.toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
      });
      slot.appendChild(time);

      const names = document.createElement('div');
      names.className = 'slot-names';
      names.textContent = `${fight.fighter1_name} vs ${fight.fighter2_name}`;
      slot.appendChild(names);

      slot.addEventListener('click', () => {
        window.location.href = fight.status === 'active' ? `/watch/${fight.id}` : `/fight/${fight.id}`;
      });

      container.appendChild(slot);
    });
  }

  function renderPlayoffs(playoffs, betMap) {
    const container = document.getElementById('playoffs-card');
    if (!container) return;
    container.innerHTML = '';

    const frames = [
      { time: '22:30', label: 'Semifinal: Group A vs Group B' },
      { time: '23:00', label: 'Semifinal: Group C vs Group D' },
      { time: '23:30', label: 'Final: Winners face off' },
    ];

    frames.forEach((frame) => {
      const fight = playoffs.find((pf) => pf.timeLabel === frame.time);

      const row = document.createElement('div');
      row.className = 'match';

      const time = document.createElement('span');
      time.className = 'time';
      time.textContent = frame.time;
      row.appendChild(time);

      const vs = document.createElement('span');
      vs.className = 'vs';
      const badge = document.createElement('span');
      badge.className = 'badge';

      if (fight) {
        vs.textContent = `${fight.fighter1_name} vs ${fight.fighter2_name}`;
        badge.textContent = fight.status;
        if (fight.status === 'active') badge.classList.add('badge-live');
        else badge.classList.add('badge-upcoming');
        if (betMap[fight.id]) row.classList.add('bet');
      } else {
        vs.textContent = '▓▓▓▓▓▓▓▓▓ vs ▓▓▓▓▓▓▓▓▓';
        badge.textContent = 'upcoming';
        badge.classList.add('badge-upcoming');
      }

      row.appendChild(vs);
      row.appendChild(badge);
      container.appendChild(row);
    });
  }

  function renderStatusCards(fights, now) {
    const liveCard = document.getElementById('live-card');
    const nextCard = document.getElementById('next-card');
    if (!liveCard || !nextCard) return;

    const live = fights.find((f) => f.status === 'active');
    const upcoming = fights.filter((f) => f.status === 'scheduled' && f.scheduledTime >= now);
    const next = upcoming.length ? upcoming[0] : null;

    const liveLabel = liveCard.querySelector('#live-label');
    const watchLink = liveCard.querySelector('#watch-now');
    const progress = liveCard.querySelector('#day-progress');

    if (live) {
      liveLabel.textContent = `${live.fighter1_name} vs ${live.fighter2_name}`;
      watchLink.href = `/watch/${live.id}`;
      watchLink.classList.add('btn-live');
    } else {
      liveLabel.textContent = '—';
      watchLink.href = '#';
      watchLink.classList.remove('btn-live');
    }

    const total = 27;
    const completed = fights.filter((f) => f.status === 'completed' || f.status === 'voided').length;
    progress.style.width = `${Math.min(100, Math.round((completed / total) * 100))}%`;

    const countdown = nextCard.querySelector('#next-countdown');
    const nextLabel = nextCard.querySelector('#next-label');

    if (next) {
      nextLabel.textContent = `${next.fighter1_name} vs ${next.fighter2_name}`;
      startCountdown(countdown, next.scheduledTime);
    } else {
      nextLabel.textContent = '—';
      countdown.textContent = '--:--';
    }
  }

  function startCountdown(el, target) {
    if (!el) return;

    const update = () => {
      const diff = target - new Date();
      if (diff <= 0) {
        el.textContent = '00:00';
        return;
      }
      const minutes = Math.floor(diff / 60000);
      const seconds = Math.floor((diff % 60000) / 1000);
      el.textContent = `${pad(minutes)}:${pad(seconds)}`;
      requestAnimationFrame(update);
    };

    update();
  }

  function pad(n) {
    return n < 10 ? `0${n}` : `${n}`;
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
        renderGroups(partitions.groups, betMap);
        renderTimeline(refreshed, betMap);
        renderPlayoffs(partitions.playoffs, betMap);
        renderStatusCards(refreshed, new Date(payload.meta.now));
      } catch (err) {
        console.error('Saturday poll failed', err);
      } finally {
        pollForUpdates();
      }
    }, 45000);
  }
})();

