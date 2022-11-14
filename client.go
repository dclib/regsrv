package regsrv

import (
	"context"
	"time"

	"github.com/yxlib/yx"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const BaseCfgPath = "deploy/reg_etcd.json"

type Config struct {
	Endpoints      []string `json:"endpoints"`
	DialTimeoutSec int      `json:"dts"`
	EtcdUser       string   `json:"etcdUser"`
	EtcdPass       string   `json:"etcdPass"`
	Watcher        []string `json:"watcher"`
}

type EtcdClient struct {
	Cfg    *Config `json:"etcd"`
	cli    *clientv3.Client
	logger yx.Logger
}

func NewEtcdCli() *EtcdClient {
	return &EtcdClient{
		Cfg:    &Config{},
		logger: *yx.NewLogger("EtcdClient"),
	}
}

func (e *EtcdClient) Conn(cfgPath string) error {
	// 创建客户端，连接etcd
	if cfgPath == "" {
		cfgPath = BaseCfgPath
	}

	err := yx.LoadJsonConf(e, cfgPath, nil)
	if err != nil {
		e.logger.E("load log config err: ", err)
		return err
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   e.Cfg.Endpoints,
		DialTimeout: time.Duration(e.Cfg.DialTimeoutSec) * time.Second,
		Username:    e.Cfg.EtcdUser,
		Password:    e.Cfg.EtcdPass,
	})

	if err != nil {
		e.logger.E("connect to etcd failed, err: ", err)
		return err
	}

	e.cli = cli
	return e.ConnWithParam(e.Cfg.Endpoints, e.Cfg.DialTimeoutSec, e.Cfg.EtcdUser, e.Cfg.EtcdPass)
}

func (e *EtcdClient) ConnWithParam(endpoints []string, dial int, userName string, pass string) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Duration(dial) * time.Second,
		Username:    userName,
		Password:    pass,
	})

	if err != nil {
		e.logger.E("connect to etcd failed, err: ", err)
		return err
	}

	e.cli = cli
	return nil
}

//put
func (e *EtcdClient) Put(key, val string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := e.cli.Put(ctx, key, val)
	cancel()
	return err
}

func (e *EtcdClient) PutWithOpts(key, val string, opts clientv3.OpOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := e.cli.Put(ctx, key, val, opts)
	cancel()
	return err
}

// get string
func (e *EtcdClient) Get(key string, opts clientv3.OpOption) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := e.cli.Get(ctx, key, opts)
	cancel()
	if err != nil {
		return nil, err
	}

	mapKeyValue := make(map[string]string, 0)
	for _, kv := range resp.Kvs {
		mapKeyValue[string(kv.Key)] = string(kv.Value)
	}

	return mapKeyValue, nil
}

// delete
func (e *EtcdClient) Delete(key string) error {
	// del 取数据
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := e.cli.Delete(ctx, key)
	cancel()
	if err != nil {
		e.logger.E("del from etcd failed, err ", err)
		return err
	}

	return err
}

//watch 监听某个key
func (e *EtcdClient) Watch(key string, opts clientv3.OpOption) clientv3.WatchChan {
	watchChan := e.cli.Watch(context.Background(), key, opts)
	return watchChan
}

// 创建租约 ttl duration 返回租约id
func (e *EtcdClient) Lease(key, val string, ttl int64) (int64, error) {
	resp, err := e.cli.Grant(context.TODO(), ttl)
	if err != nil {
		return 0, err
	}

	err = e.PutWithOpts(key, val, clientv3.WithLease(resp.ID))
	if err != nil {
		return 0, err
	}

	return int64(resp.ID), nil
}

// keepLive
func (e *EtcdClient) KeepAlive(leaseId int64) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	ch, err := e.cli.KeepAlive(context.Background(), clientv3.LeaseID(leaseId))
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// 废除租约
func (e *EtcdClient) Revoke(leaseId int64) error {
	_, err := e.cli.Revoke(context.Background(), clientv3.LeaseID(leaseId))
	return err
}

func (e *EtcdClient) Close() error {
	return e.cli.Close()
}
