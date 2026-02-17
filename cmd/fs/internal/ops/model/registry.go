package model

// ─── etcd 服务注册表 ───

// EtcdServiceInstance 描述一个在 etcd 中注册的服务实例。
type EtcdServiceInstance struct {
	Key  string `json:"key"`  // etcd 完整 Key，如 "seckill.rpc/7587869c5e217"
	Addr string `json:"addr"` // 注册地址，如 "172.18.0.11:8086"
}

// EtcdService 描述一个服务的所有注册实例。
type EtcdService struct {
	ServiceKey string                `json:"service_key"` // go-zero 注册 Key，如 "seckill.rpc"
	Instances  []EtcdServiceInstance `json:"instances"`
}

// EtcdRegistrySnapshot 聚合所有已知服务的 etcd 注册表快照。
type EtcdRegistrySnapshot struct {
	Available bool          `json:"available"` // etcd 是否可用
	Endpoint  string        `json:"endpoint"`  // 连接的 etcd 地址
	Services  []EtcdService `json:"services"`
	Error     string        `json:"error,omitempty"`
}
