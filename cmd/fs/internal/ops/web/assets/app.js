// app.js 是 ops 控制台前端入口，负责组装组件与页面状态。
import { OpsApiClient } from "./api-client.js";
import { createStore } from "./store.js";
import { createStatusBar } from "./components/status-bar.js";
import { createAuthPanel } from "./components/auth-panel.js";
import { createTaskRunnerForm } from "./components/task-runner-form.js";
import { createTaskTable } from "./components/task-table.js";
import { createJobTable } from "./components/job-table.js";
import { createLogStreamPanel } from "./components/log-stream-panel.js";
import { createStatusPanel } from "./components/status-panel.js";
import { createServiceLogPanel } from "./components/service-log-panel.js";

const api = new OpsApiClient("");
const store = createStore({
  tasks: [],
  jobs: [],
  currentJobId: "",
});

const statusBar = createStatusBar(document.getElementById("status-bar"));
const statusPanel = createStatusPanel(document.getElementById("status-panel"), {
  onRefresh: async () => {
    await reloadStatus();
  },
});
const serviceLogPanel = createServiceLogPanel(document.getElementById("service-log-panel"), {
  api,
  statusBar,
});
const taskTable = createTaskTable(document.getElementById("task-table"));
const jobTable = createJobTable(document.getElementById("job-table"), {
  onViewLog: async (jobId) => {
    await viewJobLog(jobId);
  },
});
const logPanel = createLogStreamPanel(document.getElementById("log-stream"));

createAuthPanel(document.getElementById("auth-panel"), {
  initialKey: api.getAccessKey(),
  onSave: async (value) => {
    try {
      api.setAccessKey(value);
      statusBar.showOK("密钥已保存");
      await reloadAll();
    } catch (error) {
      statusBar.showError(error.message);
    }
  },
  onClear: () => {
    api.setAccessKey("");
    logPanel.stopStream();
    statusBar.showOK("密钥已清空");
  },
});

const taskRunner = createTaskRunnerForm(document.getElementById("task-runner"), {
  onRun: async ({ task, args }) => {
    try {
      const job = await api.createJob(task, args);
      statusBar.showOK(`任务已创建: ${job.id}`);
      store.setState({ currentJobId: job.id });
      await reloadJobs();
      await viewJobLog(job.id);
    } catch (error) {
      statusBar.showError(error.message);
    }
  },
  onRefresh: async () => {
    await reloadAll();
  },
});

store.subscribe((state) => {
  taskTable.render(state.tasks);
  jobTable.render(state.jobs);
  taskRunner.setTasks(state.tasks);
});

async function reloadTasks() {
  const tasks = await api.listTasks();
  store.setState({ tasks });
}

async function reloadJobs() {
  const jobs = await api.listJobs(30);
  store.setState({ jobs });
}

async function reloadStatus() {
  try {
    statusPanel.setLoading();
    const status = await api.getStatus();
    statusPanel.render(status);
  } catch (error) {
    statusPanel.setError(error.message);
  }
}

async function reloadAll() {
  statusBar.clear();
  await reloadStatus();
  await serviceLogPanel.refreshFiles();
  await reloadTasks();
  await reloadJobs();
  const { currentJobId } = store.getState();
  if (currentJobId) {
    await viewJobLog(currentJobId);
  }
}

// viewJobLog 先加载日志快照，再建立 SSE 实时流订阅。
async function viewJobLog(jobId) {
  const logText = await api.getJobLog(jobId);
  store.setState({ currentJobId: jobId });
  logPanel.setText(logText);
  logPanel.startStream(api.buildLogStreamUrl(jobId), {
    onError: () => {
      statusBar.showError("日志流已断开，请重新选择任务查看。");
    },
  });
}

setInterval(async () => {
  try {
    await reloadJobs();
  } catch (error) {
    statusBar.showError(error.message);
  }
}, 3000);

setInterval(async () => {
  await reloadStatus();
}, 5000);

reloadAll().catch((error) => {
  statusBar.showError(error.message);
});
