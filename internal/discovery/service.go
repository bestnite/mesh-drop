package discovery

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"mesh-drop/internal/config"
	"mesh-drop/internal/security"
)

const (
	DiscoveryPort = 9988
	HeartbeatRate = 1 * time.Second
	PeerTimeout   = 2 * time.Second
)

type Service struct {
	app *application.App

	ID             string
	config         *config.Config
	FileServerPort int

	// Key: peer.ID
	peers      map[string]*Peer
	peersMutex sync.RWMutex

	self Peer
}

func NewService(config *config.Config, app *application.App, port int) *Service {
	return &Service{
		app:            app,
		ID:             config.GetID(),
		config:         config,
		FileServerPort: port,
		peers:          make(map[string]*Peer),
		self: Peer{
			ID:        config.GetID(),
			Name:      config.GetHostName(),
			Port:      port,
			OS:        OS(runtime.GOOS),
			PublicKey: config.GetPublicKey(),
		},
	}
}

func GetLocalIPs() ([]string, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		slog.Error("Failed to get network interfaces", "error", err, "component", "discovery")
		return nil, false
	}
	var ips []string
	for _, iface := range interfaces {
		// 过滤掉 Down 的接口和 Loopback 接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// 获取该接口的地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() == nil {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	return ips, true
}

func (s *Service) GetLocalIPInSameSubnet(receiverIP string) (string, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		slog.Error("Failed to get network interfaces", "error", err, "component", "discovery")
		return "", false
	}
	for _, iface := range interfaces {
		// 过滤掉 Down 的接口和 Loopback 接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// 获取该接口的地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, ipNet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() == nil {
				continue
			}
			if ipNet.Contains(net.ParseIP(receiverIP)) {
				return ip.String(), true
			}
		}
	}
	slog.Error(
		"Failed to get local IP in same subnet",
		"receiverIP",
		receiverIP,
		"component",
		"discovery",
	)
	return "", false
}

func (s *Service) startBroadcasting() {
	ticker := time.NewTicker(HeartbeatRate)
	for range ticker.C {
		interfaces, err := net.Interfaces()
		if err != nil {
			slog.Error("Failed to get network interfaces", "error", err, "component", "discovery")
			continue
		}
		packet := PresencePacket{
			ID:        s.ID,
			Name:      s.config.GetHostName(),
			Port:      s.FileServerPort,
			OS:        OS(runtime.GOOS),
			PublicKey: s.config.GetPublicKey(),
		}

		// 签名
		sigData := packet.SignPayload()
		sig, err := security.Sign(s.config.GetPrivateKey(), sigData)
		if err != nil {
			slog.Error("Failed to sign discovery packet", "error", err)
			continue
		}
		packet.Signature = sig

		data, _ := json.Marshal(packet)
		for _, iface := range interfaces {
			// 过滤掉 Down 的接口和 Loopback 接口
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			// 获取该接口的地址
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ip, ipNet, err := net.ParseCIDR(addr.String())
				if err != nil {
					continue
				}
				if ip.To4() == nil {
					continue
				}
				// 计算该网段的广播地址
				// 例如 IP: 192.168.1.5/24 -> 广播地址: 192.168.1.255
				broadcastIPV4 := make(net.IP, len(ip.To4()))
				copy(broadcastIPV4, ip.To4())
				for i, b := range ipNet.Mask {
					broadcastIPV4[i] |= ^b
				}
				slog.Debug("Broadcast IP", "ip", broadcastIPV4.String(), "component", "discovery")
				s.sendPacketTo(broadcastIPV4.String(), DiscoveryPort, data)
			}
		}
	}
}

func (s *Service) sendPacketTo(ip string, port int, data []byte) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()
	_, err = conn.Write(data)
	if err != nil {
		slog.Error("Failed to send packet", "error", err, "component", "discovery")
		return
	}
}

func (s *Service) startListening() {
	addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DiscoveryPort))
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		slog.Error("Failed to start listening", "error", err, "component", "discovery")
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		var packet PresencePacket
		if err := json.Unmarshal(buf[:n], &packet); err != nil {
			continue
		}

		// 忽略自己发出的包
		if packet.ID == s.ID {
			continue
		}

		// 验证签名
		sig := packet.Signature
		sigData := packet.SignPayload()
		valid, err := security.Verify(packet.PublicKey, sigData, sig)
		if err != nil || !valid {
			slog.Warn(
				"Received invalid discovery packet signature",
				"id",
				packet.ID,
				"ip",
				remoteAddr.IP.String(),
			)
			continue
		}

		// 验证身份一致性 (防止 ID 欺骗)
		trustMismatch := false
		trustedKeys := s.config.GetTrusted()
		if knownKey, ok := trustedKeys[packet.ID]; ok {
			if knownKey != packet.PublicKey {
				slog.Warn(
					"SECURITY ALERT: Peer ID mismatch with known public key (Spoofing attempt?)",
					"id",
					packet.ID,
					"known_key",
					knownKey,
					"received_key",
					packet.PublicKey,
				)
				trustMismatch = true
				// 当发现 ID 欺骗时，不更新 peer，而是标记为 trustMismatch
				// 用户可以手动重新添加信任
			}
		} else {
			// 不存在于信任列表
			// 存在之前在信任列表，但是不匹配被用户手动重置了，此时需要将 peer.TrustMismatch 标记为 false
			// 否则在 handleHeartbeat 里会一直标记为不匹配
			if peer, ok := s.peers[packet.ID]; ok {
				peer.TrustMismatch = false
			}
		}

		s.handleHeartbeat(packet, remoteAddr.IP.String(), trustMismatch)
	}
}

// handleHeartbeat 处理心跳包
func (s *Service) handleHeartbeat(pkt PresencePacket, ip string, trustMismatch bool) {
	s.peersMutex.Lock()

	peer, exists := s.peers[pkt.ID]
	if !exists {
		// 发现新节点
		peer = &Peer{
			ID:   pkt.ID,
			Name: pkt.Name,
			Routes: map[string]*RouteState{
				ip: {
					IP:       ip,
					LastSeen: time.Now(),
				},
			},
			Port:          pkt.Port,
			OS:            pkt.OS,
			PublicKey:     pkt.PublicKey,
			TrustMismatch: trustMismatch,
		}
		s.peers[peer.ID] = peer
		slog.Info("New device found", "name", pkt.Name, "ip", ip, "component", "discovery")
	} else {
		// 更新节点
		// 只有在没有身份不匹配的情况下才更新元数据，防止欺骗攻击导致 UI 闪烁/篡改
		if !trustMismatch {
			peer.Name = pkt.Name
			peer.OS = pkt.OS
			peer.PublicKey = pkt.PublicKey
		}
		peer.Routes[ip] = &RouteState{
			IP:       ip,
			LastSeen: time.Now(),
		}
		// 如果之前存在不匹配，即使这次匹配了，也不要重置，防止欺骗攻击
		peer.TrustMismatch = peer.TrustMismatch || trustMismatch
	}

	s.peersMutex.Unlock()

	// 触发前端更新 (防抖逻辑可以之后加，这里每次变动都推)
	s.app.Event.Emit("peers:update", s.GetPeers())
}

// 3. 掉线清理协程
func (s *Service) startCleanup() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		s.peersMutex.Lock()
		changed := false
		now := time.Now()

		for id, peer := range s.peers {
			for ip, route := range peer.Routes {
				if now.Sub(route.LastSeen) > PeerTimeout {
					delete(peer.Routes, ip)
					changed = true
					slog.Info("Device offline", "name", peer.Name, "component", "discovery")
				}
			}

			if len(peer.Routes) == 0 {
				delete(s.peers, id)
				changed = true
				slog.Info("Device offline", "name", peer.Name, "component", "discovery")
			}
		}
		s.peersMutex.Unlock()

		if changed {
			s.app.Event.Emit("peers:update", s.GetPeers())
		}
	}
}

func (s *Service) Start() {
	go s.startBroadcasting()
	go s.startListening()
	go s.startCleanup()
}

func (s *Service) GetPeerByIP(ip string) (*Peer, bool) {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	for _, p := range s.peers {
		if p.Routes[ip] != nil {
			return p.DeepCopy(), true
		}
	}
	return nil, false
}

func (s *Service) GetPeerByID(id string) (*Peer, bool) {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	peer, ok := s.peers[id]
	if !ok {
		return nil, false
	}
	return peer.DeepCopy(), true
}

func (s *Service) GetPeers() []Peer {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	list := make([]Peer, 0)
	for _, p := range s.peers {
		list = append(list, *p.DeepCopy())
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func (s *Service) GetID() string {
	return s.ID
}

func (s *Service) GetSelf() Peer {
	return s.self
}
