const state = {
  token: null,
  graph: null,
};

async function ensureToken() {
  if (state.token) {
    return;
  }
  const stored = localStorage.getItem("niac-token");
  if (stored) {
    state.token = stored;
    return;
  }
  const token = window.prompt("Enter NIAC API token (leave empty if not set):", "");
  if (token !== null) {
    state.token = token.trim();
    localStorage.setItem("niac-token", state.token);
  }
}

function authHeaders() {
  const headers = { "Content-Type": "application/json" };
  if (state.token) {
    headers["Authorization"] = `Bearer ${state.token}`;
  }
  return headers;
}

async function fetchJSON(path) {
  await ensureToken();
  const response = await fetch(path, {
    headers: authHeaders(),
  });
  if (response.status === 401) {
    localStorage.removeItem("niac-token");
    state.token = null;
    throw new Error("Unauthorized");
  }
  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }
  return response.json();
}

function updateStats(stats) {
  document.getElementById("stat-interface").innerText = stats.interface || "-";
  document.getElementById("stat-tx").innerText = stats.stack.packets_sent;
  document.getElementById("stat-rx").innerText = stats.stack.packets_received;
  document.getElementById("stat-errors").innerText = stats.stack.errors;
  document.getElementById("stat-devices").innerText = stats.device_count;
  document.getElementById("version-info").innerText = `NIAC ${stats.version}`;
  document.getElementById("api-status").innerText = "online";
  document.getElementById("api-status").classList.add("online");
}

function updateDeviceTable(devices) {
  const tbody = document.querySelector("#device-table tbody");
  tbody.innerHTML = "";
  devices.forEach((device) => {
    const row = document.createElement("tr");
    row.innerHTML = `
      <td>${device.name}</td>
      <td>${device.type}</td>
      <td>${device.ips.join(", ")}</td>
      <td>${device.protocols.join(", ")}</td>
    `;
    tbody.appendChild(row);
  });
}

function updateHistoryTable(history) {
  const tbody = document.querySelector("#history-table tbody");
  tbody.innerHTML = "";
  history.forEach((run) => {
    const row = document.createElement("tr");
    const totalPackets = run.packets_sent + run.packets_received;
    row.innerHTML = `
      <td>${new Date(run.started_at).toLocaleString()}</td>
      <td>${Math.round(run.duration / 1e9)}s</td>
      <td>${run.interface}</td>
      <td>${run.device_count}</td>
      <td>${totalPackets}</td>
    `;
    tbody.appendChild(row);
  });
}

function initGraph(container) {
  state.graph = ForceGraph()(container)
    .width(container.clientWidth)
    .height(400)
    .nodeLabel("name")
    .nodeAutoColorBy("type")
    .linkDirectionalArrowLength(4)
    .linkDirectionalArrowRelPos(1);
}

function updateTopology(topology) {
  if (!state.graph) {
    initGraph(document.getElementById("topology-container"));
  }
  state.graph.graphData(topology);
}

async function refresh() {
  try {
    const [stats, devices, history, topology] = await Promise.all([
      fetchJSON("/api/v1/stats"),
      fetchJSON("/api/v1/devices"),
      fetchJSON("/api/v1/history"),
      fetchJSON("/api/v1/topology"),
    ]);
    updateStats(stats);
    updateDeviceTable(devices);
    updateHistoryTable(history);
    updateTopology(topology);
  } catch (err) {
    console.error(err);
    document.getElementById("api-status").innerText = "offline";
    document.getElementById("api-status").classList.remove("online");
  }
}

document.getElementById("token-reset").addEventListener("click", () => {
  localStorage.removeItem("niac-token");
  state.token = null;
  ensureToken();
});

ensureToken().then(() => {
  refresh();
  setInterval(refresh, 5000);
});
