// service-log-panel 提供服务日志文件选择、tail 快照和 SSE 实时流。
import { createLogStreamPanel } from "./log-stream-panel.js";

export function createServiceLogPanel(container, { api, statusBar } = {}) {
  container.innerHTML = `
    <div class="row" style="margin-bottom:8px; justify-content:space-between">
      <div class="row" style="flex:1">
        <select id="ops-svc-log-select" style="min-width:320px"></select>
        <input id="ops-svc-log-lines" type="number" min="10" max="5000" value="200" style="width:110px" />
        <button id="ops-svc-log-tail" class="secondary">Tail</button>
        <button id="ops-svc-log-stream" class="secondary">实时流</button>
        <button id="ops-svc-log-stop" class="secondary">停止流</button>
      </div>
      <div class="row">
        <button id="ops-svc-log-refresh" class="secondary">刷新列表</button>
      </div>
    </div>
    <div class="muted" id="ops-svc-log-hint" style="margin-bottom:8px"></div>
    <div id="ops-svc-log-view"></div>
  `;

  const select = container.querySelector("#ops-svc-log-select");
  const linesInput = container.querySelector("#ops-svc-log-lines");
  const tailBtn = container.querySelector("#ops-svc-log-tail");
  const streamBtn = container.querySelector("#ops-svc-log-stream");
  const stopBtn = container.querySelector("#ops-svc-log-stop");
  const refreshBtn = container.querySelector("#ops-svc-log-refresh");
  const hint = container.querySelector("#ops-svc-log-hint");
  const view = container.querySelector("#ops-svc-log-view");

  const logPanel = createLogStreamPanel(view);

  let files = [];

  function getSelectedFileId() {
    return (select.value || "").trim();
  }

  function setHint(text) {
    hint.textContent = text || "";
  }

  function renderOptions() {
    const current = getSelectedFileId();
    select.innerHTML = files
      .map((f) => {
        const label = `${f.rel_path || f.name || f.id}`;
        const selected = f.id === current ? "selected" : "";
        return `<option value="${String(f.id)}" ${selected}>${escapeHTML(label)}</option>`;
      })
      .join("");
  }

  function escapeHTML(str) {
    return String(str || "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#039;");
  }

  async function refreshFiles() {
    if (!api) return;
    try {
      setHint("加载文件列表中...");
      const list = await api.listServiceLogFiles();
      files = Array.isArray(list) ? list : [];
      renderOptions();
      setHint(files.length ? `共 ${files.length} 个日志文件` : "未找到日志文件");
    } catch (err) {
      setHint("");
      if (statusBar) statusBar.showError(err.message);
    }
  }

  async function tailOnce() {
    const fileId = getSelectedFileId();
    if (!fileId) {
      if (statusBar) statusBar.showError("请选择日志文件");
      return;
    }
    const lines = Math.min(5000, Math.max(10, Number(linesInput.value || 200)));
    try {
      const data = await api.getServiceLogTail(fileId, lines);
      const meta = [];
      meta.push(`文件：${data.rel_path || fileId}`);
      if (data.truncated) meta.push("尾部快照已截断(1MB)");
      setHint(meta.join(" | "));
      logPanel.stopStream();
      logPanel.setText(data.text || "");
    } catch (err) {
      if (statusBar) statusBar.showError(err.message);
    }
  }

  function startStream() {
    const fileId = getSelectedFileId();
    if (!fileId) {
      if (statusBar) statusBar.showError("请选择日志文件");
      return;
    }
    setHint(`实时追踪：${fileId}`);
    logPanel.setText("");
    logPanel.startStream(api.buildServiceLogStreamUrl(fileId, { fromEnd: true }), {
      onError: () => {
        if (statusBar) statusBar.showError("服务日志流已断开，请重试。");
      },
    });
  }

  function stopStream() {
    logPanel.stopStream();
    setHint("已停止实时流");
  }

  if (refreshBtn) refreshBtn.addEventListener("click", () => refreshFiles());
  if (tailBtn) tailBtn.addEventListener("click", () => tailOnce());
  if (streamBtn) streamBtn.addEventListener("click", () => startStream());
  if (stopBtn) stopBtn.addEventListener("click", () => stopStream());
  if (select) {
    select.addEventListener("change", () => {
      stopStream();
      logPanel.setText("");
      setHint(select.value ? `已选择：${select.value}` : "");
    });
  }

  return {
    refreshFiles,
    stopStream,
  };
}

