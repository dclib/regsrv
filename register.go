package regsrv

import (
	"github.com/yxlib/yx"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const TTL = 60 // 租约时长 单位 s 则续租为 1/3 10s

// 服务注册
type ServiceRegister struct {
	etcdCli *EtcdClient // etcd 客户端
	LeaseId int64       // 租约id
	Stop    bool        // 停止
	logger  *yx.Logger
}

func NewServiceRegister(baseCfgPath string) *ServiceRegister {
	etcdCli := NewEtcdCli()
	err := etcdCli.Conn(baseCfgPath)
	if err != nil {
		panic(err)
	}

	return &ServiceRegister{
		etcdCli: etcdCli,
		logger:  yx.NewLogger("ServiceRegister"),
	}
}

func (k *ServiceRegister) LeaseAndKeepAlive(key, val string) error {
	leaseId, err := k.etcdCli.Lease(key, val, TTL)
	if err != nil {
		return err
	}

	//设置续租 定期发送需求请求
	ch, err := k.etcdCli.KeepAlive(leaseId)
	if err != nil {
		return err
	}

	k.LeaseId = leaseId

	go k.readResponse(ch)
	return nil
}

func (k *ServiceRegister) readResponse(ch <-chan *clientv3.LeaseKeepAliveResponse) {
	for {
		<-ch
		// 读出来清空,不然ch会满
	}
}

func (k *ServiceRegister) Close() {
	// 废除租约
	err := k.etcdCli.Revoke(k.LeaseId)
	if err != nil {
		panic(err)
	}

	k.etcdCli.Close()
}
