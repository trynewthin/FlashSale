package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"flashsale/cmd/fs/internal/ops/model"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// knownServiceKeys 定义所有已知的 go-zero RPC 注册 Key。
var knownServiceKeys = []string{
	"user.rpc",
	"admin.rpc",
	"product.rpc",
	"order.rpc",
	"seckill.rpc",
}

// ---------- etcd client 单例 ----------

var (
	etcdMu       sync.Mutex
	etcdClient   *clientv3.Client
	etcdEndpoint string
)

func getOrCreateEtcdClient(endpoint string) (*clientv3.Client, error) {
	etcdMu.Lock()
	defer etcdMu.Unlock()

	if etcdClient != nil && etcdEndpoint == endpoint {
		return etcdClient, nil
	}
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

// QueryEtcdRegistry 查询 etcd 注册表快照。
func QueryEtcdRegistry(env *EnvContext) model.EtcdRegistrySnapshot {
	endpoint := env.EtcdEndpoint
	snap := model.EtcdRegistrySnapshot{
		Endpoint: endpoint,
		Services: make([]model.EtcdService, 0, len(knownServiceKeys)),
	}

	cli, err := getOrCreateEtcdClient(endpoint)
	if err != nil {
		snap.Error = fmt.Sprintf("etcd connect failed: %v", err)
		return snap
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Status(ctx, endpoint)
	if err != nil {
		snap.Error = fmt.Sprintf("etcd unreachable: %v", err)
		return snap
	}
	snap.Available = true

	for _, key := range knownServiceKeys {
		svc := model.EtcdService{
			ServiceKey: key,
			Instances:  make([]model.EtcdServiceInstance, 0),
		}

		prefix := key + "/"
		resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
		if err != nil {
			continue
		}

		for _, kv := range resp.Kvs {
			svc.Instances = append(svc.Instances, model.EtcdServiceInstance{
				Key:  string(kv.Key),
				Addr: string(kv.Value),
			})
		}

		sort.Slice(svc.Instances, func(i, j int) bool {
			return svc.Instances[i].Addr < svc.Instances[j].Addr
		})

		snap.Services = append(snap.Services, svc)
	}

	return snap
}
