package regsrv

import (
	"fmt"
	"testing"
)

func TestWeightServerRoundRobin(t *testing.T) {
	balance := NewWeightBalance()
	balance.BuildBalance("127.0.0.1:80", "10")
	balance.BuildBalance("127.0.0.1:81", "20")
	balance.BuildBalance("127.0.0.1:82", "30")
	balance.BuildBalance("127.0.0.1:83", "40")
	balance.BuildBalance("127.0.0.1:84", "40")

	for i := 0; i < 20; i++ {
		fmt.Println(balance.Next())
	}
}

func BenchmarkWightNext(b *testing.B) {
	b.ReportAllocs()
	balance := NewWeightBalance()
	balance.BuildBalance("127.0.0.1:80", "10")
	balance.BuildBalance("127.0.0.1:81", "20")
	balance.BuildBalance("127.0.0.1:82", "30")
	balance.BuildBalance("127.0.0.1:83", "40")
	balance.BuildBalance("127.0.0.1:84", "40")

	for i := 0; i < b.N; i++ {
		balance.Next()
	}
}
