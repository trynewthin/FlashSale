package catalog

import (
	"os"
	"regexp"
	"strings"

	"flashsale/ops/backend/model"
)

// ObservabilityTools 可观测性工具定义。
var ObservabilityTools = []model.ObservabilityToolDef{
	{Name: "Jaeger", ServiceName: "jaeger", ContainerPort: "16686", Protocol: "http", EnvOverride: "FLASHSALE_JAEGER_URL"},
	{Name: "Prometheus", ServiceName: "prometheus", ContainerPort: "9090", Protocol: "http", EnvOverride: "FLASHSALE_PROMETHEUS_URL"},
	{Name: "Grafana", ServiceName: "grafana", ContainerPort: "3000", Protocol: "http", EnvOverride: "FLASHSALE_GRAFANA_URL"},
}

// hostPortPattern 匹配 docker ps 输出中的 host:port->container_port 映射。
var hostPortPattern = regexp.MustCompile(`(?:(\d+\.\d+\.\d+\.\d+)|::):(\d+)->(\d+)`)

// BuildObservabilityLinks 构建可观测性工具入口链接。
func BuildObservabilityLinks(repoRoot, reqHost string) model.ObservabilityLinks {
	containers, _ := ListDockerContainers(repoRoot)

	containerByService := map[string]model.ContainerState{}
	for _, c := range containers {
		svc := strings.TrimSpace(c.Service)
		if svc == "" {
			continue
		}
		if _, exists := containerByService[svc]; !exists || c.Running {
			containerByService[svc] = c
		}
	}

	return model.ObservabilityLinks{
		Jaeger:     buildObservabilityLink(ObservabilityTools[0], containerByService, reqHost),
		Prometheus: buildObservabilityLink(ObservabilityTools[1], containerByService, reqHost),
		Grafana:    buildObservabilityLink(ObservabilityTools[2], containerByService, reqHost),
	}
}

func buildObservabilityLink(tool model.ObservabilityToolDef, containers map[string]model.ContainerState, reqHost string) model.ObservabilityLink {
	link := model.ObservabilityLink{Name: tool.Name}

	if envURL := strings.TrimSpace(os.Getenv(tool.EnvOverride)); envURL != "" {
		link.URL = envURL
		link.Available = true
		return link
	}

	c, ok := containers[tool.ServiceName]
	if !ok {
		return link
	}

	link.Available = c.Running

	hostPort := ExtractHostPort(c.Ports, tool.ContainerPort)
	if hostPort == "" {
		hostPort = tool.ContainerPort
	}

	host := reqHost
	if host == "" {
		host = "localhost"
	}
	link.URL = tool.Protocol + "://" + host + ":" + hostPort

	return link
}

// ExtractHostPort 从 Docker ps 的 Ports 字段中提取指定容器端口对应的宿主机端口。
func ExtractHostPort(ports string, containerPort string) string {
	matches := hostPortPattern.FindAllStringSubmatch(ports, -1)
	for _, m := range matches {
		if len(m) >= 4 && m[3] == containerPort {
			return m[2]
		}
	}
	return ""
}

// ExtractHostname 从 HTTP Host header 提取主机名（不含端口）。
func ExtractHostname(host string) string {
	if host == "" {
		return ""
	}
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		if !strings.Contains(host[idx:], "]") {
			host = host[:idx]
		}
	}
	return host
}
