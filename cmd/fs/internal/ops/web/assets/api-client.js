// api-client.js 封装 ops-control 的 HTTP 请求与密钥注入逻辑。
const STORAGE_KEY = "flashsale_ops_key";

export class OpsApiClient {
  constructor(baseUrl = "") {
    this.baseUrl = baseUrl;
    this.accessKey = (localStorage.getItem(STORAGE_KEY) || "").trim();
  }

  getAccessKey() {
    return this.accessKey;
  }

  setAccessKey(key) {
    this.accessKey = (key || "").trim();
    if (this.accessKey) {
      localStorage.setItem(STORAGE_KEY, this.accessKey);
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  }

  async listTasks() {
    const data = await this.fetchJSON("/api/v1/tasks");
    return data.tasks || [];
  }

  async listJobs(limit = 30) {
    const data = await this.fetchJSON(`/api/v1/jobs?limit=${encodeURIComponent(String(limit))}`);
    return data.jobs || [];
  }

  async createJob(task, args) {
    const data = await this.fetchJSON("/api/v1/jobs", {
      method: "POST",
      body: JSON.stringify({ task, args }),
    });
    return data.job;
  }

  async getJobLog(jobId) {
    const data = await this.fetchJSON(`/api/v1/jobs/${encodeURIComponent(jobId)}/log`);
    return data.log || "";
  }

  // buildLogStreamUrl 生成 SSE 日志流地址，兼容 query 方式传递密钥。
  buildLogStreamUrl(jobId) {
    const key = this.getAccessKey();
    const query = key ? `?key=${encodeURIComponent(key)}` : "";
    return `${this.baseUrl}/api/v1/jobs/${encodeURIComponent(jobId)}/stream${query}`;
  }

  // fetchJSON 统一处理 code/message/data 包装响应。
  async fetchJSON(path, init = {}) {
    const headers = {
      ...(init.headers || {}),
    };
    if (this.accessKey) {
      headers["X-Ops-Key"] = this.accessKey;
    }
    if (init.body && !headers["Content-Type"]) {
      headers["Content-Type"] = "application/json";
    }
    const resp = await fetch(`${this.baseUrl}${path}`, { ...init, headers });
    const payload = await resp.json();
    if (payload.code !== "OK") {
      throw new Error(payload.message || "request failed");
    }
    return payload.data;
  }
}
