package regsrv

// Name is the name of weight balancer.
const Name = "weight"

var (
	minWeight = 1
	maxWeight = 5
)

// attributeKey is the type used as the key to store AddrInfo in the Attributes
// field of resolver.Address.
type attributeKey struct{}

// AddrInfo will be stored inside Address metadata in order to use weighted balancer.
type AddrInfo struct {
	Weight int
}

func SetAddrInfo(addr Address, addrInfo AddrInfo) Address {
	addr.Attributes = NewAttributes()
	addr.Attributes = addr.Attributes.WithValues(attributeKey{}, addrInfo)
	return addr
}

// GetAddrInfo returns the AddrInfo stored in the Attributes fields of addr.
func GetAddrInfo(addr Address) AddrInfo {
	v := addr.Attributes.Value(attributeKey{})
	ai, _ := v.(AddrInfo)
	return ai
}
