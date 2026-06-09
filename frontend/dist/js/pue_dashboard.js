var _charts = {
  pue: null,
  ranking: null,
  detailTemp: null,
  detailFlow: null,
  detailPower: null,
  detailCop: null
};

var _pueData = [];
var _devices = [];
var _alerts = [];

function copColorCSS(cop) {
  if (cop > 6) return 'var(--green)';
  if (cop >= 4) return 'var(--yellow)';
  return 'var(--red)';
}

function copClass(cop) {
  if (cop > 6) return 'cop-green';
  if (cop >= 4) return 'cop-yellow';
  return 'cop-red';
}

function formatTime(d) {
  return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
}

function formatDateTime(d) {
  return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0') + ' ' + String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0') + ':' + String(d.getSeconds()).padStart(2, '0');
}

export function init(stateRef) {
  _devices = stateRef.devices;
  _pueData = stateRef.pueData;
  _alerts = stateRef.alerts;
}

export function renderPUEChart(pueData) {
  var data = pueData || _pueData;
  var statsEl = document.getElementById('pue-stats');
  var latest = data[data.length - 1];
  if (!latest) return;

  statsEl.innerHTML =
    '<div class="stat-card"><div class="stat-card-label">当前PUE</div><div class="stat-card-value" style="color:var(--accent)">' + latest.pue.toFixed(2) + '</div></div>' +
    '<div class="stat-card"><div class="stat-card-label">IT负载(kW)</div><div class="stat-card-value" style="color:var(--accent2)">' + latest.itPower.toFixed(0) + '</div></div>' +
    '<div class="stat-card"><div class="stat-card-label">制冷功率(kW)</div><div class="stat-card-value" style="color:var(--yellow)">' + latest.coolingPower.toFixed(0) + '</div></div>' +
    '<div class="stat-card"><div class="stat-card-label">配电损耗(kW)</div><div class="stat-card-value" style="color:#ff8844">' + (latest.distributionLoss || 0).toFixed(0) + '</div></div>' +
    '<div class="stat-card"><div class="stat-card-label">其他基础设施(kW)</div><div class="stat-card-value" style="color:#aa88ff">' + (latest.otherInfraPower || 0).toFixed(0) + '</div></div>' +
    '<div class="stat-card"><div class="stat-card-label">总设施功率(kW)</div><div class="stat-card-value">' + (latest.totalFacilityPower || 0).toFixed(0) + '</div></div>';

  var ctx = document.getElementById('pue-chart').getContext('2d');
  if (_charts.pue) _charts.pue.destroy();

  var labels = data.map(function (d) { return formatTime(d.time); });
  var values = data.map(function (d) { return d.pue; });
  var gradient = ctx.createLinearGradient(0, 0, 0, 400);
  gradient.addColorStop(0, 'rgba(0,212,255,0.3)');
  gradient.addColorStop(1, 'rgba(0,212,255,0.01)');

  _charts.pue = new Chart(ctx, {
    type: 'line',
    data: {
      labels: labels,
      datasets: [{
        label: 'PUE',
        data: values,
        borderColor: '#00d4ff',
        backgroundColor: gradient,
        fill: true,
        tension: 0.4,
        pointRadius: 3,
        pointBackgroundColor: '#00d4ff',
        borderWidth: 2
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { ticks: { color: '#8899bb', maxTicksLimit: 12 }, grid: { color: 'rgba(30,42,69,0.5)' } },
        y: { min: 1.2, max: 1.7, ticks: { color: '#8899bb' }, grid: { color: 'rgba(30,42,69,0.5)' } }
      }
    },
    plugins: [{
      id: 'pueRefLines',
      afterDraw: function (chart) {
        var yAxis = chart.scales.y;
        var ctx2 = chart.ctx;
        [1.4, 1.5].forEach(function (val, idx) {
          var y = yAxis.getPixelForValue(val);
          ctx2.save();
          ctx2.strokeStyle = idx === 0 ? 'rgba(255,204,0,0.5)' : 'rgba(255,68,85,0.5)';
          ctx2.lineWidth = 1;
          ctx2.setLineDash([5, 5]);
          ctx2.beginPath();
          ctx2.moveTo(chart.chartArea.left, y);
          ctx2.lineTo(chart.chartArea.right, y);
          ctx2.stroke();
          ctx2.fillStyle = idx === 0 ? 'rgba(255,204,0,0.7)' : 'rgba(255,68,85,0.7)';
          ctx2.font = '11px sans-serif';
          ctx2.fillText('PUE ' + val, chart.chartArea.right - 50, y - 4);
          ctx2.restore();
        });
      }
    }]
  });
}

export function renderSankey(devices, allocationData) {
  var devs = devices || _devices;
  var svg = d3.select('#sankey-svg');
  svg.selectAll('*').remove();
  var container = document.querySelector('.sankey-container');
  var width = container.clientWidth - 48;
  var height = Math.max(300, container.clientHeight - 220);
  svg.attr('width', width).attr('height', height);

  var chillers = devs.filter(function (d) { return d.type === 'chiller'; });
  var nodes = [];
  var links = [];

  if (allocationData && allocationData.length > 0) {
    var areaNames = allocationData.map(function (a) { return a.area; });
    chillers.forEach(function (c) {
      nodes.push({ name: c.code, type: 'source' });
    });
    areaNames.forEach(function (a) {
      nodes.push({ name: a, type: 'target' });
    });
    allocationData.forEach(function (alloc, ai) {
      var totalAlloc = alloc.allocatedCooling || alloc.heatLoad || 100;
      var perChiller = totalAlloc / chillers.length;
      chillers.forEach(function (c, ci) {
        var val = perChiller * (0.7 + Math.random() * 0.6);
        links.push({ source: ci, target: chillers.length + ai, value: val });
      });
    });
  } else {
    var areas = ['A区', 'B区', 'C区', 'D区', 'E区', 'F区', 'G区', 'H区'];
    chillers.forEach(function (c) {
      nodes.push({ name: c.code, type: 'source' });
    });
    areas.forEach(function (a) {
      nodes.push({ name: a, type: 'target' });
    });
    chillers.forEach(function (c, ci) {
      var numLinks = 2 + Math.floor(Math.random() * 3);
      for (var j = 0; j < numLinks; j++) {
        var ai = (ci + j) % areas.length;
        links.push({ source: ci, target: chillers.length + ai, value: 50 + Math.random() * 200 });
      }
    });
  }

  var sankeyGen = d3.sankey()
    .nodeId(function (d) { return d.index; })
    .nodeWidth(16)
    .nodePadding(12)
    .extent([[1, 1], [width - 1, height - 1]]);

  var graph = sankeyGen({
    nodes: nodes.map(function (d, i) { return Object.assign({}, d, { index: i }); }),
    links: links.map(function (d) { return Object.assign({}, d); })
  });

  var blueShades = ['#1a5276', '#1f6f8b', '#2596be', '#2ea8d8', '#48b8d8', '#5cc8e0', '#70d8e8', '#8ae8f0'];
  var greenShades = ['#1a6b3c', '#1e8c4e', '#23a85e', '#28c46e', '#32d87e', '#42e88e', '#52f09e', '#62f8ae'];

  svg.append('g')
    .selectAll('path')
    .data(graph.links)
    .join('path')
    .attr('d', d3.sankeyLinkHorizontal())
    .attr('fill', 'none')
    .attr('stroke', function (d) { return d.source.type === 'source' ? blueShades[d.source.index % blueShades.length] : '#4488ff'; })
    .attr('stroke-opacity', 0.4)
    .attr('stroke-width', function (d) { return Math.max(1, d.width); });

  svg.append('g')
    .selectAll('rect')
    .data(graph.nodes)
    .join('rect')
    .attr('x', function (d) { return d.x0; })
    .attr('y', function (d) { return d.y0; })
    .attr('height', function (d) { return Math.max(1, d.y1 - d.y0); })
    .attr('width', function (d) { return d.x1 - d.x0; })
    .attr('fill', function (d) { return d.type === 'source' ? blueShades[d.index % blueShades.length] : greenShades[d.index % greenShades.length]; })
    .attr('stroke', '#0a0e1a');

  svg.append('g')
    .selectAll('text')
    .data(graph.nodes)
    .join('text')
    .attr('x', function (d) { return d.x0 < width / 2 ? d.x1 + 4 : d.x0 - 4; })
    .attr('y', function (d) { return (d.y0 + d.y1) / 2; })
    .attr('dy', '0.35em')
    .attr('text-anchor', function (d) { return d.x0 < width / 2 ? 'start' : 'end'; })
    .attr('fill', '#e0e6f0')
    .attr('font-size', '11px')
    .text(function (d) { return d.name; });

  var tableEl = document.getElementById('sankey-table');
  var tableData = allocationData && allocationData.length > 0
    ? allocationData
    : ['A区', 'B区', 'C区', 'D区', 'E区', 'F区', 'G区', 'H区'].map(function (a) {
        return { area: a, heatLoad: 200 + Math.random() * 300, allocatedCooling: 180 + Math.random() * 280, setpointTemp: 24 + Math.random() * 2, actualTemp: 23 + Math.random() * 4 };
      });

  var html = '<table><thead><tr><th>区域</th><th>热负荷(kW)</th><th>分配冷量(kW)</th><th>设定温度(℃)</th><th>实际温度(℃)</th></tr></thead><tbody>';
  tableData.forEach(function (row) {
    html += '<tr><td>' + row.area + '</td><td>' + (row.heatLoad || 0).toFixed(0) + '</td><td>' + (row.allocatedCooling || 0).toFixed(0) + '</td><td>' + (row.setpointTemp || 0).toFixed(1) + '</td><td>' + (row.actualTemp || 0).toFixed(1) + '</td></tr>';
  });
  html += '</tbody></table>';
  tableEl.innerHTML = html;
}

export function renderRanking(devices) {
  var devs = devices || _devices;
  var sorted = devs.slice().sort(function (a, b) { return b.cop - a.cop; });

  var ctx = document.getElementById('ranking-chart').getContext('2d');
  if (_charts.ranking) _charts.ranking.destroy();

  var top30 = sorted.slice(0, 30);
  _charts.ranking = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: top30.map(function (d) { return d.code; }),
      datasets: [{
        label: 'COP',
        data: top30.map(function (d) { return d.cop; }),
        backgroundColor: top30.map(function (d) {
          return d.cop > 6 ? 'rgba(0,255,136,0.7)' : d.cop >= 4 ? 'rgba(255,204,0,0.7)' : 'rgba(255,68,85,0.7)';
        }),
        borderWidth: 0,
        barPercentage: 0.7
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      indexAxis: 'y',
      plugins: { legend: { display: false } },
      scales: {
        x: { min: 0, max: 8, ticks: { color: '#8899bb' }, grid: { color: 'rgba(30,42,69,0.5)' } },
        y: { ticks: { color: '#8899bb', font: { size: 9 } }, grid: { display: false } }
      }
    }
  });

  var tableEl = document.getElementById('ranking-table');
  var typeMap = { chiller: '冷水机组', cooling_tower: '冷却塔', precision_ac: '精密空调', cdu: '列间空调' };
  var html = '<table><thead><tr><th>设备编码</th><th>设备名称</th><th>类型</th><th>COP</th><th>功率(kW)</th><th>冷量(kW)</th><th>状态</th></tr></thead><tbody>';
  sorted.forEach(function (d) {
    html += '<tr><td>' + d.code + '</td><td>' + d.name + '</td><td>' + (typeMap[d.type] || d.type) + '</td><td class="' + copClass(d.cop) + '">' + d.cop.toFixed(2) + '</td><td>' + d.power.toFixed(0) + '</td><td>' + d.cooling.toFixed(0) + '</td><td>' + (d.status === 'running' ? '运行' : '维护') + '</td></tr>';
  });
  html += '</tbody></table>';
  tableEl.innerHTML = html;
}

export function renderAlerts(alerts) {
  var data = alerts || _alerts;
  var tbody = document.getElementById('alert-tbody');
  var html = '';
  data.forEach(function (a) {
    html += '<tr>' +
      '<td>' + formatDateTime(a.time) + '</td>' +
      '<td><span class="badge badge-' + a.level + '">' + (a.level === 1 ? '一级' : '二级') + '</span></td>' +
      '<td>' + a.device + '</td>' +
      '<td>' + a.type + '</td>' +
      '<td>' + a.message + '</td>' +
      '<td>' + a.value + '</td>' +
      '<td>' + a.threshold + '</td>' +
      '<td><button class="ack-btn" data-id="' + a.id + '"' + (a.acknowledged ? ' disabled' : '') + '>' + (a.acknowledged ? '已确认' : '确认') + '</button></td>' +
      '</tr>';
  });
  tbody.innerHTML = html;

  tbody.querySelectorAll('.ack-btn').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var id = parseInt(this.dataset.id);
      acknowledgeAlert(id, this);
    });
  });
}

function acknowledgeAlert(id, btn) {
  fetch('/api/alerts/' + id + '/acknowledge', { method: 'PUT' })
    .then(function (r) { return r.json(); })
    .then(function () {
      var alert = _alerts.find(function (a) { return a.id === id; });
      if (alert) alert.acknowledged = true;
      btn.disabled = true;
      btn.textContent = '已确认';
    })
    .catch(function () {
      var alert = _alerts.find(function (a) { return a.id === id; });
      if (alert) alert.acknowledged = true;
      btn.disabled = true;
      btn.textContent = '已确认';
    });
}

export function renderDeviceDetail(device) {
  document.getElementById('device-detail-title').textContent = device.code + ' 详情';
  var typeMap = { chiller: '冷水机组', cooling_tower: '冷却塔', precision_ac: '精密空调', cdu: '列间空调' };
  var infoEl = document.getElementById('device-info');
  infoEl.innerHTML =
    '<div class="info-row"><span class="info-label">设备编码</span><span class="info-value">' + device.code + '</span></div>' +
    '<div class="info-row"><span class="info-label">设备名称</span><span class="info-value">' + device.name + '</span></div>' +
    '<div class="info-row"><span class="info-label">设备类型</span><span class="info-value">' + (typeMap[device.type] || device.type) + '</span></div>' +
    '<div class="info-row"><span class="info-label">所属区域</span><span class="info-value">' + device.area + '</span></div>' +
    '<div class="info-row"><span class="info-label">额定功率(kW)</span><span class="info-value">' + device.ratedPower.toFixed(0) + '</span></div>' +
    '<div class="info-row"><span class="info-label">额定冷量(kW)</span><span class="info-value">' + device.ratedCooling.toFixed(0) + '</span></div>' +
    '<div class="info-row"><span class="info-label">当前COP</span><span class="info-value ' + copClass(device.cop) + '">' + device.cop.toFixed(2) + '</span></div>' +
    '<div class="info-row"><span class="info-label">状态</span><span class="info-value">' + (device.status === 'running' ? '运行' : '维护') + '</span></div>';

  fetch('/api/devices/' + device.id + '/data?hours=24')
    .then(function (r) { return r.json(); })
    .then(function (apiData) {
      if (Array.isArray(apiData) && apiData.length > 0) {
        buildDetailCharts(device, apiData);
      } else {
        buildDetailChartsGenerated(device);
      }
    })
    .catch(function () {
      buildDetailChartsGenerated(device);
    });
}

function buildDetailCharts(device, apiData) {
  var hours = apiData.map(function (d) {
    var t = d.timestamp ? new Date(d.timestamp) : new Date();
    return formatTime(t);
  });
  var supplyData = apiData.map(function (d) { return d.supply_temp || d.supplyTemp || device.supplyTemp; });
  var returnData = apiData.map(function (d) { return d.return_temp || d.returnTemp || device.returnTemp; });
  var flowData = apiData.map(function (d) { return d.flow || device.flow; });
  var powerData = apiData.map(function (d) { return d.power || device.power; });
  var copData = apiData.map(function (d) { return d.cop || device.cop; });

  renderDetailChartsInner(hours, supplyData, returnData, flowData, powerData, copData);
}

function buildDetailChartsGenerated(device) {
  var hours = [];
  for (var i = 0; i < 24; i++) hours.push(String(i).padStart(2, '0') + ':00');

  var supplyData = gen24hSeries(device.supplyTemp, 2);
  var returnData = gen24hSeries(device.returnTemp, 2);
  var flowData = gen24hSeries(device.flow, 10);
  var powerData = gen24hSeries(device.power, 30);
  var copData = gen24hSeries(device.cop, 2);

  renderDetailChartsInner(hours, supplyData, returnData, flowData, powerData, copData);
}

function gen24hSeries(base, variance) {
  var data = [];
  for (var i = 0; i < 24; i++) data.push(base - variance / 2 + Math.random() * variance);
  return data;
}

function renderDetailChartsInner(hours, supplyData, returnData, flowData, powerData, copData) {
  var chartOpts = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: { legend: { display: true, labels: { color: '#8899bb', font: { size: 9 } } } },
    scales: {
      x: { ticks: { color: '#8899bb', font: { size: 8 }, maxTicksLimit: 8 }, grid: { color: 'rgba(30,42,69,0.3)' } },
      y: { ticks: { color: '#8899bb', font: { size: 8 } }, grid: { color: 'rgba(30,42,69,0.3)' } }
    }
  };

  if (_charts.detailTemp) _charts.detailTemp.destroy();
  if (_charts.detailFlow) _charts.detailFlow.destroy();
  if (_charts.detailPower) _charts.detailPower.destroy();
  if (_charts.detailCop) _charts.detailCop.destroy();

  _charts.detailTemp = new Chart(document.getElementById('dc-temp').getContext('2d'), {
    type: 'line',
    data: {
      labels: hours,
      datasets: [
        { label: '供水', data: supplyData, borderColor: '#00d4ff', borderWidth: 1.5, pointRadius: 0, tension: 0.3 },
        { label: '回水', data: returnData, borderColor: '#ff8844', borderWidth: 1.5, pointRadius: 0, tension: 0.3 }
      ]
    },
    options: chartOpts
  });

  _charts.detailFlow = new Chart(document.getElementById('dc-flow').getContext('2d'), {
    type: 'line',
    data: {
      labels: hours,
      datasets: [
        { label: '流量', data: flowData, borderColor: '#44cc88', borderWidth: 1.5, pointRadius: 0, tension: 0.3, fill: true, backgroundColor: 'rgba(68,204,136,0.1)' }
      ]
    },
    options: chartOpts
  });

  _charts.detailPower = new Chart(document.getElementById('dc-power').getContext('2d'), {
    type: 'line',
    data: {
      labels: hours,
      datasets: [
        { label: '功率', data: powerData, borderColor: '#ffcc00', borderWidth: 1.5, pointRadius: 0, tension: 0.3, fill: true, backgroundColor: 'rgba(255,204,0,0.1)' }
      ]
    },
    options: chartOpts
  });

  _charts.detailCop = new Chart(document.getElementById('dc-cop').getContext('2d'), {
    type: 'line',
    data: {
      labels: hours,
      datasets: [
        { label: 'COP', data: copData, borderColor: '#aa66ff', borderWidth: 1.5, pointRadius: 0, tension: 0.3, fill: true, backgroundColor: 'rgba(170,102,255,0.1)' }
      ]
    },
    options: Object.assign({}, chartOpts, {
      plugins: Object.assign({}, chartOpts.plugins)
    }),
    plugins: [{
      id: 'copRefLines',
      afterDraw: function (chart) {
        var yAxis = chart.scales.y;
        var ctx2 = chart.ctx;
        [4, 6].forEach(function (val, idx) {
          var y = yAxis.getPixelForValue(val);
          if (y < chart.chartArea.top || y > chart.chartArea.bottom) return;
          ctx2.save();
          ctx2.strokeStyle = idx === 0 ? 'rgba(255,68,85,0.4)' : 'rgba(0,255,136,0.4)';
          ctx2.lineWidth = 1;
          ctx2.setLineDash([3, 3]);
          ctx2.beginPath();
          ctx2.moveTo(chart.chartArea.left, y);
          ctx2.lineTo(chart.chartArea.right, y);
          ctx2.stroke();
          ctx2.restore();
        });
      }
    }]
  });
}

export function destroyCharts() {
  var keys = Object.keys(_charts);
  for (var i = 0; i < keys.length; i++) {
    if (_charts[keys[i]]) {
      _charts[keys[i]].destroy();
      _charts[keys[i]] = null;
    }
  }
}
