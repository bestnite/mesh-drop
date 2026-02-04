package transfer

import (
	"context"
	"fmt"
	"log/slog"
	"mesh-drop/internal/discovery"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	app      *application.App
	port     int
	savePath string // 默认下载目录

	// pendingRequests 存储等待用户确认的通道
	// Key: TransferID, Value: Transfer
	transferList sync.Map

	discoveryService *discovery.Service

	// cancelMap 存储取消操作的通道
	// Key: TransferID, Value: context.CancelFunc
	cancelMap sync.Map
}

func NewService(app *application.App, port int, defaultSavePath string, discoveryService *discovery.Service) *Service {
	gin.SetMode(gin.ReleaseMode)

	return &Service{
		app:              app,
		port:             port,
		savePath:         defaultSavePath,
		discoveryService: discoveryService,
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
		addr := fmt.Sprintf(":%d", s.port)
		slog.Info("Transfer service listening", "address", addr, "component", "transfer")
		if err := r.Run(addr); err != nil {
			slog.Error("Transfer service error", "error", err, "component", "transfer")
		}
	}()
}

func (s *Service) GetTransferList() []Transfer {
	var requests []Transfer
	s.transferList.Range(func(key, value any) bool {
		requests = append(requests, value.(Transfer))
		return true
	})
	return requests
}

func (s *Service) GetTransfer(transferID string) (Transfer, bool) {
	val, ok := s.transferList.Load(transferID)
	if !ok {
		return Transfer{}, false
	}
	return val.(Transfer), true
}

func (s *Service) CancelTransfer(transferID string) {
	if cancel, ok := s.cancelMap.Load(transferID); ok {
		cancel.(context.CancelFunc)()
		s.cancelMap.Delete(transferID)
		t, ok := s.GetTransfer(transferID)
		if ok {
			t.Status = TransferStatusCanceled
			s.transferList.Store(transferID, t)
			s.app.Event.Emit("transfer:refreshList")
		}
	}
}
