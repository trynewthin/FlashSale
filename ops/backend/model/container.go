package model

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
	Optional        bool     `json:"optional"`
	Absent          bool     `json:"absent"`
	Replicas        int      `json:"replicas"`
	RunningReplicas int      `json:"running_replicas"`
	DependsOn       []string `json:"depends_on"`
}

func NewServiceRuntime(name, role string, scalable bool) *ServiceRuntime {
	return &ServiceRuntime{Name: name, Role: role, Scalable: scalable, DependsOn: []string{}}
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

type ServiceDef struct {
	Name      string
	Role      string
	Scalable  bool
	Optional  bool
	DependsOn []string
}
