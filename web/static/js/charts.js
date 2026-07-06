function dashboardCharts(endpoint) {
  fetch(endpoint)
    .then(r => r.json())
    .then(data => {
      const container = document.getElementById('dashboard-charts');
      container.innerHTML = '';
      if (!data.length) {
        container.innerHTML = '<p class="text-sm text-gray-500">No hosts configured.</p>';
        return;
      }

      data.forEach(host => {
        const wrapper = document.createElement('div');
        wrapper.className = 'bg-white rounded-lg shadow-sm border border-gray-200 p-4';

        const header = document.createElement('div');
        header.className = 'flex items-center justify-between mb-3';
        const status = host.active
          ? '<span class="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">Active</span>'
          : '<span class="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800">Paused</span>';
        header.innerHTML = `<h3 class="font-medium text-gray-900">${escapeHtml(host.name)}</h3>${status}`;
        wrapper.appendChild(header);

        const meta = document.createElement('p');
        meta.className = 'text-xs text-gray-500 mb-3';
        meta.textContent = `${host.url} · ${host.successes_24h} ok / ${host.failures_24h} failed (24h)`;
        wrapper.appendChild(meta);

        const chartDiv = document.createElement('div');
        chartDiv.className = 'w-full';
        wrapper.appendChild(chartDiv);
        container.appendChild(wrapper);

        const points = host.points || [];
        if (points.length < 2) {
          chartDiv.innerHTML = '<p class="text-sm text-gray-500">Not enough data yet.</p>';
          return;
        }

        const xs = points.map(p => p.ts);
        const ys = points.map(p => p.response_time_ms || 0);
        const opts = {
          width: chartDiv.clientWidth,
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
        new uPlot(opts, [xs, ys], chartDiv);
      });
    })
    .catch(err => {
      document.getElementById('dashboard-charts').innerHTML =
        `<p class="text-sm text-red-600">Failed to load chart data: ${escapeHtml(err.message)}</p>`;
    });
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
