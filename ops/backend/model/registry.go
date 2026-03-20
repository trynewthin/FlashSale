package model

type EtcdServiceInstance struct {
	Key  string `json:"key"`
	Addr string `json:"addr"`
}

type EtcdService struct {
	ServiceKey string                `json:"service_key"`
	Instances  []EtcdServiceInstance `json:"instances"`
}

type EtcdRegistrySnapshot struct {
	Available bool          `json:"available"`
	Endpoint  string        `json:"endpoint"`
	Services  []EtcdService `json:"services"`
	Error     string        `json:"error,omitempty"`
}
