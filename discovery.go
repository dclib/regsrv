package regsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/yxlib/yx"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	SRV_TYPE_TCP       = 0
	SRV_TYPE_WEBSOCKET = 1
)

//服务发现
type ServiceDiscovery struct {
	cli           *EtcdClient              // etcd 客户端
	tcpSrvBalance *WeightRoundRobinBalance //加权负载均衡
	webSrvBalance *WeightRoundRobinBalance
	logger        *yx.Logger
}

func NewServiceDiscovery(baseCfgPath string) *ServiceDiscovery {
	cli := NewEtcdCli()
	err := cli.Conn(baseCfgPath)
	if err != nil {
		panic(err)
	}

	return &ServiceDiscovery{
		cli:           cli,
		tcpSrvBalance: NewWeightBalance(),
		webSrvBalance: NewWeightBalance(),
		logger:        yx.NewLogger("ServiceDiscovery"),
	}
}

//WatchService 初始化服务列表和监视
func (s *ServiceDiscovery) WatchService() error {
	//根据前缀获取key/value
	for _, prefix := range s.cli.Cfg.Watcher {
		mapKey2Value, err := s.cli.Get(prefix, clientv3.WithPrefix())
		if err != nil {
			return err
		}

		for k, v := range mapKey2Value {
			s.SetServiceList(k, v)
		}

		//监视前缀，修改变更的server
		go s.watcher(prefix)
	}

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

// 加权轮询获取服务的ip和端口
func (s *ServiceDiscovery) GetServiceIpAndPort(srvType uint16) string {
	if srvType == SRV_TYPE_TCP {
		return s.tcpSrvBalance.Next()
	}

	return s.webSrvBalance.Next()
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

	if node.SrvType == SRV_TYPE_TCP {
		s.tcpSrvBalance.serverList.Store(key, addr)
		s.tcpSrvBalance.UpdateBalance()
	} else if node.SrvType == SRV_TYPE_WEBSOCKET {
		s.webSrvBalance.serverList.Store(key, addr)
		s.webSrvBalance.UpdateBalance()
	}

	s.logger.I("put key: ", key, "val: ", val)
}

// 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	_, ok := s.tcpSrvBalance.serverList.Load(key)
	if ok {
		s.tcpSrvBalance.serverList.Delete(key)
		s.tcpSrvBalance.UpdateBalance()
	}

	_, ok = s.webSrvBalance.serverList.Load(key)
	if ok {
		s.webSrvBalance.serverList.Delete(key)
		s.webSrvBalance.UpdateBalance()
	}

	s.logger.I("del key: ", key)
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}

// watch
type Watch struct {
	revision      int64
	cancel        context.CancelFunc   //控制 watcher 退出
	eventChan     chan *clientv3.Event // 返回给上层的数据channel
	eventChanSize int
	lock          *sync.RWMutex
	logger        *yx.Logger

	incipientKVs []*mvccpb.KeyValue
}

func (s *ServiceDiscovery) WatchPrefix(ctx context.Context, prefix string) (*Watch, error) {
	resp, err := s.cli.cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var w = &Watch{
		eventChanSize: 64,
		revision:      resp.Header.Revision,
		eventChan:     make(chan *clientv3.Event, 64),
		incipientKVs:  resp.Kvs,
	}

	go func() {
		ctx, cancel := context.WithCancel(context.Background())

		// 给外部的cancel 方法,用于取消下面的watch
		w.cancel = cancel

		rch := s.cli.cli.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithCreatedNotify(), clientv3.WithRev(w.revision))
		for {
			for n := range rch {
				// 一般情况下，协程的逻辑会阻塞在此
				if n.CompactRevision > w.revision {
					w.revision = n.CompactRevision
				}

				//是否需要更新当前最新的revision
				if n.Header.GetRevision() > w.revision {
					w.revision = n.Header.GetRevision()
				}

				if err := n.Err(); err != nil {
					s.logger.E(fmt.Sprintf("WatchPrefix %s,%v", prefix, err))
					continue
				}

				for _, ev := range n.Events {
					select {
					// 将事件event 通过eventChan 通知上层
					case w.eventChan <- ev:
					default:
						s.logger.E("watch etcd with prefix block event chan, drop event message")
					}
				}
			}

			//当 watch() 被上层取消时.逻辑会走到此
			ctx, cancel := context.WithCancel(context.Background())
			w.cancel = cancel
			if w.revision > 0 {
				// 如果 revision 非 0，那么使用 WithRev 从 revision 的位置开始监听好了
				rch = s.cli.cli.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithCreatedNotify(), clientv3.WithRev(w.revision))
			} else {
				rch = s.cli.cli.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithCreatedNotify())
			}
		}
	}()

	return w, nil
}
