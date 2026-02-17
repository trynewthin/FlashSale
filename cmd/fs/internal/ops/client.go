// client 封装 ops-control API 调用，供 fs ops 子命令复用。
package ops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/ops/model"
)

// Client 是 ops-control 的 HTTP 客户端。
type Client struct {
	baseURL string
	authKey string
	httpCli *http.Client
}

// NewClient 创建客户端。
func NewClient(baseURL, authKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		authKey: strings.TrimSpace(authKey),
		httpCli: &http.Client{Timeout: 15 * time.Second},
	}
}

type clientResp struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) doJSON(method, path string, body any, out any) error {
	url := c.baseURL + path
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(buf)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.authKey != "" {
		req.Header.Set("X-Ops-Key", c.authKey)
	}
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var wrapper clientResp
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}
	if wrapper.Code != "OK" {
		return fmt.Errorf("api error: %s", wrapper.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(wrapper.Data, out)
}

// ListTasks 查询任务白名单。
func (c *Client) ListTasks() ([]model.TaskDef, error) {
	var data struct {
		Tasks []model.TaskDef `json:"tasks"`
	}
	if err := c.doJSON(http.MethodGet, "/api/v1/tasks", nil, &data); err != nil {
		return nil, err
	}
	return data.Tasks, nil
}

// CreateJob 创建任务。
func (c *Client) CreateJob(task string, args []string) (model.JobDetail, error) {
	var data struct {
		Job model.JobDetail `json:"job"`
	}
	if err := c.doJSON(http.MethodPost, "/api/v1/jobs", map[string]any{
		"task": task,
		"args": args,
	}, &data); err != nil {
		return model.JobDetail{}, err
	}
	return data.Job, nil
}

// ListJobs 查询任务列表。
func (c *Client) ListJobs(limit int) ([]model.JobSummary, error) {
	var data struct {
		Jobs []model.JobSummary `json:"jobs"`
	}
	path := fmt.Sprintf("/api/v1/jobs?limit=%d", limit)
	if err := c.doJSON(http.MethodGet, path, nil, &data); err != nil {
		return nil, err
	}
	return data.Jobs, nil
}

// GetJobLog 查询任务日志。
func (c *Client) GetJobLog(jobID string) (string, error) {
	var data struct {
		Log string `json:"log"`
	}
	path := fmt.Sprintf("/api/v1/jobs/%s/log", jobID)
	if err := c.doJSON(http.MethodGet, path, nil, &data); err != nil {
		return "", err
	}
	return data.Log, nil
}
