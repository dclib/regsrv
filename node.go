package regsrv

// 服务节点信息
type SrvNode struct {
	Ip      string `json:"ip"`       // ip
	Port    string `json:"port"`     // 端口
	Weight  int    `json:"weight"`   // 权重 可调节权重
	SrvType uint16 `json:"srv_type"` // 服务器类型 0:默认tcp 1:websocket
}

func NewSrvNode() *SrvNode {
	return &SrvNode{}
}
