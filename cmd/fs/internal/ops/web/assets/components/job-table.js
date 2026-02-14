// job-table 展示任务执行历史并提供日志查看入口。
export function createJobTable(container, options) {
  container.innerHTML = `
    <table>
      <thead>
        <tr>
          <th>Job ID</th>
          <th>Task</th>
          <th>状态</th>
          <th>创建时间</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody id="ops-job-table-body"></tbody>
    </table>
  `;
  const tbody = container.querySelector("#ops-job-table-body");

  return {
    render(jobs) {
      tbody.innerHTML = "";
      jobs.forEach((job) => {
        const tr = document.createElement("tr");
        tr.innerHTML = `
          <td>${escapeHtml(job.id)}</td>
          <td>${escapeHtml(job.task_id)}</td>
          <td>${renderStatus(job.status)}</td>
          <td>${escapeHtml(job.created_at || "")}</td>
          <td><button class="secondary" data-job-id="${escapeHtml(job.id)}">查看日志</button></td>
        `;
        const button = tr.querySelector("button[data-job-id]");
        button.addEventListener("click", () => options.onViewLog(job.id));
        tbody.appendChild(tr);
      });
    },
  };
}

function renderStatus(status) {
  const value = String(status || "");
  if (value === "success") {
    return `<span class="tag tag-success">${value}</span>`;
  }
  if (value === "failed") {
    return `<span class="tag tag-failed">${value}</span>`;
  }
  return `<span class="tag tag-running">${escapeHtml(value)}</span>`;
}

function escapeHtml(value) {
  return String(value || "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
