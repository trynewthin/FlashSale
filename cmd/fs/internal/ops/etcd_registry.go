// etcd_registry 提供 etcd 服务注册表查询能力。
// 使用 Go etcd v3 client 直连 etcd，按 go-zero 注册 Key 解析实例列表。
package ops

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

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

// knownServiceKeys 定义所有已知的 go-zero RPC 注册 Key。
// 与 apps/*/rpc/etc/*.docker.yaml 中 Etcd.Key 保持一致。
var knownServiceKeys = []string{
	"user.rpc",
	"admin.rpc",
	"product.rpc",
	"order.rpc",
	"seckill.rpc",
}

// defaultEtcdEndpoint 返回默认的 etcd 连接地址。
// 优先使用 FLASHSALE_ETCD_ENDPOINT 环境变量覆盖。
func defaultEtcdEndpoint() string {
	if ep := strings.TrimSpace(os.Getenv("FLASHSALE_ETCD_ENDPOINT")); ep != "" {
		return ep
	}
	return "localhost:2379"
}

// ---------- etcd client 单例 ----------

var (
	etcdMu       sync.Mutex
	etcdClient   *clientv3.Client
	etcdEndpoint string // 上次创建 client 时的 endpoint，变化时重建
)

// getOrCreateEtcdClient 返回单例 etcd client，endpoint 变化时自动重建。
func getOrCreateEtcdClient(endpoint string) (*clientv3.Client, error) {
	etcdMu.Lock()
	defer etcdMu.Unlock()

	if etcdClient != nil && etcdEndpoint == endpoint {
		return etcdClient, nil
	}
	// endpoint 变化或首次创建，关闭旧连接。
	if etcdClient != nil {
		_ = etcdClient.Close()
		etcdClient = nil
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	etcdClient = cli
	etcdEndpoint = endpoint
	return cli, nil
}

func (s *Server) getEtcdServices(w http.ResponseWriter, _ *http.Request) {
	snap := queryEtcdRegistry()
	writeOK(w, map[string]any{"registry": snap})
}

func queryEtcdRegistry() EtcdRegistrySnapshot {
	endpoint := defaultEtcdEndpoint()
	snap := EtcdRegistrySnapshot{
		Endpoint: endpoint,
		Services: make([]EtcdService, 0, len(knownServiceKeys)),
	}

	cli, err := getOrCreateEtcdClient(endpoint)
	if err != nil {
		snap.Error = fmt.Sprintf("etcd connect failed: %v", err)
		return snap
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 验证可连接性。
	_, err = cli.Status(ctx, endpoint)
	if err != nil {
		snap.Error = fmt.Sprintf("etcd unreachable: %v", err)
		return snap
	}
	snap.Available = true

	for _, key := range knownServiceKeys {
		svc := EtcdService{
			ServiceKey: key,
			Instances:  make([]EtcdServiceInstance, 0),
		}

		prefix := key + "/"
		resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
		if err != nil {
			// 单个 key 失败不阻塞其他服务。
			continue
		}

		for _, kv := range resp.Kvs {
			svc.Instances = append(svc.Instances, EtcdServiceInstance{
				Key:  string(kv.Key),
				Addr: string(kv.Value),
			})
		}

		// 按地址排序，保证结果稳定。
		sort.Slice(svc.Instances, func(i, j int) bool {
			return svc.Instances[i].Addr < svc.Instances[j].Addr
		})

		snap.Services = append(snap.Services, svc)
	}

	return snap
}
