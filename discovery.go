package regsrv

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/yxlib/yx"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

//服务发现
type ServiceDiscovery struct {
	cli        *EtcdClient // etcd 客户端
	serverList sync.Map    // 服务器列表
	//prefix     string      // 监视的前缀

	balance *WeightRoundRobinBalance // 加权负载均衡
	logger  *yx.Logger
}

func NewServiceDiscovery(baseCfgPath string) *ServiceDiscovery {
	cli := NewEtcdCli()
	err := cli.Conn(baseCfgPath)
	if err != nil {
		panic(err)
	}

	return &ServiceDiscovery{
		cli:     cli,
		balance: NewWeightBalance(),
		logger:  yx.NewLogger("ServiceDiscovery"),
	}
}

//WatchService 初始化服务列表和监视
func (s *ServiceDiscovery) WatchService(prefix string) error {
	//根据前缀获取key/value
	mapKey2Value, err := s.cli.Get(prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for k, v := range mapKey2Value {
		s.SetServiceList(k, v)
	}

	//监视前缀，修改变更的server
	go s.watcher(prefix)
	return nil
}

//watcher 监听前缀
func (s *ServiceDiscovery) watcher(prefix string) {
	rch := s.cli.Watch(prefix, clientv3.WithPrefix())
	s.logger.I(fmt.Sprintf("watching prefix:%s now...", prefix))
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				s.DelServiceList(string(ev.Kv.Key))
			}
		}
	}
}

//GetServices 获取服务地址
func (s *ServiceDiscovery) GetServices() []Address {
	addrs := make([]Address, 0, 10)
	s.serverList.Range(func(k, v interface{}) bool {
		addrs = append(addrs, v.(Address))
		return true
	})
	return addrs
}

// 加权轮询获取服务的ip和端口
func (s *ServiceDiscovery) GetServiceIpAndPort() string {
	return s.balance.Next()
}

//SetServiceList 新增服务地址
func (s *ServiceDiscovery) SetServiceList(key, val string) {
	// 解析val
	node := NewSrvNode()
	json.Unmarshal([]byte(val), node)

	//获取服务地址
	addr := Address{Addr: node.Ip + ":" + node.Port}
	//获取服务地址权重
	nodeWeight := node.Weight
	if nodeWeight == 0 {
		// = 0默认权重为1
		nodeWeight = 1
	}

	//把服务地址权重存储到resolver.Address的元数据中
	addr.Attributes = NewAttributes(key, nodeWeight)
	s.serverList.Store(key, addr)
	s.balance.UpdateBalance(s.serverList)

	s.logger.I("put key: ", key, "val: ", val)
}

//DelServiceList 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.serverList.Delete(key)
	s.balance.UpdateBalance(s.serverList)

	s.logger.I("del key: ", key)
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}
