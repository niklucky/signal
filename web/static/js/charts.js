function dashboardCharts(endpoint) {
  fetch(endpoint)
    .then(r => r.json())
    .then(data => {
      data.forEach(host => {
        const row = document.querySelector(`tr[data-host-id="${host.host_id}"]`);
        if (!row) return;

        const intervalSec = host.interval_sec || parseInt(row.dataset.interval, 10) || 60;
        const slaCell = row.querySelector('.sla-cell');
        const rtCell = row.querySelector('.rt-cell');
        const hours = fillMissingHours(host.hours || []);

        if (slaCell) renderSLA(slaCell, hours, intervalSec);
        if (rtCell) renderSparkline(rtCell, hours);
      });
    })
    .catch(err => {
      console.error('Failed to load dashboard data', err);
      document.querySelectorAll('.sla-cell, .rt-cell').forEach(cell => {
        cell.innerHTML = '<span class="text-sm text-red-600">Error</span>';
      });
    });
}

function hostChart(endpoint, containerId) {
  const container = document.getElementById(containerId);
  if (!container) return;

  fetch(endpoint)
    .then(r => r.json())
    .then(data => {
      const points = data.points || [];
      if (points.length < 2) {
        container.innerHTML = '<p class="text-sm text-gray-500">Not enough data yet.</p>';
        return;
      }

      const xs = points.map(p => p.ts);
      const ys = points.map(p => p.response_time_ms || 0);
      const opts = {
        width: container.clientWidth,
        height: 180,
        padding: [8, 8, 0, 0],
        axes: [
          { values: (u, splits) => splits.map(ts => fmtTime(ts)) },
          {
            values: (u, splits) => splits.map(v => v == null ? '' : v + 'ms'),
            size: 70,
            gap: 4
          }
        ],
        scales: { x: { time: true } },
        series: [
          { label: 'Time' },
          { label: 'Response time (ms)', stroke: '#4f46e5' }
        ]
      };
      new uPlot(opts, [xs, ys], container);
    })
    .catch(err => {
      container.innerHTML =
        `<p class="text-sm text-red-600">Failed to load chart data: ${escapeHtml(err.message)}</p>`;
    });
}

function renderSLA(cell, hours, intervalSec) {
  cell.innerHTML = '';

  const wrap = document.createElement('div');
  wrap.className = 'flex items-center gap-2 min-w-0';

  const bar = document.createElement('div');
  bar.className = 'flex items-end gap-1 h-6 min-w-0 flex-1 overflow-hidden';

  const now = new Date();
  now.setMinutes(0, 0, 0);

  let totalChecks = 0;
  let totalSuccess = 0;

  const bucketMap = new Map(hours.map(h => [h.ts, h]));

  for (let i = 23; i >= 0; i--) {
    const d = new Date(now.getTime() - i * 3600000);
    const hourKey = Math.floor(d.getTime() / 1000);
    const bucket = bucketMap.get(hourKey) || { ts: hourKey, response_time_ms: 0, success_count: 0, failure_count: 0 };
    const total = bucket.success_count + bucket.failure_count;

    totalChecks += total;
    totalSuccess += bucket.success_count;

    const el = document.createElement('div');
    el.className = 'w-2 rounded-sm';
    el.style.height = '100%';

    let colorClass = 'bg-gray-200';
    if (total > 0) {
      const failPct = bucket.failure_count / total;
      const failMinutes = (bucket.failure_count * intervalSec) / 60;
      if (failPct > 0.05) {
        colorClass = 'bg-red-500';
      } else if (failMinutes > 1) {
        colorClass = 'bg-yellow-400';
      } else {
        colorClass = 'bg-green-500';
      }
    }
    el.classList.add(colorClass);

    const timeLabel = d.toLocaleString([], {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
    el.title = `${timeLabel}: ${bucket.success_count}/${total} ok, ${bucket.failure_count} failed`;
    bar.appendChild(el);
  }

  wrap.appendChild(bar);

  const pctSpan = document.createElement('span');
  pctSpan.className = 'text-sm text-gray-700 tabular-nums flex-shrink-0';
  if (totalChecks > 0) {
    pctSpan.textContent = `${((totalSuccess / totalChecks) * 100).toFixed(1)}%`;
  } else {
    pctSpan.textContent = '—';
  }
  wrap.appendChild(pctSpan);

  cell.appendChild(wrap);
}

function renderSparkline(cell, hours) {
  cell.innerHTML = '';

  const nonEmpty = (hours || []).filter(h => h.response_time_ms > 0);
  if (nonEmpty.length < 2) {
    cell.innerHTML = '<span class="text-sm text-gray-400">—</span>';
    return;
  }

  const ys = nonEmpty.map(h => h.response_time_ms);
  const xs = nonEmpty.map((_, i) => i);
  const avg = ys.reduce((a, b) => a + b, 0) / ys.length;

  const wrap = document.createElement('div');
  wrap.className = 'flex items-center gap-2 min-w-0';

  const valueSpan = document.createElement('span');
  valueSpan.className = 'text-sm text-gray-900 tabular-nums flex-shrink-0';
  valueSpan.textContent = `${Math.round(avg)}ms`;
  wrap.appendChild(valueSpan);

  const opts = {
    width: 80,
    height: 24,
    pxAlign: false,
    cursor: { show: false },
    select: { show: false },
    legend: { show: false },
    scales: { x: { time: false } },
    axes: [{ show: false }, { show: false }],
    series: [
      {},
      {
        stroke: '#4f46e5',
        fill: '#c7d2fe',
        width: 1
      }
    ]
  };

  const canvas = new uPlot(opts, [xs, ys]).ctx.canvas;
  canvas.style.width = '80px';
  canvas.style.height = '24px';
  canvas.style.display = 'block';

  wrap.appendChild(canvas);
  cell.appendChild(wrap);
}

function fillMissingHours(hours) {
  if (!hours || hours.length === 0) return [];

  const map = new Map(hours.map(h => [h.ts, h]));
  const minTs = hours[0].ts;
  const maxTs = hours[hours.length - 1].ts;
  const filled = [];

  for (let ts = minTs; ts <= maxTs; ts += 3600) {
    if (map.has(ts)) {
      filled.push(map.get(ts));
    } else {
      filled.push({ ts, response_time_ms: 0, success_count: 0, failure_count: 0 });
    }
  }

  return filled;
}

function fmtTime(ts) {
  const d = new Date(ts * 1000);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}
