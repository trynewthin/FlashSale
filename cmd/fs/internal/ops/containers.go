// containers 提供容器编排相关只读与控制接口（状态、容器动作、服务扩缩容）。
package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ContainerRuntimeSnapshot struct {
	GeneratedAtUnix int64            `json:"generated_at_unix"`
	DeploymentMode  string           `json:"deployment_mode"`
	Compose         ComposeContext   `json:"compose"`
	Services        []ServiceRuntime `json:"services"`
	Containers      []ContainerState `json:"containers"`
	Edges           []ServiceEdge    `json:"edges"`
	Error           string           `json:"error,omitempty"`
}

type ComposeContext struct {
	ComposeFile string `json:"compose_file"`
	EnvFile     string `json:"env_file"`
	Project     string `json:"project"`
	Detected    bool   `json:"detected"`
}

type ServiceRuntime struct {
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	Scalable        bool     `json:"scalable"`
	Replicas        int      `json:"replicas"`
	RunningReplicas int      `json:"running_replicas"`
	DependsOn       []string `json:"depends_on"`
}

type ContainerState struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Service string `json:"service"`
	Project string `json:"project"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Health  string `json:"health"`
	Ports   string `json:"ports"`
	Running bool   `json:"running"`
}

type ServiceEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type serviceDef struct {
	Name      string
	Role      string
	Scalable  bool
	DependsOn []string
}

var (
	containerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	serviceNamePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

func (s *Server) getContainersStatus(w http.ResponseWriter, _ *http.Request) {
	snap := buildContainerRuntimeSnapshot(s.runner.repoRoot)
	writeOK(w, map[string]any{"status": snap})
}

type containerActionReq struct {
	Action string `json:"action"`
}

func (s *Server) containerAction(w http.ResponseWriter, r *http.Request) {
	containerID := strings.TrimSpace(r.PathValue("container_id"))
	if !containerNamePattern.MatchString(containerID) {
		writeErr(w, http.StatusBadRequest, "container_id 非法")
		return
	}
	var req containerActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "start" && action != "stop" && action != "restart" {
		writeErr(w, http.StatusBadRequest, "action 仅支持 start/stop/restart")
		return
	}
	out, err := runCmdWithErr(s.runner.repoRoot, 30*time.Second, "docker", action, containerID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, strings.TrimSpace(out))
		return
	}
	writeOK(w, map[string]any{
		"container_id": containerID,
		"action":       action,
		"output":       strings.TrimSpace(out),
	})
}

type serviceScaleReq struct {
	Replicas int `json:"replicas"`
}

func (s *Server) scaleService(w http.ResponseWriter, r *http.Request) {
	service := strings.TrimSpace(strings.ToLower(r.PathValue("service")))
	if !serviceNamePattern.MatchString(service) {
		writeErr(w, http.StatusBadRequest, "service 非法")
		return
	}
	var req serviceScaleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体非法")
		return
	}
	if req.Replicas < 0 || req.Replicas > 20 {
		writeErr(w, http.StatusBadRequest, "replicas 范围必须在 0..20")
		return
	}

	catalog := defaultServiceCatalog()
	def, ok := catalog[service]
	if !ok {
		writeErr(w, http.StatusNotFound, "服务不存在")
		return
	}
	if !def.Scalable {
		writeErr(w, http.StatusBadRequest, "该服务不支持扩缩容")
		return
	}

	composeCtx := detectComposeContext(s.runner.repoRoot)
	if strings.TrimSpace(composeCtx.ComposeFile) == "" {
		writeErr(w, http.StatusBadRequest, "未检测到 compose 文件")
		return
	}
	args := composeCommandArgs(composeCtx,
		"up", "-d", "--scale", fmt.Sprintf("%s=%d", service, req.Replicas), service)
	out, err := runCmdWithErr(s.runner.repoRoot, 90*time.Second, "docker", args...)
	if err != nil {
		writeErr(w, http.StatusBadRequest, strings.TrimSpace(out))
		return
	}
	writeOK(w, map[string]any{
		"service":  service,
		"replicas": req.Replicas,
		"output":   strings.TrimSpace(out),
	})
}

func buildContainerRuntimeSnapshot(repoRoot string) ContainerRuntimeSnapshot {
	out := ContainerRuntimeSnapshot{
		GeneratedAtUnix: time.Now().Unix(),
		Compose:         detectComposeContext(repoRoot),
	}

	containers, listErr := listDockerContainers(repoRoot)
	if listErr != nil {
		out.Error = listErr.Error()
	}
	if project := strings.TrimSpace(out.Compose.Project); project != "" {
		containers = filterContainersByProject(containers, project)
	}
	out.Containers = containers

	// 与状态页的部署模式保持一致，确保口径统一。
	dockerContainers := make([]DockerContainer, 0, len(containers))
	for _, c := range containers {
		dockerContainers = append(dockerContainers, DockerContainer{
			Name:   c.Name,
			Status: c.Status,
			Ports:  c.Ports,
		})
	}
	out.DeploymentMode = detectDeploymentMode(dockerContainers)

	serviceMap := map[string]*ServiceRuntime{}
	catalog := defaultServiceCatalog()
	for _, def := range catalog {
		cp := def
		serviceMap[cp.Name] = &ServiceRuntime{
			Name:            cp.Name,
			Role:            cp.Role,
			Scalable:        cp.Scalable,
			Replicas:        0,
			RunningReplicas: 0,
			DependsOn:       append([]string{}, cp.DependsOn...),
		}
	}
	for _, c := range containers {
		svc := strings.TrimSpace(c.Service)
		if svc == "" {
			svc = "unknown"
		}
		if _, ok := serviceMap[svc]; !ok {
			serviceMap[svc] = &ServiceRuntime{
				Name:            svc,
				Role:            inferServiceRole(svc),
				Scalable:        false,
				Replicas:        0,
				RunningReplicas: 0,
				DependsOn:       nil,
			}
		}
		serviceMap[svc].Replicas++
		if c.Running {
			serviceMap[svc].RunningReplicas++
		}
	}

	out.Services = make([]ServiceRuntime, 0, len(serviceMap))
	for _, item := range serviceMap {
		out.Services = append(out.Services, *item)
	}
	sort.Slice(out.Services, func(i, j int) bool {
		ri := roleOrder(out.Services[i].Role)
		rj := roleOrder(out.Services[j].Role)
		if ri != rj {
			return ri < rj
		}
		return out.Services[i].Name < out.Services[j].Name
	})
	out.Edges = buildServiceEdges(catalog)

	return out
}

func filterContainersByProject(containers []ContainerState, project string) []ContainerState {
	filtered := make([]ContainerState, 0, len(containers))
	for _, item := range containers {
		if strings.TrimSpace(item.Project) == project {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func roleOrder(role string) int {
	switch role {
	case "ingress":
		return 1
	case "gateway":
		return 2
	case "rpc":
		return 3
	case "infra":
		return 4
	case "job":
		return 5
	default:
		return 9
	}
}

func inferServiceRole(name string) string {
	switch {
	case strings.Contains(name, "gateway"):
		return "gateway"
	case strings.HasSuffix(name, "-rpc"):
		return "rpc"
	case name == "nginx":
		return "ingress"
	case name == "mysql" || name == "redis" || name == "kafka":
		return "infra"
	default:
		return "other"
	}
}

func buildServiceEdges(catalog map[string]serviceDef) []ServiceEdge {
	edges := make([]ServiceEdge, 0)
	seen := map[string]struct{}{}
	addEdge := func(from, to, typ string) {
		key := from + "|" + to + "|" + typ
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		edges = append(edges, ServiceEdge{From: from, To: to, Type: typ})
	}
	for _, def := range catalog {
		for _, dep := range def.DependsOn {
			addEdge(def.Name, dep, "depends_on")
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Type < edges[j].Type
	})
	return edges
}

func defaultServiceCatalog() map[string]serviceDef {
	list := []serviceDef{
		{Name: "nginx", Role: "ingress", Scalable: false, DependsOn: []string{"user-gateway", "admin-gateway"}},
		{Name: "grpc-lb", Role: "ingress", Scalable: false, DependsOn: []string{"seckill-rpc"}},

		{Name: "user-gateway", Role: "gateway", Scalable: true, DependsOn: []string{"user-rpc", "product-rpc", "order-rpc", "seckill-rpc"}},
		{Name: "admin-gateway", Role: "gateway", Scalable: true, DependsOn: []string{"user-rpc", "admin-rpc", "product-rpc", "order-rpc", "seckill-rpc"}},

		{Name: "user-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis"}},
		{Name: "admin-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis"}},
		{Name: "product-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis"}},
		{Name: "order-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis", "kafka"}},
		{Name: "seckill-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis", "kafka", "product-rpc", "order-rpc"}},

		{Name: "mysql", Role: "infra", Scalable: false, DependsOn: nil},
		{Name: "redis", Role: "infra", Scalable: false, DependsOn: nil},
		{Name: "kafka", Role: "infra", Scalable: false, DependsOn: nil},

		{Name: "mysql-init-user", Role: "job", Scalable: false, DependsOn: []string{"mysql"}},
		{Name: "kafka-init", Role: "job", Scalable: false, DependsOn: []string{"kafka"}},
		{Name: "migrate", Role: "job", Scalable: false, DependsOn: []string{"mysql"}},
	}
	out := make(map[string]serviceDef, len(list))
	for _, item := range list {
		out[item.Name] = item
	}
	return out
}

func detectComposeContext(repoRoot string) ComposeContext {
	ctx := ComposeContext{}
	composeCandidates := []string{
		strings.TrimSpace(os.Getenv("FLASHSALE_APP_COMPOSE_FILE")),
		filepath.Join("deploy", "compose", "docker-compose.app.yml"),
		filepath.Join("deploy", "compose", "docker-compose.yml"),
	}
	for _, cand := range composeCandidates {
		if strings.TrimSpace(cand) == "" {
			continue
		}
		abs := cand
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(repoRoot, cand)
		}
		if _, err := os.Stat(abs); err == nil {
			ctx.ComposeFile = abs
			ctx.Detected = true
			ctx.Project = detectComposeProjectFromFile(abs)
			break
		}
	}

	envCandidates := []string{
		strings.TrimSpace(os.Getenv("FLASHSALE_APP_ENV_FILE")),
		filepath.Join("configs", "prod", "server.env"),
		filepath.Join("configs", "prod", "prod.env"),
		filepath.Join("configs", "local", "dev.env"),
	}
	for _, cand := range envCandidates {
		if strings.TrimSpace(cand) == "" {
			continue
		}
		abs := cand
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(repoRoot, cand)
		}
		if _, err := os.Stat(abs); err == nil {
			ctx.EnvFile = abs
			break
		}
	}
	return ctx
}

func detectComposeProjectFromFile(composeFile string) string {
	content, err := os.ReadFile(composeFile)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(content), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "name:") {
			continue
		}
		project := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		project = strings.Trim(project, `"'`)
		return project
	}
	return ""
}

func composeCommandArgs(composeCtx ComposeContext, tail ...string) []string {
	args := []string{"compose"}
	if strings.TrimSpace(composeCtx.EnvFile) != "" {
		args = append(args, "--env-file", composeCtx.EnvFile)
	}
	if strings.TrimSpace(composeCtx.ComposeFile) != "" {
		args = append(args, "-f", composeCtx.ComposeFile)
	}
	args = append(args, tail...)
	return args
}

func listDockerContainers(repoRoot string) ([]ContainerState, error) {
	out, err := runCmdWithErr(repoRoot, 8*time.Second, "docker", "ps", "-a", "--format",
		"{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}\t{{.Label \"com.docker.compose.service\"}}\t{{.Label \"com.docker.compose.project\"}}")
	if err != nil {
		return nil, fmt.Errorf("读取 docker 容器失败: %w", err)
	}
	lines := strings.Split(out, "\n")
	containers := make([]ContainerState, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}
		status := strings.TrimSpace(parts[3])
		running := strings.HasPrefix(strings.ToLower(status), "up")
		health := "unknown"
		lowerStatus := strings.ToLower(status)
		switch {
		case strings.Contains(lowerStatus, "(healthy)"):
			health = "healthy"
		case strings.Contains(lowerStatus, "(unhealthy)"):
			health = "unhealthy"
		case running:
			health = "running"
		default:
			health = "stopped"
		}
		service := strings.TrimSpace(parts[5])
		if service == "" {
			service = inferServiceNameFromContainer(strings.TrimSpace(parts[1]))
		}
		containers = append(containers, ContainerState{
			ID:      strings.TrimSpace(parts[0]),
			Name:    strings.TrimSpace(parts[1]),
			Image:   strings.TrimSpace(parts[2]),
			Status:  status,
			Ports:   strings.TrimSpace(parts[4]),
			Service: service,
			Project: strings.TrimSpace(parts[6]),
			Health:  health,
			Running: running,
		})
	}
	sort.Slice(containers, func(i, j int) bool {
		if containers[i].Service != containers[j].Service {
			return containers[i].Service < containers[j].Service
		}
		return containers[i].Name < containers[j].Name
	})
	return containers, nil
}

func inferServiceNameFromContainer(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "-")
	if len(parts) >= 3 {
		// 约定格式: <project>-<service>-<index>
		idx, err := strconv.Atoi(parts[len(parts)-1])
		if err == nil && idx >= 0 {
			return strings.Join(parts[1:len(parts)-1], "-")
		}
	}
	return strings.TrimSpace(name)
}

func runCmdWithErr(dir string, timeout time.Duration, name string, args ...string) (string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return "", err
		}
		return string(out), fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
