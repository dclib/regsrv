package regsrv

type AddressType uint8

const (
	// Backend indicates the address is for a backend server.
	//
	// Deprecated: use Attributes in Address instead.
	Backend AddressType = iota
	// GRPCLB indicates the address is for a grpclb load balancer.
	//
	// Deprecated: use Attributes in Address instead.
	GRPCLB
)

type Address struct {
	// Addr is the server address on which a connection will be established.
	Addr string

	ServerName string

	Attributes *Attributes

	Type AddressType

	Metadata interface{}
}
