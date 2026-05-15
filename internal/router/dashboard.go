package router

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Super AI Gateway — Dashboard</title>
<style>
:root{--bg:#0a0a0f;--panel:#111118;--border:#1e1e2e;--text:#c9d1d9;--green:#3fb950;--red:#f85149;--blue:#58a6ff;--yellow:#d2991d;--muted:#6e7681}
*{box-sizing:border-box;margin:0;padding:0}
body{background:var(--bg);color:var(--text);font-family:ui-monospace,SFMono-Regular,monospace;padding:20px;min-height:100vh}
h1{font-size:20px;margin-bottom:4px}h1 span{color:var(--green)}
.subtitle{color:var(--muted);font-size:12px;margin-bottom:24px}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:16px;margin-bottom:20px}
.card{background:var(--panel);border:1px solid var(--border);border-radius:8px;padding:16px}
.card h3{font-size:13px;color:var(--muted);text-transform:uppercase;letter-spacing:0.5px;margin-bottom:8px}
.card .value{font-size:28px;font-weight:700}
.value.green{color:var(--green)}.value.red{color:var(--red)}.value.blue{color:var(--blue)}
table{width:100%;border-collapse:collapse;margin-top:12px}
th,td{padding:8px 12px;text-align:left;font-size:12px;border-bottom:1px solid var(--border)}
th{color:var(--muted);font-weight:600}
tr:hover{background:rgba(88,166,255,0.04)}
.badge{padding:2px 8px;border-radius:10px;font-size:10px;font-weight:600}
.badge-ok{background:rgba(63,185,80,0.15);color:var(--green)}
.badge-err{background:rgba(248,81,73,0.15);color:var(--red)}
.badge-cache{background:rgba(88,166,255,0.15);color:var(--blue)}
.progress-bar{height:4px;background:var(--border);border-radius:2px;margin-top:8px;overflow:hidden}
.progress-fill{height:100%;background:var(--green);transition:width 0.3s}
.auto-refresh{color:var(--muted);font-size:11px;margin-top:16px}
</style>
</head>
<body>
<h1>⚡ Super <span>AI</span> Gateway</h1>
<div class="subtitle">Real-time dashboard · Auto-refresh 5s</div>

<div class="grid" id="stats-grid">
  <div class="card"><h3>Providers</h3><div class="value blue" id="stat-providers">—</div></div>
  <div class="card"><h3>Models</h3><div class="value" id="stat-models">—</div></div>
  <div class="card"><h3>Free Keys</h3><div class="value green" id="stat-keys">—</div></div>
  <div class="card"><h3>Cache Hit Rate</h3><div class="value" id="stat-cache">—</div></div>
</div>

<div class="card" style="margin-bottom:20px">
  <h3>Recent Requests</h3>
  <div id="logs-container" style="max-height:400px;overflow-y:auto">
    <table><thead><tr><th>Time</th><th>Model</th><th>Provider</th><th>Tokens</th><th>Latency</th><th>Status</th></tr></thead>
    <tbody id="logs-body"></tbody></table>
  </div>
</div>

<div class="card">
  <h3>Provider Health</h3>
  <div id="providers-container"><span style="color:var(--muted)">Loading...</span></div>
</div>

<div class="auto-refresh">Auto-refreshing every 5s · <span id="last-update"></span></div>

<script>
async function refresh(){
  try{
    const[s,r]=await Promise.all([
      fetch('/v1/stats').then(r=>r.json()),
      fetch('/v1/logs').then(r=>r.json())
    ]);
    document.getElementById('stat-providers').textContent=s.providers||0;
    document.getElementById('stat-models').textContent=s.models||0;
    document.getElementById('stat-keys').textContent=s.free_keys||0;
    const total=s.cache_hits+s.cache_misses||1;
    document.getElementById('stat-cache').textContent=Math.round((s.cache_hits/total)*100)+'%';
    document.getElementById('stat-cache').className='value '+(s.cache_hits/total>0.5?'green':'blue');

    // Render logs
    const tb=document.getElementById('logs-body');
    tb.innerHTML=(r.entries||[]).slice(0,50).map(e=>'<tr>'+
      '<td>'+new Date(e.timestamp).toLocaleTimeString()+'</td>'+
      '<td><code>'+e.model+'</code></td>'+
      '<td>'+e.provider+'</td>'+
      '<td>'+e.total_tokens+'</td>'+
      '<td>'+(e.latency_ms||'—')+'ms</td>'+
      '<td><span class="badge '+(e.cache_hit?'badge-cache':e.success?'badge-ok':'badge-err')+'">'+
      (e.cache_hit?'CACHE':e.success?'OK':'ERR')+'</span></td>'+
      '</tr>').join('');

    // Render provider health
    const pc=document.getElementById('providers-container');
    let html='<table><thead><tr><th>Provider</th><th>Keys</th><th>Status</th></tr></thead><tbody>';
    for(const[k,v] of Object.entries(s)){
      if(k.startsWith('provider_')){
        const name=k.replace('provider_','');
        html+='<tr><td>'+name+'</td><td>'+v.available_keys+'</td>'+
          '<td><span class="badge '+(v.available_keys>0?'badge-ok':'badge-err')+'">'+
          (v.available_keys>0?'ACTIVE':'NO KEYS')+'</span></td></tr>';
      }
    }
    html+='</tbody></table>';
    pc.innerHTML=html;

    document.getElementById('last-update').textContent=new Date().toLocaleTimeString();
  }catch(e){console.error(e)}
}
refresh();setInterval(refresh,5000);
</script>
</body>
</html>`
