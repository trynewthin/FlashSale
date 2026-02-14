// task-table 展示任务白名单与危险标识。
export function createTaskTable(container) {
  container.innerHTML = `
    <table>
      <thead>
        <tr>
          <th>Task ID</th>
          <th>名称</th>
          <th>说明</th>
          <th>危险</th>
        </tr>
      </thead>
      <tbody id="ops-task-table-body"></tbody>
    </table>
  `;
  const tbody = container.querySelector("#ops-task-table-body");

  return {
    render(tasks) {
      tbody.innerHTML = "";
      tasks.forEach((task) => {
        const tr = document.createElement("tr");
        tr.innerHTML = `
          <td>${escapeHtml(task.id)}</td>
          <td>${escapeHtml(task.name)}</td>
          <td>${escapeHtml(task.description)}</td>
          <td>${task.dangerous ? "Y" : "N"}</td>
        `;
        tbody.appendChild(tr);
      });
    },
  };
}

function escapeHtml(value) {
  return String(value || "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
