// task-runner-form 负责任务选择、参数输入与执行触发。
export function createTaskRunnerForm(container, options) {
  container.innerHTML = `
    <h3>任务执行</h3>
    <div class="row">
      <select id="ops-task-select" style="min-width:360px"></select>
      <input id="ops-task-args" placeholder="额外参数，例如: -rate 120 -open-duration 30s" style="min-width:360px" />
      <button id="ops-task-run">执行任务</button>
      <button id="ops-task-refresh" class="secondary">刷新</button>
    </div>
  `;

  const taskSelect = container.querySelector("#ops-task-select");
  const argsInput = container.querySelector("#ops-task-args");
  const runBtn = container.querySelector("#ops-task-run");
  const refreshBtn = container.querySelector("#ops-task-refresh");

  runBtn.addEventListener("click", () => {
    options.onRun({
      task: taskSelect.value,
      args: parseArgs(argsInput.value),
    });
  });
  refreshBtn.addEventListener("click", () => options.onRefresh());

  return {
    setTasks(tasks) {
      taskSelect.innerHTML = "";
      tasks.forEach((task) => {
        const option = document.createElement("option");
        option.value = task.id;
        option.textContent = `${task.id} - ${task.name}`;
        taskSelect.appendChild(option);
      });
    },
    setArgs(value) {
      argsInput.value = value || "";
    },
  };
}

function parseArgs(raw) {
  const value = (raw || "").trim();
  if (!value) {
    return [];
  }
  return value.split(/\s+/);
}
