// status-panel 展示后端/容器/代理的状态快照（只读）。
export function createStatusPanel(container, { onRefresh } = {}) {
  container.innerHTML = `
    <div class="row" style="justify-content:space-between; margin-bottom:8px">
      <div class="muted">用于快速判断“服务是否启动、代理是否可达、容器是否健康”。</div>
      <div class="row">
        <button id="ops-status-refresh" class="secondary">刷新</button>
      </div>
    </div>
    <div id="ops-status-content" class="muted">尚未加载</div>
  `;

  const refreshBtn = container.querySelector("#ops-status-refresh");
  const content = container.querySelector("#ops-status-content");

  if (refreshBtn && onRefresh) {
    refreshBtn.addEventListener("click", () => onRefresh());
  }

  function escapeHTML(str) {
    return String(str || "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#039;");
  }

  function renderTable(headers, rows) {
    const headHtml = headers.map((h) => `<th>${escapeHTML(h)}</th>`).join("");
    const bodyHtml = rows
      .map((cells) => `<tr>${cells.map((c) => `<td>${c}</td>`).join("")}</tr>`)
      .join("");
    return `
      <table>
        <thead><tr>${headHtml}</tr></thead>
        <tbody>${bodyHtml}</tbody>
      </table>
    `;
  }

  function render(status) {
    if (!status) {
      content.innerHTML = `<div class="muted">无状态数据</div>`;
      return;
    }

    const git = status.git || {};
    const docker = status.docker || {};
    const ports = status.ports || [];
    const http = status.http || [];
    const files = status.files || {};

    const dirtyTag = git.dirty ? `<span class="tag tag-failed">dirty</span>` : `<span class="tag tag-success">clean</span>`;
    const dockerTag = docker.ok ? `<span class="tag tag-success">ok</span>` : `<span class="tag tag-failed">fail</span>`;
    const seedTag = files.seed_result_ok ? `<span class="tag tag-success">ok</span>` : `<span class="tag tag-failed">missing</span>`;

    const portsTable = renderTable(
      ["服务", "地址", "监听"],
      ports.map((p) => [
        escapeHTML(p.name),
        `<code>${escapeHTML(p.addr)}</code>`,
        p.ok ? `<span class="tag tag-success">ok</span>` : `<span class="tag tag-failed">down</span>`,
      ])
    );

    const httpTable = renderTable(
      ["检查项", "URL", "结果"],
      http.map((h) => [
        escapeHTML(h.name),
        `<code>${escapeHTML(h.url)}</code>`,
        h.ok ? `<span class="tag tag-success">${escapeHTML(h.status_code || 0)}</span>` : `<span class="tag tag-failed">fail</span>`,
      ])
    );

    const dockerTable = renderTable(
      ["容器", "状态", "端口"],
      (docker.containers || []).map((c) => [
        escapeHTML(c.name),
        escapeHTML(c.status),
        `<code>${escapeHTML(c.ports)}</code>`,
      ])
    );

    const dockerErr = docker.ok ? "" : `<div class="status status-error" style="margin-top:8px">docker ps 失败：${escapeHTML(docker.error || "")}</div>`;

    content.innerHTML = `
      <div class="row" style="gap:14px; margin-bottom:8px">
        <div><strong>Git</strong>: <code>${escapeHTML(git.branch || "-")}</code> <code>${escapeHTML(git.commit || "-")}</code> ${dirtyTag}</div>
        <div><strong>Docker</strong>: ${dockerTag}</div>
        <div><strong>Seed</strong>: ${seedTag}</div>
      </div>

      <div class="grid-2">
        <div class="card-inner">
          <div class="row" style="justify-content:space-between">
            <strong>端口监听</strong>
            <span class="muted">TCP dial</span>
          </div>
          ${portsTable}
        </div>
        <div class="card-inner">
          <div class="row" style="justify-content:space-between">
            <strong>HTTP 健康检查</strong>
            <span class="muted">2xx 即 ok</span>
          </div>
          ${httpTable}
        </div>
      </div>

      <div class="card-inner" style="margin-top:10px">
        <div class="row" style="justify-content:space-between">
          <strong>Docker 容器</strong>
          <span class="muted">docker ps</span>
        </div>
        ${dockerTable}
        ${dockerErr}
      </div>

      <div class="muted" style="margin-top:8px">
        Seed 文件：<code>${escapeHTML(files.seed_result_path || "-")}</code>
      </div>
    `;
  }

  function setLoading() {
    content.innerHTML = `<div class="muted">加载中...</div>`;
  }

  function setError(message) {
    content.innerHTML = `<div class="status status-error">加载失败：${escapeHTML(message || "")}</div>`;
  }

  return {
    render,
    setLoading,
    setError,
  };
}

