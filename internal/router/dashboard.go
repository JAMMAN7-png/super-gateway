package router

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Super AI Gateway</title>
<link rel="stylesheet" href="https://esm.sh/@mantine/core@7.10.0/styles.css">
<link rel="stylesheet" href="https://esm.sh/@mantine/notifications@7.10.0/styles.css">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
<script type="importmap">
{
  "imports": {
    "react": "https://esm.sh/react@18.3.1?bundle",
    "react-dom/client": "https://esm.sh/react-dom@18.3.1/client?bundle",
    "@mantine/core": "https://esm.sh/@mantine/core@7.10.0?bundle",
    "@mantine/hooks": "https://esm.sh/@mantine/hooks@7.10.0?bundle",
    "@mantine/notifications": "https://esm.sh/@mantine/notifications@7.10.0?bundle"
  }
}
</script>
<style>
#root { min-height: 100vh; }
body { margin: 0; }
.response-content { white-space: pre-wrap; word-break: break-word; font-family: 'Inter', monospace; font-size: 13px; line-height: 1.6; }
.stat-card { transition: transform 0.15s ease; }
.stat-card:hover { transform: translateY(-2px); }
@keyframes pulse-border { 0%,100% { border-color: #228be6; } 50% { border-color: #4dabf7; } }
.model-loading { animation: pulse-border 1.5s ease-in-out infinite; }
</style>
</head>
<body>
<div id="root"></div>
<script type="module">
import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { createRoot } from 'react-dom/client';
import {
  MantineProvider, createTheme, AppShell, Container, Group, Stack,
  SimpleGrid, Card, Text, Title, Badge, Table, Textarea, Button,
  MultiSelect, Select, NumberInput, Code, ScrollArea, Paper,
  NavLink, ThemeIcon, Progress, Divider, Tooltip, Alert, Loader,
  Skeleton, ActionIcon, TextInput, Modal, Tabs, Switch, Kbd,
  useMantineTheme, rgba, darken, lighten, Notification,
  rem, em, getThemeColor, defaultVariantColorsResolver
} from '@mantine/core';
import { Notifications, notifications } from '@mantine/notifications';

const theme = createTheme({
  fontFamily: 'Inter, sans-serif',
  defaultRadius: 'md',
  primaryColor: 'blue',
  colors: {
    dark: [
      '#C1C2C5', '#A6A7AB', '#909296', '#5C5F66',
      '#373A40', '#2C2E33', '#1A1B1E', '#0a0a14',
      '#141517', '#101113'
    ]
  },
  components: {
    Card: { defaultProps: { padding: 'lg', radius: 'md', withBorder: true } },
    Table: { defaultProps: { striped: true, highlightOnHover: true, fontSize: 'sm' } },
    Badge: { defaultProps: { size: 'sm' } },
    Button: { defaultProps: { size: 'sm' } },
  }
});

const API_BASE = '';
const REFRESH_INTERVAL = 5000;

// ---- Router ----
function useHash() {
  const [hash, setHash] = useState(() => window.location.hash || '#/dashboard');
  useEffect(() => {
    const onHashChange = () => setHash(window.location.hash || '#/dashboard');
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);
  const navigate = useCallback((h) => { window.location.hash = h; }, []);
  return [hash, navigate];
}

// ---- API helpers ----
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

// ---- Stat Card ----
function StatCard({ icon, label, value, color, sub }) {
  return (
    <Card className="stat-card" padding="lg">
      <Group justify="space-between" mb="xs">
        <Text size="xs" tt="uppercase" fw={600} c="dimmed">{label}</Text>
        {icon && <ThemeIcon variant="light" color={color || 'blue'} size="lg" radius="xl">{icon}</ThemeIcon>}
      </Group>
      <Text size="xxl" fw={700} c={color || undefined}>{value ?? '--'}</Text>
      {sub && <Text size="xs" c="dimmed" mt={4}>{sub}</Text>}
    </Card>
  );
}

// ---- Dashboard Page ----
function DashboardPage() {
  const [stats, setStats] = useState(null);
  const [logs, setLogs] = useState([]);
  const [time, setTime] = useState('');

  const fetchData = useCallback(async () => {
    try {
      const [s, l] = await Promise.all([api('/v1/stats'), api('/v1/logs')]);
      setStats(s);
      setLogs(l.entries || []);
      setTime(new Date().toLocaleTimeString());
    } catch(e) { console.error(e); }
  }, []);

  useEffect(() => { fetchData(); const id = setInterval(fetchData, REFRESH_INTERVAL); return () => clearInterval(id); }, [fetchData]);

  const cacheRate = stats ? ((stats.cache_hits || 0) / Math.max((stats.cache_hits || 0) + (stats.cache_misses || 0), 1) * 100).toFixed(1) : null;

  const providers = useMemo(() => {
    if (!stats) return [];
    const provs = [];
    for (const [k, v] of Object.entries(stats)) {
      if (k.startsWith('provider_')) provs.push({ name: k.slice(9), ...v });
    }
    return provs;
  }, [stats]);

  const tiered = useMemo(() => {
    if (!stats) return [];
    const t = [];
    for (const [k, v] of Object.entries(stats)) {
      if (k.startsWith('tiered_')) t.push({ name: k.slice(7), ...v });
    }
    return t;
  }, [stats]);

  const IconBox = ( <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18 3v5h-5"/><path d="M9 12l2 2 4-4"/></svg> );
  const IconBrain = ( <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 4a4 4 0 0 1 3.5 2.1A4 4 0 0 1 20 8a4 4 0 0 1-1.1 2.8A4 4 0 0 1 20 14a4 4 0 0 1-4.5 3.9A4 4 0 0 1 12 20a4 4 0 0 1-3.5-2.1A4 4 0 0 1 4 14a4 4 0 0 1 1.1-2.8A4 4 0 0 1 4 8a4 4 0 0 1 4.5-3.9A4 4 0 0 1 12 4z"/></svg> );
  const IconKey = ( <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="8" cy="15" r="4"/><path d="M10.85 12.15L19 4"/><path d="M18 5l2 2"/><path d="M15 8l2 2"/></svg> );
  const IconPercent = ( <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="19" y1="5" x2="5" y2="19"/><circle cx="6.5" cy="6.5" r="2.5"/><circle cx="17.5" cy="17.5" r="2.5"/></svg> );

  return (
    <Container size="xl" py="md">
      <Group justify="space-between" mb="lg">
        <div>
          <Title order={3}>Dashboard</Title>
          <Text size="sm" c="dimmed">Real-time gateway overview</Text>
        </div>
        <Group>
          <Badge variant="dot" color="green" size="lg">Auto-refresh {REFRESH_INTERVAL/1000}s</Badge>
          <Text size="xs" c="dimmed">{time}</Text>
        </Group>
      </Group>

      <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} mb="xl">
        <StatCard icon={IconBox} label="Providers" value={stats?.providers ?? '--'} color="blue"
          sub={stats ? providers.filter(p => p.available_keys > 0).length + ' active' : undefined} />
        <StatCard icon={IconBrain} label="Models" value={stats?.models ?? '--'} color="violet" />
        <StatCard icon={IconKey} label="Free Keys" value={stats?.free_keys ?? '--'} color="green" />
        <StatCard icon={IconPercent} label="Cache Hit Rate" value={cacheRate ? cacheRate + '%' : '--'} color="cyan"
          sub={stats ? (stats.cache_hits || 0) + ' hits / ' + (stats.cache_misses || 0) + ' misses' : undefined} />
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, lg: 2 }} mb="xl">
        <Card>
          <Title order={5} mb="md">Provider Health</Title>
          {(providers.length + tiered.length) > 0 ? (
            <ScrollArea h={300}>
              <Table>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Provider</Table.Th>
                    <Table.Th>Keys</Table.Th>
                    <Table.Th>Tier</Table.Th>
                    <Table.Th>Status</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {providers.map(p => (
                    <Table.Tr key={p.name}>
                      <Table.Td><Text fw={500}>{p.name}</Text></Table.Td>
                      <Table.Td>{p.available_keys ?? 0}</Table.Td>
                      <Table.Td><Badge color="green" variant="light">free</Badge></Table.Td>
                      <Table.Td>
                        <Badge color={(p.available_keys || 0) > 0 ? 'green' : 'red'} variant="dot">
                          {(p.available_keys || 0) > 0 ? 'ACTIVE' : 'NO KEYS'}
                        </Badge>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                  {tiered.map(p => (
                    <Table.Tr key={p.name}>
                      <Table.Td><Text fw={500}>{p.name}</Text></Table.Td>
                      <Table.Td>{p.available_keys ?? 0}</Table.Td>
                      <Table.Td><Badge color="yellow" variant="light">tiered</Badge></Table.Td>
                      <Table.Td>
                        <Badge color={(p.available_keys || 0) > 0 ? 'green' : 'red'} variant="dot">
                          {(p.available_keys || 0) > 0 ? 'ACTIVE' : 'NO KEYS'}
                        </Badge>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </ScrollArea>
          ) : <Text c="dimmed" size="sm">Loading...</Text>}
        </Card>

        <Card>
          <Title order={5} mb="md">Performance</Title>
          <Stack gap="sm">
            <Group justify="space-between"><Text size="sm">Total Requests</Text><Text fw={600}>{stats?.total_requests ?? '--'}</Text></Group>
            <Divider />
            <Group justify="space-between"><Text size="sm">Avg Latency</Text><Text fw={600}>{stats?.avg_latency_ms ? stats.avg_latency_ms.toFixed(1) + 'ms' : '--'}</Text></Group>
            <Divider />
            <Group justify="space-between"><Text size="sm">Input Tokens</Text><Text fw={600}>{(stats?.tokens_input ?? 0).toLocaleString()}</Text></Group>
            <Divider />
            <Group justify="space-between"><Text size="sm">Output Tokens</Text><Text fw={600}>{(stats?.tokens_output ?? 0).toLocaleString()}</Text></Group>
          </Stack>
        </Card>
      </SimpleGrid>

      <Card>
        <Title order={5} mb="md">Recent Requests</Title>
        <ScrollArea h={400}>
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Time</Table.Th>
                <Table.Th>Model</Table.Th>
                <Table.Th>Provider</Table.Th>
                <Table.Th>Tokens</Table.Th>
                <Table.Th>Latency</Table.Th>
                <Table.Th>Status</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {logs.slice(0, 50).map((e, i) => (
                <Table.Tr key={e.id || i}>
                  <Table.Td><Text size="xs">{e.timestamp ? new Date(e.timestamp).toLocaleTimeString() : '--'}</Text></Table.Td>
                  <Table.Td><Code>{e.model || '--'}</Code></Table.Td>
                  <Table.Td>{e.provider || '--'}</Table.Td>
                  <Table.Td>{e.total_tokens ?? '--'}</Table.Td>
                  <Table.Td>{e.latency_ms ? e.latency_ms + 'ms' : '--'}</Table.Td>
                  <Table.Td>
                    {e.cache_hit ? <Badge color="blue">CACHE</Badge> :
                     e.success ? <Badge color="green">OK</Badge> :
                     <Badge color="red">ERR</Badge>}
                  </Table.Td>
                </Table.Tr>
              ))}
              {logs.length === 0 && (
                <Table.Tr><Table.Td colSpan={6}><Text c="dimmed" ta="center" py="xl">No requests yet</Text></Table.Td></Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </ScrollArea>
      </Card>
    </Container>
  );
}

// ---- Chat Page ----
function ChatPage() {
  const [models, setModels] = useState([]);
  const [selectedModels, setSelectedModels] = useState([]);
  const [prompt, setPrompt] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => { api('/v1/models').then(d => setModels(d.data || [])).catch(console.error); }, []);

  const modelOptions = useMemo(() => models.map(m => ({
    value: m.id,
    label: m.id + (m.owned_by && m.owned_by !== 'meta' ? ' (' + m.owned_by + ')' : ''),
    group: m.owned_by || 'other'
  })), [models]);

  const sendToAll = useCallback(async () => {
    if (!prompt.trim() || selectedModels.length === 0) return;
    setLoading(true);
    const newResults = selectedModels.map(m => ({ model: m, status: 'loading', content: '', latency: 0, tokens: 0 }));
    setResults(newResults);

    const resultsCopy = [...newResults];
    await Promise.all(selectedModels.map(async (model, idx) => {
      const start = performance.now();
      try {
        const resp = await api('/v1/chat/completions', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ model, messages: [{ role: 'user', content: prompt }], stream: false })
        });
        resultsCopy[idx] = {
          model,
          status: 'done',
          content: resp.choices?.[0]?.message?.content || '(empty response)',
          latency: Math.round(performance.now() - start),
          tokens: resp.usage?.total_tokens || 0,
          provider: resp.model || model
        };
      } catch (err) {
        resultsCopy[idx] = { model, status: 'error', content: err.message, latency: Math.round(performance.now() - start), tokens: 0 };
      }
      setResults([...resultsCopy]);
    }));
    setLoading(false);
  }, [prompt, selectedModels]);

  return (
    <Container size="xl" py="md">
      <Title order={3} mb="lg">Multi-Model Chat</Title>
      <Text size="sm" c="dimmed" mb="lg">Send one prompt to multiple models simultaneously and compare responses.</Text>

      <Card mb="lg">
        <Stack gap="md">
          <MultiSelect
            label="Select Models"
            placeholder="Choose models to query..."
            data={modelOptions}
            value={selectedModels}
            onChange={setSelectedModels}
            searchable
            clearable
            nothingFoundMessage="No models found"
          />
          <Textarea
            label="Prompt"
            placeholder="Enter your prompt here..."
            minRows={3}
            maxRows={8}
            value={prompt}
            onChange={(e) => setPrompt(e.currentTarget.value)}
          />
          <Group justify="space-between">
            <Text size="xs" c="dimmed">{selectedModels.length} model(s) selected</Text>
            <Button
              onClick={sendToAll}
              loading={loading}
              disabled={!prompt.trim() || selectedModels.length === 0}
            >
              Send to All
            </Button>
          </Group>
        </Stack>
      </Card>

      {results.length > 0 && (
        <SimpleGrid cols={{ base: 1, sm: 2, lg: results.length <= 2 ? 2 : 3 }} spacing="md">
          {results.map((r, i) => (
            <Card key={i} padding="md" className={r.status === 'loading' ? 'model-loading' : ''}
              style={r.status === 'done' ? { borderLeft: '3px solid var(--mantine-color-green-6)' } :
                     r.status === 'error' ? { borderLeft: '3px solid var(--mantine-color-red-6)' } : {}}>
              <Group justify="space-between" mb="sm">
                <Group gap="xs">
                  <Text fw={600} size="sm">{r.model}</Text>
                  {r.provider && r.provider !== r.model && <Badge variant="light" color="gray">{r.provider}</Badge>}
                </Group>
                {r.status === 'loading' ? <Badge color="blue" variant="dot">RUNNING</Badge> :
                 r.status === 'done' ? <Badge color="green">DONE</Badge> :
                 <Badge color="red">ERROR</Badge>}
              </Group>
              <ScrollArea h={300} mb="sm">
                <Text className="response-content" size="sm">
                  {r.status === 'loading' ? <Skeleton height={200} /> :
                   r.status === 'error' ? <Text c="red">{r.content}</Text> :
                   r.content}
                </Text>
              </ScrollArea>
              <Divider mb="xs" />
              <Group gap="md">
                {r.latency > 0 && <Text size="xs" c="dimmed">Latency: {r.latency}ms</Text>}
                {r.tokens > 0 && <Text size="xs" c="dimmed">Tokens: {r.tokens}</Text>}
              </Group>
            </Card>
          ))}
        </SimpleGrid>
      )}
    </Container>
  );
}

// ---- Fusion Page (up to 8 models) ----
function FusionPage() {
  const [models, setModels] = useState([]);
  const [modelCount, setModelCount] = useState(4);
  const [prompt, setPrompt] = useState('');
  const [panels, setPanels] = useState([]);
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [autoSelect, setAutoSelect] = useState(true);

  useEffect(() => { api('/v1/models').then(d => setModels(d.data || [])).catch(console.error); }, []);

  const distinctModels = useMemo(() => {
    const seen = new Set();
    return models.filter(m => { const dup = seen.has(m.id); seen.add(m.id); return !dup; });
  }, [models]);

  const modelOptions = useMemo(() => distinctModels.map(m => ({
    value: m.id,
    label: m.id + (m.owned_by && m.owned_by !== 'meta' ? ' (' + m.owned_by + ')' : ''),
  })), [distinctModels]);

  useEffect(() => {
    const newPanels = [];
    for (let i = 0; i < modelCount; i++) {
      newPanels.push({ index: i, model: autoSelect && distinctModels[i] ? distinctModels[i].id : '' });
    }
    setPanels(newPanels);
    setResults([]);
  }, [modelCount, autoSelect, distinctModels]);

  const updatePanelModel = (idx, val) => {
    const p = [...panels]; p[idx] = { ...p[idx], model: val }; setPanels(p);
  };

  const compare = useCallback(async () => {
    if (!prompt.trim() || panels.every(p => !p.model)) { notifications.show({ message: 'Select at least one model', color: 'red' }); return; }
    setLoading(true);
    const newResults = panels.filter(p => p.model).map(p => ({ ...p, status: 'loading', content: '', latency: 0, tokens: 0 }));
    setResults(newResults);

    const resultsCopy = [...newResults];
    const activePanels = panels.filter(p => p.model);

    await Promise.all(activePanels.map(async (panel, idx) => {
      const start = performance.now();
      try {
        const resp = await api('/v1/chat/completions', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ model: panel.model, messages: [{ role: 'user', content: prompt }], stream: false })
        });
        resultsCopy[idx] = {
          ...panel,
          status: 'done',
          content: resp.choices?.[0]?.message?.content || '(empty response)',
          latency: Math.round(performance.now() - start),
          tokens: resp.usage?.total_tokens || 0,
          provider: resp.model || panel.model
        };
      } catch (err) {
        resultsCopy[idx] = { ...panel, status: 'error', content: err.message, latency: Math.round(performance.now() - start), tokens: 0 };
      }
      setResults([...resultsCopy]);
    }));
    setLoading(false);
  }, [prompt, panels]);

  const gridCols = { base: 1, sm: 2, lg: modelCount <= 2 ? modelCount : modelCount <= 4 ? 2 : modelCount <= 6 ? 3 : 4 };

  return (
    <Container size="xl" py="md">
      <Group justify="space-between" mb="lg">
        <div>
          <Title order={3}>Model Fusion</Title>
          <Text size="sm" c="dimmed">Compare up to 8 models side by side. Like OpenRouter.ai/fusion but with up to 8 models.</Text>
        </div>
        <Badge variant="light" color="yellow" size="lg">BETA</Badge>
      </Group>

      <Card mb="lg">
        <Stack gap="md">
          <Group>
            <NumberInput
              label="Number of Models"
              value={modelCount}
              onChange={(v) => setModelCount(Math.min(Math.max(parseInt(v) || 2, 2), 8))}
              min={2}
              max={8}
              style={{ width: 180 }}
            />
            <Switch
              label="Auto-select models"
              checked={autoSelect}
              onChange={(e) => setAutoSelect(e.currentTarget.checked)}
              mt={24}
            />
          </Group>

          <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="xs">
            {panels.map((p, i) => (
              <Select
                key={i}
                label={'Model ' + (i + 1)}
                placeholder="Select model..."
                data={modelOptions}
                value={p.model}
                onChange={(v) => updatePanelModel(i, v || '')}
                searchable
                clearable
                nothingFoundMessage="No model found"
              />
            ))}
          </SimpleGrid>

          <Textarea
            label="Shared Prompt"
            placeholder="Enter the prompt to compare across models..."
            minRows={3}
            maxRows={6}
            value={prompt}
            onChange={(e) => setPrompt(e.currentTarget.value)}
          />

          <Group justify="space-between">
            <Text size="xs" c="dimmed">{panels.filter(p => p.model).length} model(s) configured</Text>
            <Button
              size="md"
              onClick={compare}
              loading={loading}
              disabled={!prompt.trim() || panels.every(p => !p.model)}
            >
              Compare Models
            </Button>
          </Group>
        </Stack>
      </Card>

      {results.length > 0 && (
        <SimpleGrid cols={gridCols} spacing="md">
          {results.map((r, i) => (
            <Card key={i} padding="md"
              className={r.status === 'loading' ? 'model-loading' : ''}
              style={r.status === 'done' ? { borderLeft: '3px solid var(--mantine-color-green-6)' } :
                     r.status === 'error' ? { borderLeft: '3px solid var(--mantine-color-red-6)' } :
                     { borderLeft: '3px solid var(--mantine-color-blue-6)' }}>
              <Group justify="space-between" mb="sm">
                <Group gap="xs">
                  <Text fw={600} size="sm">{r.model}</Text>
                  {r.provider && <Badge variant="light" color="gray">{r.provider}</Badge>}
                </Group>
                {r.status === 'loading' ? <Badge color="blue" variant="dot">RUNNING</Badge> :
                 r.status === 'done' ? <Badge color="green">DONE</Badge> :
                 <Badge color="red">ERROR</Badge>}
              </Group>
              <ScrollArea h={350} mb="sm">
                <Text className="response-content" size="sm">
                  {r.status === 'loading' ? <Stack><Skeleton height={30} /><Skeleton height={100} /><Skeleton height={60} /></Stack> :
                   r.status === 'error' ? <Text c="red">{r.content}</Text> :
                   r.content}
                </Text>
              </ScrollArea>
              <Divider mb="xs" />
              <Group gap="md">
                {r.latency > 0 && <Text size="xs" c="dimmed"><Kbd>{r.latency}ms</Kbd></Text>}
                {r.tokens > 0 && <Text size="xs" c="dimmed">{r.tokens} tokens</Text>}
                {r.status === 'done' && r.content && <Text size="xs" c="dimmed">{r.content.split(' ').length} words</Text>}
              </Group>
            </Card>
          ))}
        </SimpleGrid>
      )}
    </Container>
  );
}

// ---- App Shell ----
function App() {
  const [hash, navigate] = useHash();
  const [opened, { toggle }] = React.useState(false);

  const page = hash.slice(1) || 'dashboard';
  const setPage = (p) => navigate('#' + p);

  const IconDashboard = ( <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg> );
  const IconChat = ( <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg> );
  const IconFusion = ( <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 3l4 8-4 8-4-8z"/><path d="M4 12h16"/></svg> );

  return (
    <AppShell
      navbar={{ width: 220, breakpoint: 'sm', collapsed: { mobile: false } }}
      padding={0}
    >
      <AppShell.Navbar p="xs" style={{ background: 'var(--mantine-color-dark-8)' }}>
        <AppShell.Section>
          <Group p="sm" gap="xs">
            <ThemeIcon variant="gradient" gradient={{ from: 'blue', to: 'cyan' }} size="lg" radius="md">
              <Text fw={700} size="lg" c="white">SG</Text>
            </ThemeIcon>
            <div>
              <Text fw={700} size="sm">Super Gateway</Text>
              <Text size="xs" c="dimmed">v2.0.0</Text>
            </div>
          </Group>
          <Divider mb="xs" />
        </AppShell.Section>

        <AppShell.Section grow>
          <NavLink
            label="Dashboard"
            leftSection={IconDashboard}
            active={page === 'dashboard'}
            onClick={() => setPage('dashboard')}
            variant="filled"
            color="blue"
          />
          <NavLink
            label="Multi Chat"
            leftSection={IconChat}
            active={page === 'chat'}
            onClick={() => setPage('chat')}
            variant="filled"
            color="blue"
          />
          <NavLink
            label="Fusion"
            leftSection={IconFusion}
            active={page === 'fusion'}
            onClick={() => setPage('fusion')}
            variant="filled"
            color="blue"
          />
        </AppShell.Section>

        <AppShell.Section>
          <Divider mb="xs" />
          <Group p="sm" gap={4}>
            <Badge color="green" variant="dot" size="sm">Running</Badge>
            <Text size="xs" c="dimmed">Coolify</Text>
          </Group>
        </AppShell.Section>
      </AppShell.Navbar>

      <AppShell.Main style={{ background: 'var(--mantine-color-dark-9)', minHeight: '100vh' }}>
        {page === 'dashboard' && <DashboardPage />}
        {page === 'chat' && <ChatPage />}
        {page === 'fusion' && <FusionPage />}
      </AppShell.Main>
    </AppShell>
  );
}

const root = createRoot(document.getElementById('root'));
root.render(
  <MantineProvider theme={theme} defaultColorScheme="dark">
    <Notifications />
    <App />
  </MantineProvider>
);
</script>
</body>
</html>`
