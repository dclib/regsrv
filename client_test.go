package regsrv

import (
	"fmt"
	"testing"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestGameSrv(t *testing.T) {
	client := NewEtcdCli()
	err := client.Conn("")
	if err != nil {
		return
	}

	client.Put("/gamesrv/v1", "8.134.99.107:80")
	client.Put("/gamesrv/v2", "8.134.99.107:8081")
	client.Put("/gamesrv/v3", "8.134.99.107:8082")

	kv, err := client.Get("/gamesrv/", clientv3.WithPrefix())
	if err != nil {
		return
	}

	for k, v := range kv {
		fmt.Println(k, " ", v)
	}
}
