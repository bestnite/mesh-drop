package discovery

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"mesh-drop/internal/config"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	DiscoveryPort = 9988
	HeartbeatRate = 3 * time.Second
	PeerTimeout   = 10 * time.Second
)

type Service struct {
	app *application.App

	ID             string
	config         *config.Config
	FileServerPort int

	// key 使用 peer.id 和 peer.ip 组合而成的 hash
	peers      map[string]*Peer
	peersMutex sync.RWMutex
}

func init() {
	application.RegisterEvent[[]Peer]("peers:update")
}

func NewService(config *config.Config, app *application.App, port int) *Service {
	return &Service{
		app:            app,
		ID:             config.GetID(),
		config:         config,
		FileServerPort: port,
		peers:          make(map[string]*Peer),
	}
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
			ID:   s.ID,
			Name: s.config.GetHostName(),
			Port: s.FileServerPort,
			OS:   OS(runtime.GOOS),
		}
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

		s.handleHeartbeat(packet, remoteAddr.IP.String())
	}
}

// handleHeartbeat 处理心跳包
func (s *Service) handleHeartbeat(pkt PresencePacket, ip string) {
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
			Port: pkt.Port,
			OS:   pkt.OS,
		}
		s.peers[peer.ID] = peer
		slog.Info("New device found", "name", pkt.Name, "ip", ip, "component", "discovery")
	} else {
		// 更新节点
		peer.Name = pkt.Name
		peer.OS = pkt.OS
		peer.Routes[ip] = &RouteState{
			IP:       ip,
			LastSeen: time.Now(),
		}
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
				// 超过10秒没心跳，认为下线
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

func (s *Service) GetPeerByIP(ip string) *Peer {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	for _, p := range s.peers {
		if p.Routes[ip] != nil {
			return p
		}
	}
	return nil
}

func (s *Service) GetPeers() []Peer {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	list := make([]Peer, 0)
	for _, p := range s.peers {
		list = append(list, *p)
	}
	return list
}

func (s *Service) GetID() string {
	return s.ID
}
