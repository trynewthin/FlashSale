package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"flashsale/cmd/fs/internal/ops/model"
)

// ─── 验证正则 ───

var (
	ContainerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	ServiceNamePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

// BuildContainerRuntimeSnapshot 构建容器编排的完整快照。
func BuildContainerRuntimeSnapshot(repoRoot string) model.ContainerRuntimeSnapshot {
	out := model.ContainerRuntimeSnapshot{
		GeneratedAtUnix: time.Now().Unix(),
		Compose:         DetectComposeContext(repoRoot),
	}

	containers, listErr := ListDockerContainers(repoRoot)
	if listErr != nil {
		out.Error = listErr.Error()
	}
	if project := strings.TrimSpace(out.Compose.Project); project != "" {
		containers = FilterContainersByProject(containers, project)
	}
	out.Containers = containers

	// 与状态页的部署模式保持一致。
	dockerContainers := make([]model.DockerContainer, 0, len(containers))
	for _, c := range containers {
		dockerContainers = append(dockerContainers, model.DockerContainer{
			Name:   c.Name,
			Status: c.Status,
			Ports:  c.Ports,
		})
	}
	out.DeploymentMode = DetectDeploymentMode(dockerContainers)

	serviceMap := map[string]*model.ServiceRuntime{}
	catalog := DefaultServiceCatalog()
	for _, def := range catalog {
		cp := def
		serviceMap[cp.Name] = &model.ServiceRuntime{
			Name:            cp.Name,
			Role:            cp.Role,
			Scalable:        cp.Scalable,
			Optional:        cp.Optional,
			Absent:          true,
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
			serviceMap[svc] = &model.ServiceRuntime{
				Name:            svc,
				Role:            InferServiceRole(svc),
				Scalable:        false,
				Replicas:        0,
				RunningReplicas: 0,
				DependsOn:       []string{},
			}
		}
		serviceMap[svc].Absent = false
		serviceMap[svc].Replicas++
		if c.Running {
			serviceMap[svc].RunningReplicas++
		}
	}

	out.Services = make([]model.ServiceRuntime, 0, len(serviceMap))
	for _, item := range serviceMap {
		out.Services = append(out.Services, *item)
	}
	sort.Slice(out.Services, func(i, j int) bool {
		ri := RoleOrder(out.Services[i].Role)
		rj := RoleOrder(out.Services[j].Role)
		if ri != rj {
			return ri < rj
		}
		return out.Services[i].Name < out.Services[j].Name
	})
	out.Edges = BuildServiceEdges(catalog)

	return out
}

// FilterContainersByProject 按 compose project 过滤容器。
func FilterContainersByProject(containers []model.ContainerState, project string) []model.ContainerState {
	filtered := make([]model.ContainerState, 0, len(containers))
	for _, item := range containers {
		if strings.TrimSpace(item.Project) == project {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// RoleOrder 返回角色排序权重。
func RoleOrder(role string) int {
	switch role {
	case "ingress":
		return 1
	case "gateway":
		return 2
	case "rpc":
		return 3
	case "infra":
		return 4
	case "observability":
		return 5
	case "job":
		return 6
	default:
		return 9
	}
}

// InferServiceRole 从服务名推断角色。
func InferServiceRole(name string) string {
	switch {
	case strings.Contains(name, "gateway"):
		return "gateway"
	case strings.HasSuffix(name, "-rpc"):
		return "rpc"
	case name == "nginx":
		return "ingress"
	case name == "mysql" || name == "redis" || name == "kafka" || name == "etcd":
		return "infra"
	case name == "ops-control":
		return "gateway"
	case name == "jaeger" || name == "prometheus" || name == "grafana":
		return "observability"
	default:
		return "other"
	}
}

// BuildServiceEdges 根据服务目录生成依赖边。
func BuildServiceEdges(catalog map[string]model.ServiceDef) []model.ServiceEdge {
	edges := make([]model.ServiceEdge, 0)
	seen := map[string]struct{}{}
	addEdge := func(from, to, typ string) {
		key := from + "|" + to + "|" + typ
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		edges = append(edges, model.ServiceEdge{From: from, To: to, Type: typ})
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

// DefaultServiceCatalog 返回默认的服务目录。
func DefaultServiceCatalog() map[string]model.ServiceDef {
	list := []model.ServiceDef{
		// ingress
		{Name: "nginx", Role: "ingress", Scalable: false, DependsOn: []string{"user-gateway", "admin-gateway"}},

		// gateway
		{Name: "user-gateway", Role: "gateway", Scalable: true, DependsOn: []string{"user-rpc", "product-rpc", "order-rpc", "seckill-rpc"}},
		{Name: "admin-gateway", Role: "gateway", Scalable: true, DependsOn: []string{"user-rpc", "admin-rpc", "product-rpc", "order-rpc", "seckill-rpc"}},

		// rpc
		{Name: "user-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis", "etcd"}},
		{Name: "admin-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "etcd"}},
		{Name: "product-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis", "etcd"}},
		{Name: "order-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis", "kafka", "etcd"}},
		{Name: "seckill-rpc", Role: "rpc", Scalable: true, DependsOn: []string{"mysql", "redis", "kafka", "etcd", "product-rpc", "order-rpc"}},

		// infra
		{Name: "mysql", Role: "infra", Scalable: false, DependsOn: []string{}},
		{Name: "redis", Role: "infra", Scalable: false, DependsOn: []string{}},
		{Name: "kafka", Role: "infra", Scalable: false, DependsOn: []string{}},
		{Name: "etcd", Role: "infra", Scalable: false, DependsOn: []string{}},

		// observability
		{Name: "jaeger", Role: "observability", Scalable: false, Optional: true, DependsOn: []string{}},
		{Name: "prometheus", Role: "observability", Scalable: false, Optional: true, DependsOn: []string{}},
		{Name: "grafana", Role: "observability", Scalable: false, Optional: true, DependsOn: []string{"prometheus"}},

		// ops
		{Name: "ops-control", Role: "gateway", Scalable: false, DependsOn: []string{"mysql", "redis", "etcd"}},

		// job
		{Name: "mysql-init-user", Role: "job", Scalable: false, DependsOn: []string{"mysql"}},
		{Name: "kafka-init", Role: "job", Scalable: false, DependsOn: []string{"kafka"}},
		{Name: "migrate", Role: "job", Scalable: false, DependsOn: []string{"mysql"}},
	}
	out := make(map[string]model.ServiceDef, len(list))
	for _, item := range list {
		out[item.Name] = item
	}
	return out
}

// ─── Docker 命令 ───

// ContainerAction 执行容器动作（start/stop/restart）。
func ContainerAction(repoRoot, containerID, action string) (string, error) {
	return RunCmdWithErr(repoRoot, 30*time.Second, "docker", action, containerID)
}

// ScaleService 对指定服务执行扩缩容。
// --no-recreate：仅创建/销毁副本，不因其他服务配置变更而级联 Recreate 已有容器。
func ScaleService(repoRoot string, composeCtx model.ComposeContext, service string, replicas int) (string, error) {
	args := ComposeCommandArgs(composeCtx,
		"up", "-d", "--no-recreate", "--scale", fmt.Sprintf("%s=%d", service, replicas), service)
	return RunCmdWithErr(repoRoot, 90*time.Second, "docker", args...)
}

// DetectComposeContext 检测 Docker Compose 文件上下文。
func DetectComposeContext(repoRoot string) model.ComposeContext {
	ctx := model.ComposeContext{}
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
		filepath.Join("configs", "deploy.env"),
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

// ComposeCommandArgs 构建 docker compose 命令参数。
func ComposeCommandArgs(composeCtx model.ComposeContext, tail ...string) []string {
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

// ListDockerContainers 列出所有 Docker 容器。
func ListDockerContainers(repoRoot string) ([]model.ContainerState, error) {
	out, err := RunCmdWithErr(repoRoot, 8*time.Second, "docker", "ps", "-a", "--format",
		"{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}\t{{.Label \"com.docker.compose.service\"}}\t{{.Label \"com.docker.compose.project\"}}")
	if err != nil {
		return nil, fmt.Errorf("读取 docker 容器失败: %w", err)
	}
	lines := strings.Split(out, "\n")
	containers := make([]model.ContainerState, 0, len(lines))
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
			service = InferServiceNameFromContainer(strings.TrimSpace(parts[1]))
		}
		containers = append(containers, model.ContainerState{
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

// InferServiceNameFromContainer 从容器名推断服务名。
func InferServiceNameFromContainer(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "-")
	if len(parts) >= 3 {
		idx, err := strconv.Atoi(parts[len(parts)-1])
		if err == nil && idx >= 0 {
			return strings.Join(parts[1:len(parts)-1], "-")
		}
	}
	return strings.TrimSpace(name)
}

// RunCmdWithErr 在指定目录执行命令，返回输出和错误。
func RunCmdWithErr(dir string, timeout time.Duration, name string, args ...string) (string, error) {
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
