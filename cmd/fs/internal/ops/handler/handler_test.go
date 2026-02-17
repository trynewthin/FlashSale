package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"flashsale/cmd/fs/internal/ops/model"
	"flashsale/cmd/fs/internal/ops/service"
)

func echoCommand() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo"}
	}
	return []string{"echo"}
}

func testMux(t *testing.T) (*http.ServeMux, *service.Runner) {
	t.Helper()
	tmpDir := t.TempDir()
	tasks := map[string]model.TaskDef{
		"test.echo": {
			ID:          "test.echo",
			Name:        "Echo Test",
			Command:     echoCommand(),
			DefaultArgs: []string{"hello"},
		},
	}
	runner := service.NewRunner(tmpDir, tasks)

	env := &service.EnvContext{
		RepoRoot:        tmpDir,
		AdminGatewayURL: "http://127.0.0.1:8083",
		UserGatewayURL:  "http://127.0.0.1:8082",
		EtcdEndpoint:    "localhost:2379",
		PrometheusURL:   "http://localhost:9090",
		DeploymentMode:  model.DeploymentModeHostProcess,
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, RouterDeps{
		AuthKey: "", // 不启用认证
		Runner:  runner,
		Env:     env,
	})
	return mux, runner
}

func testMuxWithAuth(t *testing.T, key string) (*http.ServeMux, *service.Runner) {
	t.Helper()
	tmpDir := t.TempDir()
	tasks := map[string]model.TaskDef{
		"test.echo": {
			ID:      "test.echo",
			Name:    "Echo Test",
			Command: echoCommand(),
		},
	}
	runner := service.NewRunner(tmpDir, tasks)

	env := &service.EnvContext{
		RepoRoot:        tmpDir,
		AdminGatewayURL: "http://127.0.0.1:8083",
		UserGatewayURL:  "http://127.0.0.1:8082",
		DeploymentMode:  model.DeploymentModeHostProcess,
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, RouterDeps{
		AuthKey: key,
		Runner:  runner,
		Env:     env,
	})
	return mux, runner
}

type apiResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func parseResp(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v\nbody: %s", err, w.Body.String())
	}
	return resp
}

// ─── Healthz ───

func TestHealthz(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/healthz", "", nil)
	if w.Code != 200 {
		t.Fatalf("healthz status = %d, want 200", w.Code)
	}
	resp := parseResp(t, w)
	if resp.Code != "OK" {
		t.Fatalf("healthz code = %q, want OK", resp.Code)
	}
}

// ─── Auth ───

func TestAuth_NoKeyRequired(t *testing.T) {
	mux, _ := testMux(t) // AuthKey = ""
	w := doRequest(t, mux, "GET", "/api/v1/tasks", "", nil)
	if w.Code != 200 {
		t.Fatalf("无认证时应返回 200, got %d", w.Code)
	}
}

func TestAuth_ValidKey(t *testing.T) {
	mux, _ := testMuxWithAuth(t, "test-secret-key")
	w := doRequest(t, mux, "GET", "/api/v1/tasks", "", map[string]string{
		"X-Ops-Key": "test-secret-key",
	})
	if w.Code != 200 {
		t.Fatalf("正确 Key 应返回 200, got %d", w.Code)
	}
}

func TestAuth_InvalidKey(t *testing.T) {
	mux, _ := testMuxWithAuth(t, "test-secret-key")
	w := doRequest(t, mux, "GET", "/api/v1/tasks", "", map[string]string{
		"X-Ops-Key": "wrong-key",
	})
	if w.Code != 401 {
		t.Fatalf("错误 Key 应返回 401, got %d", w.Code)
	}
}

func TestAuth_NoKey(t *testing.T) {
	mux, _ := testMuxWithAuth(t, "test-secret-key")
	w := doRequest(t, mux, "GET", "/api/v1/tasks", "", nil)
	if w.Code != 401 {
		t.Fatalf("缺少 Key 应返回 401, got %d", w.Code)
	}
}

func TestAuth_QueryParam(t *testing.T) {
	mux, _ := testMuxWithAuth(t, "test-secret-key")
	w := doRequest(t, mux, "GET", "/api/v1/tasks?key=test-secret-key", "", nil)
	if w.Code != 200 {
		t.Fatalf("Query 参数 Key 应返回 200, got %d", w.Code)
	}
}

// ─── Tasks ───

func TestListTasks(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/api/v1/tasks", "", nil)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	resp := parseResp(t, w)
	var data struct {
		Tasks []model.TaskDef `json:"tasks"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("解析 tasks 失败: %v", err)
	}
	if len(data.Tasks) != 1 {
		t.Fatalf("任务数 = %d, want 1", len(data.Tasks))
	}
	if data.Tasks[0].ID != "test.echo" {
		t.Fatalf("任务 ID = %q, want test.echo", data.Tasks[0].ID)
	}
}

// ─── Create Job ───

func TestCreateJob_Success(t *testing.T) {
	mux, _ := testMux(t)
	body := `{"task":"test.echo","args":["extra"]}`
	w := doRequest(t, mux, "POST", "/api/v1/jobs", body, nil)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	resp := parseResp(t, w)
	var data struct {
		Job model.JobDetail `json:"job"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("解析 job 失败: %v", err)
	}
	if data.Job.ID == "" {
		t.Fatal("job ID 不应为空")
	}
	if data.Job.TaskID != "test.echo" {
		t.Fatalf("TaskID = %q, want test.echo", data.Job.TaskID)
	}
}

func TestCreateJob_UnknownTask(t *testing.T) {
	mux, _ := testMux(t)
	body := `{"task":"nonexistent","args":[]}`
	w := doRequest(t, mux, "POST", "/api/v1/jobs", body, nil)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateJob_BadBody(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "POST", "/api/v1/jobs", "not-json{{{", nil)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// ─── List Jobs ───

func TestListJobs_Empty(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/api/v1/jobs", "", nil)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	resp := parseResp(t, w)
	var data struct {
		Jobs []model.JobSummary `json:"jobs"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("解析 jobs 失败: %v", err)
	}
	if len(data.Jobs) != 0 {
		t.Fatalf("空 runner 应返回 0 个任务, got %d", len(data.Jobs))
	}
}

func TestListJobs_WithLimit(t *testing.T) {
	mux, runner := testMux(t)
	// 创建 3 个任务
	for i := 0; i < 3; i++ {
		runner.StartJob("test.echo", nil)
	}
	w := doRequest(t, mux, "GET", "/api/v1/jobs?limit=2", "", nil)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	resp := parseResp(t, w)
	var data struct {
		Jobs []model.JobSummary `json:"jobs"`
	}
	json.Unmarshal(resp.Data, &data)
	if len(data.Jobs) != 2 {
		t.Fatalf("limit=2 应返回 2 个任务, got %d", len(data.Jobs))
	}
}

// ─── Get Job By ID ───

func TestGetJob_NotFound(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/api/v1/jobs/nonexistent-id", "", nil)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetJob_Found(t *testing.T) {
	mux, runner := testMux(t)
	detail, _ := runner.StartJob("test.echo", nil)
	w := doRequest(t, mux, "GET", "/api/v1/jobs/"+detail.ID, "", nil)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
}

// ─── Get Job Log ───

func TestGetJobLog_NotFound(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/api/v1/jobs/nonexistent/log", "", nil)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetJobLog_Found(t *testing.T) {
	mux, runner := testMux(t)
	detail, _ := runner.StartJob("test.echo", nil)

	// 等待完成
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, _ := runner.GetJob(detail.ID)
		if job.Status == model.JobSuccess || job.Status == model.JobFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	w := doRequest(t, mux, "GET", "/api/v1/jobs/"+detail.ID+"/log", "", nil)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	resp := parseResp(t, w)
	var data struct {
		Log string `json:"log"`
	}
	json.Unmarshal(resp.Data, &data)
	if data.Log == "" {
		t.Fatal("日志不应为空")
	}
}

// ─── Metrics Catalog ───

func TestMetricsCatalog(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/api/v1/metrics/catalog", "", nil)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	resp := parseResp(t, w)
	var data struct {
		Metrics []model.MetricDef `json:"metrics"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("解析 metrics catalog 失败: %v", err)
	}
	if len(data.Metrics) == 0 {
		t.Fatal("指标列表不应为空")
	}
}
