package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"flashsale/ops/backend/catalog"
	"flashsale/ops/backend/model"
	"flashsale/ops/backend/scheduler"
)

func echoCommand() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo"}
	}
	return []string{"echo"}
}

func testMux(t *testing.T) (*http.ServeMux, *scheduler.Scheduler) {
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
	schedulerSvc := scheduler.New(tmpDir, tasks)
	env := &catalog.EnvContext{
		RepoRoot:        tmpDir,
		AdminGatewayURL: "http://127.0.0.1:8083",
		UserGatewayURL:  "http://127.0.0.1:8082",
		EtcdEndpoint:    "localhost:2379",
		PrometheusURL:   "http://localhost:9090",
		DeploymentMode:  model.DeploymentModeHostProcess,
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, RouterDeps{AuthKey: "", Scheduler: schedulerSvc, Env: env})
	return mux, schedulerSvc
}

func testMuxWithAuth(t *testing.T, key string) (*http.ServeMux, *scheduler.Scheduler) {
	t.Helper()
	tmpDir := t.TempDir()
	tasks := map[string]model.TaskDef{
		"test.echo": {ID: "test.echo", Name: "Echo Test", Command: echoCommand()},
	}
	schedulerSvc := scheduler.New(tmpDir, tasks)
	env := &catalog.EnvContext{RepoRoot: tmpDir, AdminGatewayURL: "http://127.0.0.1:8083", UserGatewayURL: "http://127.0.0.1:8082", DeploymentMode: model.DeploymentModeHostProcess}
	mux := http.NewServeMux()
	RegisterRoutes(mux, RouterDeps{AuthKey: key, Scheduler: schedulerSvc, Env: env})
	return mux, schedulerSvc
}

type apiResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
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
		t.Fatalf("parse response failed: %v\nbody: %s", err, w.Body.String())
	}
	return resp
}

func waitJobDone(t *testing.T, schedulerSvc *scheduler.Scheduler, jobID string) model.JobDetail {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := schedulerSvc.GetJob(jobID)
		if ok && (job.Status == model.JobSuccess || job.Status == model.JobFailed) {
			return job
		}
		time.Sleep(50 * time.Millisecond)
	}
	job, _ := schedulerSvc.GetJob(jobID)
	return job
}

func TestHealthz(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/healthz", "", nil)
	if w.Code != 200 {
		t.Fatalf("healthz status = %d, want 200", w.Code)
	}
}

func TestAuth(t *testing.T) {
	mux, _ := testMuxWithAuth(t, "test-secret-key")
	if doRequest(t, mux, "GET", "/api/v1/tasks", "", nil).Code != 401 {
		t.Fatal("missing key should return 401")
	}
	if doRequest(t, mux, "GET", "/api/v1/tasks", "", map[string]string{"X-Ops-Key": "wrong-key"}).Code != 401 {
		t.Fatal("invalid key should return 401")
	}
	if doRequest(t, mux, "GET", "/api/v1/tasks", "", map[string]string{"X-Ops-Key": "test-secret-key"}).Code != 200 {
		t.Fatal("valid key should return 200")
	}
}

func TestListTasks(t *testing.T) {
	mux, _ := testMux(t)
	w := doRequest(t, mux, "GET", "/api/v1/tasks", "", nil)
	resp := parseResp(t, w)
	var data struct {
		Tasks []model.TaskDef `json:"tasks"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Tasks) != 1 || data.Tasks[0].ID != "test.echo" {
		t.Fatalf("unexpected tasks: %+v", data.Tasks)
	}
}

func TestCreateJobAndGetLog(t *testing.T) {
	mux, schedulerSvc := testMux(t)
	w := doRequest(t, mux, "POST", "/api/v1/jobs", `{"task":"test.echo","args":["extra"]}`, nil)
	if w.Code != 200 {
		t.Fatalf("create job status=%d body=%s", w.Code, w.Body.String())
	}
	resp := parseResp(t, w)
	var data struct {
		Job model.JobDetail `json:"job"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatal(err)
	}
	job := waitJobDone(t, schedulerSvc, data.Job.ID)
	if job.Status != model.JobSuccess {
		t.Fatalf("job status=%s", job.Status)
	}
	logResp := doRequest(t, mux, "GET", "/api/v1/jobs/"+data.Job.ID+"/log", "", nil)
	if logResp.Code != 200 {
		t.Fatalf("log status=%d", logResp.Code)
	}
}

func TestStreamJobLogUsesUnifiedEnvelope(t *testing.T) {
	mux, schedulerSvc := testMux(t)
	job, err := schedulerSvc.StartJob("test.echo", []string{"stream"})
	if err != nil {
		t.Fatal(err)
	}
	waitJobDone(t, schedulerSvc, job.ID)
	w := doRequest(t, mux, "GET", "/api/v1/jobs/"+job.ID+"/stream", "", nil)
	if w.Code != 200 {
		t.Fatalf("stream status=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "\"type\":\"snapshot\"") {
		t.Fatalf("expected snapshot envelope, got %s", body)
	}
	if !strings.Contains(body, "\"type\":\"done\"") {
		t.Fatalf("expected done envelope, got %s", body)
	}
	if !strings.Contains(body, "\"job_id\":\""+job.ID+"\"") {
		t.Fatalf("expected job id in stream body, got %s", body)
	}
}
