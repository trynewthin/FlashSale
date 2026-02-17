// observability 提供可观测性工具入口链接推断。
// 后端根据容器端口映射和运行状态推断 Jaeger/Prometheus/Grafana 的访问 URL，
// 前端只渲染，不拼接环境差异。
package ops

import (
	"net/http"
	"os"
	"regexp"
	"strings"
)

// ObservabilityLink 描述单个可观测性工具的入口。
type ObservabilityLink struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Available bool   `json:"available"`
}

// ObservabilityLinks 聚合所有可观测性工具入口。
type ObservabilityLinks struct {
	Jaeger     ObservabilityLink `json:"jaeger"`
	Prometheus ObservabilityLink `json:"prometheus"`
	Grafana    ObservabilityLink `json:"grafana"`
}

// observabilityToolDef 定义工具名称、对应容器服务名、默认容器端口和协议。
type observabilityToolDef struct {
	Name          string
	ServiceName   string
	ContainerPort string // 期望暴露的容器端口（如 "16686"）
	Protocol      string // "http"
	EnvOverride   string // 环境变量覆盖 key（如 "FLASHSALE_JAEGER_URL"）
}

var observabilityTools = []observabilityToolDef{
	{Name: "Jaeger", ServiceName: "jaeger", ContainerPort: "16686", Protocol: "http", EnvOverride: "FLASHSALE_JAEGER_URL"},
	{Name: "Prometheus", ServiceName: "prometheus", ContainerPort: "9090", Protocol: "http", EnvOverride: "FLASHSALE_PROMETHEUS_URL"},
	{Name: "Grafana", ServiceName: "grafana", ContainerPort: "3000", Protocol: "http", EnvOverride: "FLASHSALE_GRAFANA_URL"},
}

// hostPortPattern 匹配 docker ps 输出中的 host:port->container_port 映射。
// 示例输入："0.0.0.0:16686->16686/tcp, :::16686->16686/tcp"
var hostPortPattern = regexp.MustCompile(`(?:(\d+\.\d+\.\d+\.\d+)|::):(\d+)->(\d+)`)

func (s *Server) getObservabilityLinks(w http.ResponseWriter, r *http.Request) {
	// 获取请求的 Host 来推断访问域名（运维面板自身的域名/IP）。
	reqHost := extractHostname(r)

	// 获取当前容器状态快照。
	containers, _ := listDockerContainers(s.runner.repoRoot)

	// 按服务名索引容器。
	containerByService := map[string]ContainerState{}
	for _, c := range containers {
		svc := strings.TrimSpace(c.Service)
		if svc == "" {
			continue
		}
		// 每个服务取第一个运行中的容器。
		if _, exists := containerByService[svc]; !exists || c.Running {
			containerByService[svc] = c
		}
	}

	links := ObservabilityLinks{
		Jaeger:     buildObservabilityLink(observabilityTools[0], containerByService, reqHost),
		Prometheus: buildObservabilityLink(observabilityTools[1], containerByService, reqHost),
		Grafana:    buildObservabilityLink(observabilityTools[2], containerByService, reqHost),
	}
	writeOK(w, map[string]any{"links": links})
}

func buildObservabilityLink(tool observabilityToolDef, containers map[string]ContainerState, reqHost string) ObservabilityLink {
	link := ObservabilityLink{Name: tool.Name}

	// 1. 环境变量优先覆盖。
	if envURL := strings.TrimSpace(os.Getenv(tool.EnvOverride)); envURL != "" {
		link.URL = envURL
		// 有环境变量时直接视为可用（运维人员手动指定）。
		link.Available = true
		return link
	}

	// 2. 从容器端口映射推断。
	c, ok := containers[tool.ServiceName]
	if !ok {
		return link // 容器不存在，URL 为空，Available 为 false。
	}

	link.Available = c.Running

	hostPort := extractHostPort(c.Ports, tool.ContainerPort)
	if hostPort == "" {
		// 容器存在但没有 host port 映射（仅内部网络），使用默认端口。
		hostPort = tool.ContainerPort
	}

	// 用请求来源的 hostname（不带端口）拼接推断的端口。
	host := reqHost
	if host == "" {
		host = "localhost"
	}
	link.URL = tool.Protocol + "://" + host + ":" + hostPort

	return link
}

// extractHostPort 从 Docker ps 的 Ports 字段中提取指定容器端口对应的宿主机端口。
// ports 格式示例："0.0.0.0:16686->16686/tcp, :::16686->16686/tcp"
func extractHostPort(ports string, containerPort string) string {
	matches := hostPortPattern.FindAllStringSubmatch(ports, -1)
	for _, m := range matches {
		if len(m) >= 4 && m[3] == containerPort {
			// m[1] 是 IP（如 "0.0.0.0"），m[2] 是 host port。
			return m[2]
		}
	}
	return ""
}

// extractHostname 从 HTTP 请求中提取主机名（不含端口）。
func extractHostname(r *http.Request) string {
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	if host == "" {
		return ""
	}
	// 去掉端口部分。
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		// 防止误截 IPv6 地址（[::1]:8080）。
		if !strings.Contains(host[idx:], "]") {
			host = host[:idx]
		}
	}
	return host
}
