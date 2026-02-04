package transfer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"mesh-drop/internal/config"
	"mesh-drop/internal/discovery"
	"mesh-drop/internal/security"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type Service struct {
	config   *config.Config
	notifier *notifications.NotificationService
	app      *application.App
	port     int

	// pendingRequests 存储等待用户确认的通道
	// Key: TransferID, Value: *Transfer
	transferList sync.Map

	discoveryService *discovery.Service

	// cancelMap 存储取消操作的通道
	// Key: TransferID, Value: context.CancelFunc
	cancelMap sync.Map

	httpClient *http.Client
}

func NewService(config *config.Config, app *application.App, notifier *notifications.NotificationService, port int, discoveryService *discovery.Service) *Service {
	gin.SetMode(gin.ReleaseMode)

	// 配置自定义 HTTP 客户端以跳过自签名证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{
		Transport: tr,
		Timeout:   0,
	}

	return &Service{
		app:              app,
		notifier:         notifier,
		port:             port,
		discoveryService: discoveryService,
		config:           config,
		httpClient:       httpClient,
	}
}

func init() {
	application.RegisterEvent[application.Void]("transfer:refreshList")
}

func (s *Service) GetPort() int {
	return s.port
}

func (s *Service) Start() {
	r := gin.Default()
	transfer := r.Group("/transfer")
	{
		transfer.POST("/ask", s.handleAsk)
		transfer.PUT("/upload/:id", s.handleUpload)
	}

	go func() {
		configDir := config.GetConfigDir()
		certPath := filepath.Join(configDir, "server.crt")
		keyPath := filepath.Join(configDir, "server.key")

		if err := security.EnsureCertificates(certPath, keyPath); err != nil {
			slog.Error("Failed to generate certificates", "error", err, "component", "transfer")
			return
		}

		addr := fmt.Sprintf(":%d", s.port)
		slog.Info("Transfer service listening (HTTPS)", "address", addr, "component", "transfer")

		if err := r.RunTLS(addr, certPath, keyPath); err != nil {
			slog.Error("Transfer service error", "error", err, "component", "transfer")
		}
	}()
}

func (s *Service) GetTransferList() []*Transfer {
	var requests []*Transfer = make([]*Transfer, 0)
	s.transferList.Range(func(key, value any) bool {
		requests = append(requests, value.(*Transfer))
		return true
	})
	return requests
}

func (s *Service) GetTransfer(transferID string) (*Transfer, bool) {
	val, ok := s.transferList.Load(transferID)
	if !ok {
		return nil, false
	}
	return val.(*Transfer), true
}

func (s *Service) CancelTransfer(transferID string) {
	if cancel, ok := s.cancelMap.Load(transferID); ok {
		cancel.(context.CancelFunc)()
		s.cancelMap.Delete(transferID)
		t, ok := s.GetTransfer(transferID)
		if ok {
			t.Status = TransferStatusCanceled
			s.StoreTransferToList(t)
		}
	}
}

func (s *Service) StoreTransfersToList(transfers []*Transfer) {
	for _, transfer := range transfers {
		s.transferList.Store(transfer.ID, transfer)
	}
	s.NotifyTransferListUpdate()
}

func (s *Service) StoreTransferToList(transfer *Transfer) {
	s.transferList.Store(transfer.ID, transfer)
	s.NotifyTransferListUpdate()
}

func (s *Service) NotifyTransferListUpdate() {
	s.app.Event.Emit("transfer:refreshList")
}

// CleanTransferList 清理完成的 transfer
func (s *Service) CleanTransferList() {
	s.transferList.Range(func(key, value any) bool {
		task := value.(*Transfer)
		if task.Status == TransferStatusCompleted ||
			task.Status == TransferStatusError ||
			task.Status == TransferStatusCanceled ||
			task.Status == TransferStatusRejected {
			s.transferList.Delete(key)
		}
		return true
	})
	s.NotifyTransferListUpdate()
}

func (s *Service) DeleteTransfer(transferID string) {
	s.transferList.Delete(transferID)
	s.NotifyTransferListUpdate()
}
