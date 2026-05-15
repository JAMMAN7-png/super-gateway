package router

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Super AI Gateway</title>

<!-- Mantine CSS design system -->
<link rel="stylesheet" href="https://esm.sh/@mantine/core@7.10.0/styles.css">
<link rel="stylesheet" href="https://esm.sh/@mantine/notifications@7.10.0/styles.css">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">

<!-- React UMD -->
<script crossorigin src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
<script crossorigin src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>

<style>
*, *::before, *::after { box-sizing: border-box; }
body { margin: 0; font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif; background: #0a0a14; color: #c9d1d9; }
#root { min-height: 100vh; display: flex; }

/* Mantine-style tokens */
:root {
  --bg-primary: #0a0a14;
  --bg-secondary: #141517;
  --bg-card: #1a1b1e;
  --bg-hover: #25262b;
  --border-color: #373A40;
  --text-primary: #c9d1d9;
  --text-secondary: #909296;
  --text-muted: #5C5F66;
  --blue: #228be6;
  --blue-light: #4dabf7;
  --green: #40c057;
  --red: #fa5252;
  --yellow: #fab005;
  --violet: #7950f2;
  --cyan: #15aabf;
  --radius: 8px;
  --radius-sm: 4px;
  --shadow: 0 1px 3px rgba(0,0,0,0.3);
}

/* Layout */
.shell { display: flex; width: 100%; min-height: 100vh; }
.navbar { width: 220px; min-width: 220px; background: #141517; border-right: 1px solid #373A40; display: flex; flex-direction: column; }
.navbar-brand { padding: 14px 16px; display: flex; align-items: center; gap: 10px; border-bottom: 1px solid #373A40; }
.navbar-brand-icon { width: 36px; height: 36px; border-radius: 8px; background: linear-gradient(135deg, #228be6, #15aabf); display: flex; align-items: center; justify-content: center; font-weight: 800; font-size: 16px; color: #fff; }
.navbar-links { flex: 1; padding: 8px; display: flex; flex-direction: column; gap: 2px; }
.nav-link { display: flex; align-items: center; gap: 10px; padding: 10px 12px; border-radius: 6px; cursor: pointer; font-size: 14px; color: #909296; text-decoration: none; transition: all 0.15s; }
.nav-link:hover { background: #25262b; color: #c9d1d9; }
.nav-link.active { background: rgba(34,139,230,0.12); color: #4dabf7; }
.navbar-footer { border-top: 1px solid #373A40; padding: 12px 16px; }
.main-content { flex: 1; padding: 24px; overflow-y: auto; max-height: 100vh; }

/* Cards */
.card { background: #1a1b1e; border: 1px solid #373A40; border-radius: 8px; padding: 20px; }
.card-badge { display: inline-flex; align-items: center; gap: 4px; padding: 2px 10px; border-radius: 100px; font-size: 11px; font-weight: 600; }
.card-badge-green { background: rgba(64,192,87,0.12); color: #40c057; }
.card-badge-red { background: rgba(250,82,82,0.12); color: #fa5252; }
.card-badge-blue { background: rgba(34,139,230,0.12); color: #4dabf7; }
.card-badge-yellow { background: rgba(250,176,5,0.12); color: #fab005; }

/* Stats grid */
.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 24px; }
.stat-card { background: #1a1b1e; border: 1px solid #373A40; border-radius: 8px; padding: 20px; transition: transform 0.15s; cursor: default; }
.stat-card:hover { transform: translateY(-2px); }
.stat-card-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px; }
.stat-card-label { font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; color: #5C5F66; font-weight: 600; }
.stat-card-value { font-size: 28px; font-weight: 700; line-height: 1.2; }
.stat-card-sub { font-size: 12px; color: #5C5F66; margin-top: 6px; }
.stat-icon { width: 40px; height: 40px; border-radius: 40px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }

/* Tables */
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; padding: 10px 12px; color: #5C5F66; font-weight: 600; font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; border-bottom: 1px solid #373A40; }
td { padding: 10px 12px; border-bottom: 1px solid rgba(55,58,64,0.4); }
tr:hover td { background: rgba(255,255,255,0.02); }
code { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; background: rgba(255,255,255,0.05); padding: 2px 6px; border-radius: 4px; }

/* Forms */
.form-group { margin-bottom: 16px; }
.form-label { display: block; font-size: 13px; font-weight: 500; margin-bottom: 6px; color: #c9d1d9; }
.form-input, .form-textarea, .form-select { width: 100%; padding: 9px 12px; background: #25262b; border: 1px solid #373A40; border-radius: 6px; color: #c9d1d9; font-size: 14px; font-family: inherit; transition: border-color 0.15s; }
.form-input:focus, .form-textarea:focus, .form-select:focus { outline: none; border-color: #228be6; }
.form-textarea { resize: vertical; min-height: 80px; }
.form-select { cursor: pointer; }
.form-multi-select { position: relative; }
.form-multi-select-inner { display: flex; flex-wrap: wrap; gap: 4px; padding: 6px 8px; background: #25262b; border: 1px solid #373A40; border-radius: 6px; cursor: pointer; min-height: 38px; }
.form-multi-select-tag { display: inline-flex; align-items: center; gap: 4px; padding: 2px 6px; background: rgba(34,139,230,0.2); border: 1px solid rgba(34,139,230,0.3); border-radius: 4px; font-size: 12px; }
.form-multi-select-tag-remove { cursor: pointer; font-size: 14px; line-height: 1; opacity: 0.6; }
.form-multi-select-tag-remove:hover { opacity: 1; }
.form-multi-select-dropdown { position: absolute; top: 100%; left: 0; right: 0; z-index: 100; background: #25262b; border: 1px solid #373A40; border-radius: 6px; margin-top: 4px; max-height: 240px; overflow-y: auto; }
.form-multi-select-option { padding: 8px 12px; cursor: pointer; font-size: 13px; }
.form-multi-select-option:hover { background: rgba(34,139,230,0.1); }
.form-multi-select-option.selected { background: rgba(34,139,230,0.15); color: #4dabf7; }
.form-multi-select-search { width: 100%; min-width: 80px; border: none; background: transparent; color: #c9d1d9; font-size: 13px; font-family: inherit; outline: none; padding: 2px; }

/* Buttons */
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 16px; border-radius: 6px; font-size: 14px; font-weight: 500; font-family: inherit; cursor: pointer; border: none; transition: all 0.15s; }
.btn-primary { background: #228be6; color: #fff; }
.btn-primary:hover { background: #1c7ed6; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-subtle { background: transparent; color: #909296; }
.btn-subtle:hover { background: rgba(255,255,255,0.05); color: #c9d1d9; }

/* Badge inline */
.badge { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; border-radius: 100px; font-size: 11px; font-weight: 600; white-space: nowrap; }
.badge-green { background: rgba(64,192,87,0.15); color: #40c057; }
.badge-red { background: rgba(250,82,82,0.15); color: #fa5252; }
.badge-blue { background: rgba(34,139,230,0.15); color: #4dabf7; }
.badge-yellow { background: rgba(250,176,5,0.15); color: #fab005; }
.badge-gray { background: rgba(144,146,150,0.15); color: #909296; }
.badge-dot::before { content: ''; width: 6px; height: 6px; border-radius: 6px; display: inline-block; }
.badge-dot.badge-green::before { background: #40c057; }
.badge-dot.badge-red::before { background: #fa5252; }
.badge-dot.badge-blue::before { background: #4dabf7; }

/* Response cards */
.response-grid { display: grid; gap: 16px; margin-top: 16px; }
.response-grid-1 { grid-template-columns: 1fr; }
.response-grid-2 { grid-template-columns: repeat(2, 1fr); }
.response-grid-3 { grid-template-columns: repeat(3, 1fr); }
.response-grid-4 { grid-template-columns: repeat(4, 1fr); }
.response-card { background: #1a1b1e; border: 1px solid #373A40; border-radius: 8px; overflow: hidden; }
.response-card-header { padding: 12px 16px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #373A40; }
.response-card-body { padding: 16px; max-height: 400px; overflow-y: auto; }
.response-card-body pre { margin: 0; font-size: 13px; line-height: 1.6; white-space: pre-wrap; word-break: break-word; font-family: 'SF Mono', 'Fira Code', monospace; }
.response-card-footer { padding: 10px 16px; border-top: 1px solid #373A40; display: flex; gap: 16px; font-size: 12px; color: #5C5F66; }
.response-card.loading { border-color: #228be6; animation: pulse-border 1.5s ease-in-out infinite; }
.response-card.done { border-left: 3px solid #40c057; }
.response-card.error { border-left: 3px solid #fa5252; }

/* Dialogs */
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.6); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.modal-content { background: #1a1b1e; border: 1px solid #373A40; border-radius: 8px; padding: 24px; min-width: 400px; max-width: 90vw; max-height: 80vh; overflow-y: auto; box-shadow: 0 20px 60px rgba(0,0,0,0.5); }

/* Scrollbar */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: #373A40; border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: #5C5F66; }

/* Skeleton loading */
.skeleton { height: 16px; background: linear-gradient(90deg, #25262b 25%, #373A40 50%, #25262b 75%); background-size: 200% 100%; border-radius: 4px; animation: shimmer 1.5s infinite; }
.skeleton-sm { height: 12px; width: 60%; }
.skeleton-lg { height: 24px; }

@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }
@keyframes pulse-border { 0%,100% { border-color: #228be6; } 50% { border-color: #4dabf7; } }

/* Utility */
.flex { display: flex; }
.flex-col { flex-direction: column; }
.items-center { align-items: center; }
.justify-between { justify-content: space-between; }
.justify-end { justify-content: flex-end; }
.gap-xs { gap: 4px; }
.gap-sm { gap: 8px; }
.gap-md { gap: 16px; }
.mb-sm { margin-bottom: 8px; }
.mb-md { margin-bottom: 16px; }
.mb-lg { margin-bottom: 24px; }
.mt-sm { margin-top: 8px; }
.mt-md { margin-top: 16px; }
.p-sm { padding: 8px; }
.p-md { padding: 16px; }
.text-sm { font-size: 13px; }
.text-xs { font-size: 12px; }
.text-lg { font-size: 16px; }
.text-muted { color: #5C5F66; }
.text-dimmed { color: #909296; }
.text-center { text-align: center; }
.font-mono { font-family: 'SF Mono', 'Fira Code', monospace; }
.truncate { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.w-full { width: 100%; }
.grid-2 { display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; }
.grid-3 { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.grid-4 { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
@media (max-width: 1200px) { .grid-2, .grid-3, .grid-4 { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 768px) { .stats-grid { grid-template-columns: 1fr 1fr; } .grid-2, .grid-3, .grid-4 { grid-template-columns: 1fr; } .navbar { width: 60px; min-width: 60px; } .nav-link span { display: none; } .navbar-brand span { display: none; } }
@media (max-width: 480px) { .stats-grid { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<div id="root"></div>
<script>
(function() {
'use strict';

const { createElement: h, useState, useEffect, useCallback, useRef, useMemo } = React;

const API_BASE = '';
const REFRESH_INTERVAL = 5000;

// ---- Icons (inline SVGs) ----
const Icons = {
  dashboard: h('svg', { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('rect', { x: 3, y: 3, width: 7, height: 7 }),
    h('rect', { x: 14, y: 3, width: 7, height: 7 }),
    h('rect', { x: 14, y: 14, width: 7, height: 7 }),
    h('rect', { x: 3, y: 14, width: 7, height: 7 })
  ),
  chat: h('svg', { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('path', { d: 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z' })
  ),
  fusion: h('svg', { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('circle', { cx: 12, cy: 12, r: 3 }),
    h('path', { d: 'M12 5v14M5 12h14' })
  ),
  box: h('svg', { width: 22, height: 22, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('path', { d: 'M12 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7' }),
    h('path', { d: 'M18 3v5h-5' }),
    h('path', { d: 'M9 12l2 2 4-4' })
  ),
  brain: h('svg', { width: 22, height: 22, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('path', { d: 'M12 4a4 4 0 0 1 3.5 2.1A4 4 0 0 1 20 8a4 4 0 0 1-1.1 2.8A4 4 0 0 1 20 14a4 4 0 0 1-4.5 3.9A4 4 0 0 1 12 20a4 4 0 0 1-3.5-2.1A4 4 0 0 1 4 14a4 4 0 0 1 1.1-2.8A4 4 0 0 1 4 8a4 4 0 0 1 4.5-3.9A4 4 0 0 1 12 4z' })
  ),
  key: h('svg', { width: 22, height: 22, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('circle', { cx: 8, cy: 15, r: 4 }),
    h('path', { d: 'M10.85 12.15L19 4' }),
    h('path', { d: 'M18 5l2 2' }),
    h('path', { d: 'M15 8l2 2' })
  ),
  percent: h('svg', { width: 22, height: 22, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('line', { x1: 19, y1: 5, x2: 5, y2: 19 }),
    h('circle', { cx: 6.5, cy: 6.5, r: 2.5 }),
    h('circle', { cx: 17.5, cy: 17.5, r: 2.5 })
  ),
  bolt: h('svg', { width: 22, height: 22, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('path', { d: 'M13 2L3 14h9l-1 8 10-12h-9l1-8z' })
  ),
  send: h('svg', { width: 16, height: 16, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('line', { x1: 22, y1: 2, x2: 11, y2: 13 }),
    h('polygon', { points: '22 2 15 22 11 13 2 9 22 2' })
  ),
  shuffle: h('svg', { width: 16, height: 16, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('polyline', { points: '16 3 21 3 21 8' }),
    h('line', { x1: 4, y1: 20, x2: 21, y2: 3 }),
    h('polyline', { points: '21 16 21 21 16 21' }),
    h('line', { x1: 15, y1: 15, x2: 21, y2: 21 }),
    h('line', { x1: 4, y1: 4, x2: 9, y2: 9 })
  ),
  close: h('svg', { width: 14, height: 14, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('line', { x1: 18, y1: 6, x2: 6, y2: 18 }),
    h('line', { x1: 6, y1: 6, x2: 18, y2: 18 })
  ),
  check: h('svg', { width: 14, height: 14, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2 },
    h('polyline', { points: '20 6 9 17 4 12' })
  ),
};

// ---- API helper ----
async function api(url, opts) {
  const r = await fetch(API_BASE + url, opts);
  if (!r.ok) {
    const body = await r.text();
    let msg;
    try { msg = JSON.parse(body).error || body; } catch(e) { msg = body; }
    throw new Error(msg);
  }
  return r.json();
}

// ---- Simple hash router ----
function useHash() {
  const [hash, setHash] = useState(window.location.hash || '#/dashboard');
  useEffect(() => {
    const fn = () => setHash(window.location.hash || '#/dashboard');
    window.addEventListener('hashchange', fn);
    return () => window.removeEventListener('hashchange', fn);
  }, []);
  const navigate = useCallback(function(h) { window.location.hash = h; }, []);
  return [hash, navigate];
}

// ---- Stat Card component ----
function StatCard(props) {
  return h('div', { className: 'stat-card', style: props.color ? { borderTop: '3px solid ' + props.color } : {} },
    h('div', { className: 'stat-card-header' },
      h('span', { className: 'stat-card-label' }, props.label),
      h('div', { className: 'stat-icon', style: { background: props.bg || 'rgba(34,139,230,0.1)', color: props.color || '#4dabf7' } }, props.icon)
    ),
    h('div', { className: 'stat-card-value', style: { color: props.color || '#c9d1d9' } }, props.value != null ? props.value : '--'),
    props.sub ? h('div', { className: 'stat-card-sub' }, props.sub) : null
  );
}

// ---- Badge component ----
function Badge(props) {
  var cls = 'badge';
  if (props.dot) cls += ' badge-dot';
  if (props.color) cls += ' badge-' + props.color;
  return h('span', { className: cls, style: props.style }, props.children);
}

// ---- Status badge helper ----
function StatusBadge(status) {
  if (status === 'loading' || status === 'RUNNING') return Badge({ color: 'blue', dot: true }, 'RUNNING');
  if (status === 'done' || status === 'DONE') return Badge({ color: 'green' }, 'DONE');
  if (status === 'error' || status === 'ERROR') return Badge({ color: 'red' }, 'ERROR');
  if (status === 'CACHE') return Badge({ color: 'blue' }, 'CACHE');
  if (status === 'OK') return Badge({ color: 'green' }, 'OK');
  if (status === 'ERR') return Badge({ color: 'red' }, 'ERR');
  if (status === 'WAITING') return Badge({ color: 'gray' }, 'WAITING');
  return Badge({ color: 'gray' }, status || '--');
}

// ---- NavLink ----
function NavLink(props) {
  return h('a', {
    className: 'nav-link' + (props.active ? ' active' : ''),
    href: '#' + props.page,
    onClick: function(e) { e.preventDefault(); window.location.hash = '#' + props.page; }
  }, props.icon, h('span', null, props.label));
}

// ---- Multi-model selector (custom dropdown) ----
function ModelMultiSelect(props) {
  var _useState = useState(false), open = _useState[0], setOpen = _useState[1];
  var _useState2 = useState(''), search = _useState2[0], setSearch = _useState2[1];
  var ref = useRef(null);

  useEffect(function() {
    function handler(e) { if (ref.current && !ref.current.contains(e.target)) setOpen(false); }
    document.addEventListener('mousedown', handler);
    return function() { document.removeEventListener('mousedown', handler); };
  }, []);

  var filtered = search ? props.options.filter(function(o) {
    return o.label.toLowerCase().indexOf(search.toLowerCase()) >= 0;
  }) : props.options;

  function toggle(val) {
    var idx = props.value.indexOf(val);
    var next;
    if (idx >= 0) {
      next = props.value.slice(0, idx).concat(props.value.slice(idx + 1));
    } else {
      next = props.value.concat([val]);
    }
    props.onChange(next);
  }

  return h('div', { className: 'form-multi-select', ref: ref },
    h('div', {
      className: 'form-multi-select-inner',
      onClick: function() { setOpen(true); setSearch(''); }
    },
      props.value.map(function(v) {
        var label = props.options.filter(function(o) { return o.value === v; })[0];
        return h('span', { className: 'form-multi-select-tag', key: v },
          h('span', null, label ? label.label : v),
          h('span', { className: 'form-multi-select-tag-remove', onClick: function(e) { e.stopPropagation(); toggle(v); } }, Icons.close)
        );
      }),
      h('input', {
        className: 'form-multi-select-search',
        placeholder: props.value.length === 0 ? props.placeholder || 'Select...' : '',
        value: search,
        onChange: function(e) { setSearch(e.target.value); if (!open) setOpen(true); },
        onFocus: function() { setOpen(true); }
      })
    ),
    open ? h('div', { className: 'form-multi-select-dropdown' },
      filtered.length === 0 ? h('div', { className: 'form-multi-select-option', style: { color: '#5C5F66' } }, 'No options') :
      filtered.map(function(o) {
        var selected = props.value.indexOf(o.value) >= 0;
        return h('div', {
          key: o.value,
          className: 'form-multi-select-option' + (selected ? ' selected' : ''),
          onClick: function() { toggle(o.value); }
        }, selected ? Icons.check + ' ' : '', o.label);
      })
    ) : null
  );
}

// ================================================================
// PAGE: DASHBOARD
// ================================================================
function DashboardPage() {
  var _useState3 = useState(null), stats = _useState3[0], setStats = _useState3[1];
  var _useState4 = useState([]), logs = _useState4[0], setLogs = _useState4[1];
  var _useState5 = useState(''), time = _useState5[0], setTime = _useState5[1];

  var fetchData = useCallback(function() {
    Promise.all([api('/v1/stats'), api('/v1/logs')]).then(function(results) {
      setStats(results[0]);
      setLogs(results[1].entries || []);
      setTime(new Date().toLocaleTimeString());
    }).catch(console.error);
  }, []);

  useEffect(function() { fetchData(); var id = setInterval(fetchData, REFRESH_INTERVAL); return function() { clearInterval(id); }; }, [fetchData]);

  var cacheRate = useMemo(function() {
    if (!stats) return null;
    var hits = stats.cache_hits || 0, misses = stats.cache_misses || 0;
    return ((hits / Math.max(hits + misses, 1)) * 100).toFixed(1);
  }, [stats]);

  var providers = useMemo(function() {
    if (!stats) return [];
    var p = [];
    for (var k in stats) {
      if (k.indexOf('provider_') === 0) p.push({ name: k.slice(9), keys: stats[k].available_keys || 0, tier: stats[k].tier || 'free' });
    }
    return p;
  }, [stats]);

  var tiered = useMemo(function() {
    if (!stats) return [];
    var t = [];
    for (var k in stats) {
      if (k.indexOf('tiered_') === 0) t.push({ name: k.slice(7), keys: stats[k].available_keys || 0 });
    }
    return t;
  }, [stats]);

  return h('div', { style: { maxWidth: 1200, margin: '0 auto', width: '100%' } },
    h('div', { className: 'flex justify-between items-center mb-lg' },
      h('div', null,
        h('h2', { style: { margin: '0 0 4px', fontSize: 20, fontWeight: 600 } }, 'Dashboard'),
        h('div', { className: 'text-muted text-sm' }, 'Real-time gateway overview')
      ),
      h('div', { className: 'flex items-center gap-sm' },
        h('span', { className: 'badge badge-green badge-dot' }, 'Auto-refresh ' + REFRESH_INTERVAL/1000 + 's'),
        h('span', { className: 'text-xs text-muted' }, time)
      )
    ),

    // Stats cards
    h('div', { className: 'stats-grid' },
      h(StatCard, { icon: Icons.box, label: 'Providers', value: stats ? stats.providers : '--', color: '#4dabf7', bg: 'rgba(34,139,230,0.1)',
        sub: stats ? providers.filter(function(p) { return p.keys > 0; }).length + ' active' : null }),
      h(StatCard, { icon: Icons.brain, label: 'Models', value: stats ? stats.models : '--', color: '#7950f2', bg: 'rgba(121,80,242,0.1)' }),
      h(StatCard, { icon: Icons.key, label: 'Free Keys', value: stats ? stats.free_keys : '--', color: '#40c057', bg: 'rgba(64,192,87,0.1)' }),
      h(StatCard, { icon: Icons.percent, label: 'Cache Hit Rate', value: cacheRate ? cacheRate + '%' : '--', color: '#15aabf', bg: 'rgba(21,170,191,0.1)',
        sub: stats ? (stats.cache_hits || 0) + ' hits / ' + (stats.cache_misses || 0) + ' misses' : null })
    ),

    // Provider Health + Performance
    h('div', { className: 'grid-2 mb-lg' },
      // Provider Health
      h('div', { className: 'card' },
        h('h3', { style: { margin: '0 0 12px', fontSize: 14, fontWeight: 600 } }, 'Provider Health'),
        (providers.length + tiered.length) > 0 ? h('div', { className: 'table-wrap' },
          h('table', null,
            h('thead', null, h('tr', null, h('th', null, 'Provider'), h('th', null, 'Keys'), h('th', null, 'Status'))),
            h('tbody', null,
              providers.map(function(p) {
                return h('tr', { key: p.name },
                  h('td', null, h('strong', null, p.name)),
                  h('td', null, p.keys),
                  h('td', null, p.keys > 0 ? Badge({ color: 'green', dot: true }, 'ACTIVE') : Badge({ color: 'red', dot: true }, 'NO KEYS'))
                );
              }).concat(tiered.map(function(p) {
                return h('tr', { key: p.name },
                  h('td', null, h('strong', null, p.name)),
                  h('td', null, p.keys),
                  h('td', null, p.keys > 0 ? Badge({ color: 'green', dot: true }, 'ACTIVE') : Badge({ color: 'red', dot: true }, 'NO KEYS'))
                );
              }))
            )
          )
        ) : h('div', { className: 'text-muted text-sm' }, 'Loading...')
      ),
      // Performance
      h('div', { className: 'card' },
        h('h3', { style: { margin: '0 0 12px', fontSize: 14, fontWeight: 600 } }, 'Performance'),
        h('div', { className: 'flex flex-col', style: { gap: 8 } },
          h('div', { className: 'flex justify-between text-sm' }, h('span', { className: 'text-muted' }, 'Total Requests'), h('span', { style: { fontWeight: 600 } }, stats ? (stats.total_requests || 0).toLocaleString() : '--')),
          h('div', { style: { height: 1, background: '#373A40' } }),
          h('div', { className: 'flex justify-between text-sm' }, h('span', { className: 'text-muted' }, 'Avg Latency'), h('span', { style: { fontWeight: 600 } }, stats && stats.avg_latency_ms ? Number(stats.avg_latency_ms).toFixed(1) + 'ms' : '--')),
          h('div', { style: { height: 1, background: '#373A40' } }),
          h('div', { className: 'flex justify-between text-sm' }, h('span', { className: 'text-muted' }, 'Input Tokens'), h('span', { style: { fontWeight: 600 } }, stats ? (stats.tokens_input || 0).toLocaleString() : '--')),
          h('div', { style: { height: 1, background: '#373A40' } }),
          h('div', { className: 'flex justify-between text-sm' }, h('span', { className: 'text-muted' }, 'Output Tokens'), h('span', { style: { fontWeight: 600 } }, stats ? (stats.tokens_output || 0).toLocaleString() : '--'))
        )
      )
    ),

    // Recent Requests
    h('div', { className: 'card mb-lg' },
      h('h3', { style: { margin: '0 0 12px', fontSize: 14, fontWeight: 600 } }, 'Recent Requests'),
      h('div', { className: 'table-wrap', style: { maxHeight: 400, overflowY: 'auto' } },
        h('table', null,
          h('thead', null, h('tr', null, h('th', null, 'Time'), h('th', null, 'Model'), h('th', null, 'Provider'), h('th', null, 'Tokens'), h('th', null, 'Latency'), h('th', null, 'Status'))),
          h('tbody', null,
            logs.slice(0, 50).map(function(e, i) {
              return h('tr', { key: e.id || i },
                h('td', { className: 'text-xs' }, e.timestamp ? new Date(e.timestamp).toLocaleTimeString() : '--'),
                h('td', null, h('code', null, e.model || '--')),
                h('td', null, e.provider || '--'),
                h('td', null, e.total_tokens != null ? e.total_tokens : '--'),
                h('td', null, e.latency_ms ? e.latency_ms + 'ms' : '--'),
                h('td', null, e.cache_hit ? StatusBadge('CACHE') : e.success ? StatusBadge('OK') : StatusBadge('ERR'))
              );
            }),
            logs.length === 0 ? h('tr', null, h('td', { colSpan: 6, className: 'text-center text-muted p-md' }, 'No requests yet')) : null
          )
        )
      )
    )
  );
}

// ================================================================
// PAGE: CHAT
// ================================================================
function ChatPage() {
  var _useState6 = useState([]), models = _useState6[0], setModels = _useState6[1];
  var _useState7 = useState([]), selected = _useState7[0], setSelected = _useState7[1];
  var _useState8 = useState(''), prompt = _useState8[0], setPrompt = _useState8[1];
  var _useState9 = useState([]), results = _useState9[0], setResults = _useState9[1];
  var _useState10 = useState(false), loading = _useState10[0], setLoading = _useState10[1];

  useEffect(function() { api('/v1/models').then(function(d) { setModels(d.data || []); }).catch(console.error); }, []);

  var options = useMemo(function() {
    var seen = {};
    return models.filter(function(m) {
      var key = m.owned_by + '|' + m.id;
      if (seen[key]) return false;
      seen[key] = true;
      return true;
    }).map(function(m) {
      return { value: m.id, label: m.id + (m.owned_by && m.owned_by !== 'meta' ? ' [' + m.owned_by + ']' : '') };
    });
  }, [models]);

  function sendToAll() {
    if (!prompt.trim() || selected.length === 0) return;
    setLoading(true);
    var init = selected.map(function(m) { return { model: m, status: 'loading', content: '', latency: 0, tokens: 0, provider: '' }; });
    setResults(init);

    var copy = init.slice();
    Promise.all(selected.map(function(model, idx) {
      var start = performance.now();
      return api('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: model, messages: [{ role: 'user', content: prompt }], stream: false })
      }).then(function(resp) {
        copy[idx] = {
          model: model,
          status: 'done',
          content: (resp.choices && resp.choices[0] && resp.choices[0].message && resp.choices[0].message.content) || '(empty)',
          latency: Math.round(performance.now() - start),
          tokens: resp.usage ? resp.usage.total_tokens : 0,
          provider: resp.model || model
        };
        setResults(copy.slice());
      }).catch(function(err) {
        copy[idx] = { model: model, status: 'error', content: err.message, latency: Math.round(performance.now() - start), tokens: 0, provider: '' };
        setResults(copy.slice());
      });
    })).then(function() { setLoading(false); });
  }

  var gridClass = 'response-grid-' + (selected.length <= 2 ? selected.length : selected.length <= 3 ? 3 : 4);

  return h('div', { style: { maxWidth: 1200, margin: '0 auto', width: '100%' } },
    h('h2', { style: { margin: '0 0 4px', fontSize: 20, fontWeight: 600 } }, 'Multi-Model Chat'),
    h('p', { className: 'text-muted text-sm mb-lg' }, 'Send one prompt to multiple models simultaneously and compare responses.'),

    h('div', { className: 'card mb-lg' },
      h('div', { className: 'form-group' },
        h('label', { className: 'form-label' }, 'Select Models'),
        h(ModelMultiSelect, { options: options, value: selected, onChange: setSelected, placeholder: 'Choose models to query...' })
      ),
      h('div', { className: 'form-group' },
        h('label', { className: 'form-label' }, 'Prompt'),
        h('textarea', {
          className: 'form-textarea',
          placeholder: 'Enter your prompt here...',
          rows: 3,
          value: prompt,
          onChange: function(e) { setPrompt(e.target.value); }
        })
      ),
      h('div', { className: 'flex justify-between items-center' },
        h('span', { className: 'text-xs text-muted' }, selected.length + ' model(s) selected'),
        h('button', {
          className: 'btn btn-primary',
          disabled: loading || !prompt.trim() || selected.length === 0,
          onClick: sendToAll
        }, loading ? 'Sending...' : [Icons.send, ' Send to All'])
      )
    ),

    results.length > 0 ? h('div', { className: 'response-grid ' + gridClass },
      results.map(function(r, i) {
        var cls = 'response-card';
        if (r.status === 'loading') cls += ' loading';
        else if (r.status === 'done') cls += ' done';
        else if (r.status === 'error') cls += ' error';

        return h('div', { className: cls, key: i },
          h('div', { className: 'response-card-header' },
            h('div', { className: 'flex items-center gap-sm' },
              h('strong', { style: { fontSize: 13 } }, r.model),
              r.provider && r.provider !== r.model ? h('span', { className: 'badge badge-gray' }, r.provider) : null
            ),
            r.status === 'loading' ? StatusBadge('RUNNING') : r.status === 'done' ? StatusBadge('DONE') : StatusBadge('ERROR')
          ),
          h('div', { className: 'response-card-body' },
            r.status === 'loading' ? h('div', null,
              h('div', { className: 'skeleton mb-sm', style: { width: '100%' } }),
              h('div', { className: 'skeleton mb-sm', style: { width: '80%' } }),
              h('div', { className: 'skeleton', style: { width: '60%' } })
            ) :
            r.status === 'error' ? h('pre', { style: { color: '#fa5252' } }, r.content) :
            h('pre', null, r.content)
          ),
          h('div', { className: 'response-card-footer' },
            r.latency > 0 ? h('span', null, 'Latency: ' + r.latency + 'ms') : null,
            r.tokens > 0 ? h('span', null, 'Tokens: ' + r.tokens) : null
          )
        );
      })
    ) : null
  );
}

// ================================================================
// PAGE: FUSION (up to 8 models)
// ================================================================
function FusionPage() {
  var _useState11 = useState([]), models = _useState11[0], setModels = _useState11[1];
  var _useState12 = useState(4), count = _useState12[0], setCount = _useState12[1];
  var _useState13 = useState(''), prompt = _useState13[0], setPrompt = _useState13[1];
  var _useState14 = useState([]), panels = _useState14[0], setPanels = _useState14[1];
  var _useState15 = useState([]), results = _useState15[0], setResults = _useState15[1];
  var _useState16 = useState(false), loading = _useState16[0], setLoading = _useState16[1];
  var _useState17 = useState(true), autoSel = _useState17[0], setAutoSel = _useState17[1];

  useEffect(function() { api('/v1/models').then(function(d) { setModels(d.data || []); }).catch(console.error); }, []);

  var distinctModels = useMemo(function() {
    var seen = {};
    return models.filter(function(m) { var k = m.id; if (seen[k]) return false; seen[k] = true; return true; });
  }, [models]);

  var modelOptions = useMemo(function() {
    return distinctModels.map(function(m) {
      return { value: m.id, label: m.id + (m.owned_by && m.owned_by !== 'meta' ? ' [' + m.owned_by + ']' : '') };
    });
  }, [distinctModels]);

  useEffect(function() {
    var p = [];
    for (var i = 0; i < count; i++) {
      p.push({ index: i, model: autoSel && distinctModels[i] ? distinctModels[i].id : '' });
    }
    setPanels(p);
    setResults([]);
  }, [count, autoSel, distinctModels]);

  function updatePanel(idx, val) {
    var p = panels.slice();
    p[idx] = { index: idx, model: val };
    setPanels(p);
  }

  function compareAll() {
    var active = panels.filter(function(p) { return p.model; });
    if (!prompt.trim() || active.length === 0) { return; }
    setLoading(true);
    var init = active.map(function(p) { return { model: p.model, status: 'loading', content: '', latency: 0, tokens: 0, provider: '' }; });
    setResults(init);

    var copy = init.slice();
    Promise.all(active.map(function(panel, idx) {
      var start = performance.now();
      return api('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: panel.model, messages: [{ role: 'user', content: prompt }], stream: false })
      }).then(function(resp) {
        copy[idx] = {
          model: panel.model,
          status: 'done',
          content: (resp.choices && resp.choices[0] && resp.choices[0].message && resp.choices[0].message.content) || '(empty)',
          latency: Math.round(performance.now() - start),
          tokens: resp.usage ? resp.usage.total_tokens : 0,
          provider: resp.model || panel.model
        };
        setResults(copy.slice());
      }).catch(function(err) {
        copy[idx] = { model: panel.model, status: 'error', content: err.message, latency: Math.round(performance.now() - start), tokens: 0, provider: '' };
        setResults(copy.slice());
      });
    })).then(function() { setLoading(false); });
  }

  var gridCols = count <= 2 ? 'grid-2' : count <= 4 ? 'grid-2' : count <= 6 ? 'grid-3' : 'grid-4';

  var activeCount = panels.filter(function(p) { return p.model; }).length;
  var resGrid = activeCount <= 2 ? 'response-grid-2' : activeCount <= 4 ? 'response-grid-2' : activeCount <= 6 ? 'response-grid-3' : 'response-grid-4';

  return h('div', { style: { maxWidth: 1200, margin: '0 auto', width: '100%' } },
    h('div', { className: 'flex justify-between items-center mb-md' },
      h('div', null,
        h('h2', { style: { margin: '0 0 4px', fontSize: 20, fontWeight: 600 } }, 'Model Fusion'),
        h('p', { className: 'text-muted text-sm' }, 'Compare up to 8 models side by side.')
      ),
      h('span', { className: 'badge badge-yellow' }, 'BETA')
    ),

    h('div', { className: 'card mb-lg' },
      h('div', { className: 'flex items-center gap-md mb-md' },
        h('div', null,
          h('label', { className: 'form-label' }, 'Model Count'),
          h('select', {
            className: 'form-select',
            style: { width: 100 },
            value: count,
            onChange: function(e) { setCount(Math.min(Math.max(parseInt(e.target.value) || 2, 2), 8)); }
          }, [2,3,4,5,6,7,8].map(function(n) { return h('option', { key: n, value: n }, n + ' models'); }))
        ),
        h('div', { style: { marginTop: 20 } },
          h('label', { className: 'flex items-center gap-sm', style: { cursor: 'pointer' } },
            h('input', {
              type: 'checkbox',
              checked: autoSel,
              onChange: function(e) { setAutoSel(e.target.checked); },
              style: { accentColor: '#228be6' }
            }),
            h('span', { className: 'text-sm' }, 'Auto-select models')
          )
        )
      ),

      // Model selectors grid
      count > 0 ? h('div', { className: gridCols, style: { marginBottom: 16 } },
        panels.map(function(p, i) {
          return h('div', { key: i },
            h('label', { className: 'form-label' }, 'Model ' + (i + 1)),
            h('select', {
              className: 'form-select',
              value: p.model,
              onChange: function(e) { updatePanel(i, e.target.value); }
            },
              h('option', { value: '' }, '-- Select --'),
              modelOptions.map(function(o) {
                return h('option', { key: o.value, value: o.value }, o.label);
              })
            )
          );
        })
      ) : null,

      h('div', { className: 'form-group' },
        h('label', { className: 'form-label' }, 'Shared Prompt'),
        h('textarea', {
          className: 'form-textarea',
          placeholder: 'Enter the prompt to compare across models...',
          rows: 3,
          value: prompt,
          onChange: function(e) { setPrompt(e.target.value); }
        })
      ),

      h('div', { className: 'flex justify-between items-center' },
        h('span', { className: 'text-xs text-muted' }, activeCount + ' model(s) configured'),
        h('button', {
          className: 'btn btn-primary',
          disabled: loading || !prompt.trim() || activeCount === 0,
          onClick: compareAll,
          style: { padding: '10px 24px' }
        }, loading ? 'Comparing...' : [Icons.shuffle, ' Compare Models'])
      )
    ),

    results.length > 0 ? h('div', { className: 'response-grid ' + resGrid },
      results.map(function(r, i) {
        var cls = 'response-card';
        if (r.status === 'loading') cls += ' loading';
        else if (r.status === 'done') cls += ' done';
        else if (r.status === 'error') cls += ' error';

        return h('div', { className: cls, key: i },
          h('div', { className: 'response-card-header' },
            h('div', { className: 'flex items-center gap-sm' },
              h('strong', { style: { fontSize: 13 } }, r.model),
              r.provider && r.provider !== r.model ? h('span', { className: 'badge badge-gray' }, r.provider) : null
            ),
            r.status === 'loading' ? StatusBadge('RUNNING') : r.status === 'done' ? StatusBadge('DONE') : StatusBadge('ERROR')
          ),
          h('div', { className: 'response-card-body' },
            r.status === 'loading' ? h('div', null,
              h('div', { className: 'skeleton mb-sm', style: { width: '100%' } }),
              h('div', { className: 'skeleton mb-sm', style: { width: '75%' } }),
              h('div', { className: 'skeleton', style: { width: '50%' } })
            ) :
            r.status === 'error' ? h('pre', { style: { color: '#fa5252' } }, r.content) :
            h('pre', null, r.content)
          ),
          h('div', { className: 'response-card-footer' },
            r.latency > 0 ? h('span', null, 'Latency: ' + r.latency + 'ms') : null,
            r.tokens > 0 ? h('span', null, 'Tokens: ' + r.tokens) : null,
            r.status === 'done' && r.content ? h('span', null, r.content.split(' ').length + ' words') : null
          )
        );
      })
    ) : null
  );
}

// ================================================================
// APP SHELL
// ================================================================
function App() {
  var _useState18 = useHash(), hash = _useState18[0], navigate = _useState18[1];
  var page = hash.slice(1) || 'dashboard';

  var content;
  if (page === 'chat') content = h(ChatPage);
  else if (page === 'fusion') content = h(FusionPage);
  else content = h(DashboardPage);

  return h('div', { className: 'shell' },
    // Navbar
    h('div', { className: 'navbar' },
      h('div', { className: 'navbar-brand' },
        h('div', { className: 'navbar-brand-icon' }, 'SG'),
        h('span', { style: { fontWeight: 700, fontSize: 15 } }, 'Super Gateway')
      ),
      h('div', { className: 'navbar-links' },
        h(NavLink, { page: 'dashboard', label: 'Dashboard', icon: Icons.dashboard, active: page === 'dashboard' }),
        h(NavLink, { page: 'chat', label: 'Multi Chat', icon: Icons.chat, active: page === 'chat' }),
        h(NavLink, { page: 'fusion', label: 'Fusion', icon: Icons.fusion, active: page === 'fusion' })
      ),
      h('div', { className: 'navbar-footer' },
        h('div', { className: 'flex items-center gap-xs' },
          h('span', { className: 'badge badge-green badge-dot' }, 'Running'),
          h('span', { className: 'text-xs text-muted' }, 'v2.0.0')
        )
      )
    ),
    // Main content
    h('div', { className: 'main-content' }, content)
  );
}

// Mount
var root = ReactDOM.createRoot(document.getElementById('root'));
root.render(h(App));

})();
</script>
</body>
</html>`
