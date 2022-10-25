package regsrv

import (
	"time"

	"github.com/yxlib/yx"
)

const TTL = 30 // 租约时长 单位 s

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
	err = k.etcdCli.KeepAlive(leaseId)
	if err != nil {
		return err
	}

	k.LeaseId = leaseId
	//go k.RenewContract()
	return nil
}

// 续租
func (k *ServiceRegister) RenewContract() {
	ticker := time.NewTicker(time.Second * time.Duration(TTL-5))

	for {
		<-ticker.C

		err := k.etcdCli.KeepAlive(k.LeaseId)
		k.logger.D("keepAlive id: ", k.LeaseId)
		if err != nil {
			k.logger.E("renewContract err", err)
			break
		}

		if k.Stop {
			break
		}
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
