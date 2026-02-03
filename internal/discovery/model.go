package discovery

import "time"

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

	// IsOnline 标记该端点当前是否活跃 (UI 渲染用)。
	IsOnline bool `json:"is_online"`

	OS OS `json:"os"`
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
	ID   string `json:"id"`
	Name string `json:"name"`
	Port int    `json:"port"`
	OS   OS     `json:"os"`
}
