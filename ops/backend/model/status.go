package model

type StatusSnapshot struct {
	NowUnix        int64  `json:"now_unix"`
	RepoRoot       string `json:"repo_root"`
	DeploymentMode string `json:"deployment_mode"`
	Git            struct {
		Branch string `json:"branch"`
		Commit string `json:"commit"`
		Dirty  bool   `json:"dirty"`
	} `json:"git"`
	Ports  []PortStatus `json:"ports"`
	HTTP   []HTTPStatus `json:"http"`
	Docker struct {
		OK         bool              `json:"ok"`
		Error      string            `json:"error,omitempty"`
		Containers []DockerContainer `json:"containers,omitempty"`
		Raw        string            `json:"raw,omitempty"`
	} `json:"docker"`
	Files struct {
		SeedResultPath string `json:"seed_result_path"`
		SeedResultOK   bool   `json:"seed_result_ok"`
	} `json:"files"`
}

type PortStatus struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	OK   bool   `json:"ok"`
}

type HTTPStatus struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
}

type DockerContainer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Ports  string `json:"ports"`
}

const (
	DeploymentModeHostProcess = "host_process"
	DeploymentModeDockerApp   = "docker_app"
)
