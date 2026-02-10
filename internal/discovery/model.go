package discovery

import (
	"fmt"
	"time"
)

// Peer 代表一个可达的网络端点 (Network Endpoint)。
// 注意：一个物理设备 (Device) 可能通过多个网络接口广播，因此会对应多个 Peer 结构体。
type Peer struct {
	// ID 是物理设备的全局唯一标识 (UUID/MachineID)。
	// 具有相同 ID 的 Peer 属于同一台物理设备。
	ID string `json:"id"`

	// Name 是设备的主机名或用户设置的显示名称 (如 "Nite's Arch")。
	Name string `json:"name"`

	// Routes 记录了设备的 IP 地址和状态。
	// Key: ip, Value: *RouteState
	Routes map[string]*RouteState `json:"routes"`

	// Port 是文件传输服务的监听端口。
	Port int `json:"port"`

	OS OS `json:"os"`

	PublicKey string `json:"pk"`

	// TrustMismatch 指示该节点的公钥与本地信任列表中的公钥不匹配
	// 如果为 true，说明可能存在 ID 欺骗或密钥轮换
	TrustMismatch bool `json:"trust_mismatch"`
}

// RouteState 记录单条路径的状态
type RouteState struct {
	IP       string    `json:"ip"`
	LastSeen time.Time `json:"last_seen"` // 该特定 IP 最后一次响应的时间
}

type OS string

const (
	OSLinux   OS = "linux"
	OSWindows OS = "windows"
	OSMac     OS = "darwin"
)

// PresencePacket 是 UDP 广播的载荷
type PresencePacket struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Port      int    `json:"port"`
	OS        OS     `json:"os"`
	PublicKey string `json:"pk"`
	Signature string `json:"sig"`
}

// SignPayload 生成用于签名的确定性数据
func (p *PresencePacket) SignPayload() []byte {
	// 使用固定格式拼接字段，避免 JSON 序列化的不确定性
	// 格式: id|name|port|os|pk
	return fmt.Appendf(nil, "%s|%s|%d|%s|%s", p.ID, p.Name, p.Port, p.OS, p.PublicKey)
}

// DeepCopy 返回 Peer 的深拷贝
func (p Peer) DeepCopy() *Peer {
	newPeer := p // 结构体浅拷贝 (值类型字段已复制)

	// 手动深拷贝引用类型字段 (Routes)
	if p.Routes != nil {
		newPeer.Routes = make(map[string]*RouteState, len(p.Routes))
		for k, v := range p.Routes {
			// RouteState 只有值类型字段，但它是指针，所以需要新建对象并解引用赋值
			stateCopy := *v
			newPeer.Routes[k] = &stateCopy
		}
	}

	return &newPeer
}
