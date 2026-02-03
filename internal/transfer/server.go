package transfer

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mesh-drop/internal/discovery"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// handleAsk 处理接收文件请求
func (s *Service) handleAsk(c *gin.Context) {
	var task Transfer

	// Gin 的 BindJSON 自动处理 JSON 解析
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, TransferAskResponse{
			ID:      task.ID,
			Message: "Invalid request",
		})
		return
	}

	// 检查是否已经存在
	if _, exists := s.transferList.Load(task.ID); exists {
		// 如果已经存在，说明是网络重试，直接忽略
		return
	}

	// 存储请求
	task.Type = TransferTypeReceive
	task.Status = TransferStatusPending
	task.DecisionChan = make(chan Decision)
	s.transferList.Store(task.ID, task)

	// 通知 Wails 前端
	s.app.Event.Emit("transfer:refreshList")

	// 等待用户决策或发送端放弃
	select {
	case decision := <-task.DecisionChan:
		// 用户决策
		if decision.Accepted {
			task.Status = TransferStatusAccepted
			task.SavePath = decision.SavePath
			token := uuid.New().String()
			task.Token = token
			s.transferList.Store(task.ID, task)
		} else {
			task.Status = TransferStatusRejected
			s.transferList.Store(task.ID, task)
		}
		c.JSON(http.StatusOK, TransferAskResponse{
			ID:       task.ID,
			Accepted: decision.Accepted,
			Token:    task.Token,
		})
	case <-c.Done():
		// 发送端放弃
		task.Status = TransferStatusCanceled
		s.transferList.Store(task.ID, task)
		s.app.Event.Emit("transfer:refreshList")
	}
}

// ResolvePendingRequest 外部调用，解决待处理的传输请求
// 返回 true 表示成功处理，false 表示未找到该 ID 的请求
func (s *Service) ResolvePendingRequest(id string, accept bool, savePath string) bool {
	val, ok := s.transferList.Load(id)
	if !ok {
		return false
	}

	task := val.(Transfer)
	task.DecisionChan <- Decision{
		ID:       id,
		Accepted: accept,
		SavePath: savePath,
	}
	return true
}

// handleUpload 处理接收文件请求
func (s *Service) handleUpload(c *gin.Context) {
	id := c.Param("id")
	token := c.Query("token")

	if id == "" || token == "" {
		c.JSON(http.StatusBadRequest, TransferUploadResponse{
			ID:      id,
			Message: "Invalid request: missing id or token",
		})
		return
	}

	// 获取传输任务
	val, ok := s.transferList.Load(id)
	if !ok {
		c.JSON(http.StatusUnauthorized, TransferUploadResponse{
			ID:      id,
			Message: "Invalid request: task not found",
		})
		return
	}
	task := val.(Transfer)

	// 校验 token
	if task.Token != token {
		c.JSON(http.StatusUnauthorized, TransferUploadResponse{
			ID:      id,
			Message: "Token mismatch",
		})
		return
	}

	// 校验状态
	if task.Status != TransferStatusAccepted {
		c.JSON(http.StatusForbidden, TransferUploadResponse{
			ID:      id,
			Message: "Invalid task status",
		})
		return
	}

	// 更新状态为 active
	task.Status = TransferStatusActive
	s.transferList.Store(task.ID, task)
	s.app.Event.Emit("transfer:refreshList")

	savePath := task.SavePath
	if savePath == "" {
		savePath = s.savePath
	}

	switch task.ContentType {
	case ContentTypeFile:
		destPath := filepath.Join(savePath, task.FileName)
		file, err := os.Create(destPath)
		if err != nil {
			// 接收方无法创建文件，直接报错，任务结束
			c.JSON(http.StatusInternalServerError, TransferUploadResponse{
				ID:      task.ID,
				Message: "Receiver failed to create file",
			})
			slog.Error("Failed to create file", "error", err, "component", "transfer")
			task.Status = TransferStatusError
			task.ErrorMsg = fmt.Errorf("receiver failed to create file: %v", err).Error()
			s.transferList.Store(task.ID, task)
			// 通知前端传输失败
			s.app.Event.Emit("transfer:refreshList")
			return
		}
		defer file.Close()
		s.receive(c, &task, file)
	case ContentTypeText:
		var buf bytes.Buffer
		s.receive(c, &task, &buf)
		task.Text = buf.String()
		s.transferList.Store(task.ID, task)
		s.app.Event.Emit("transfer:refreshList")
	case ContentTypeFolder:
		s.receiveFolder(c, savePath, &task)
	}
}

func (s *Service) receive(c *gin.Context, task *Transfer, writer io.Writer) {
	// 包装 reader，用于计算进度
	reader := &PassThroughReader{
		Reader: c.Request.Body,
		total:  task.FileSize,
		callback: func(current, total int64, speed float64) {
			task.Progress = Progress{
				Current: current,
				Total:   total,
				Speed:   speed,
			}
			s.transferList.Store(task.ID, *task)
			s.app.Event.Emit("transfer:refreshList")
		},
	}

	_, err := io.Copy(writer, reader)
	if err != nil {
		// 文件写入失败，直接报错，任务结束
		c.JSON(http.StatusInternalServerError, TransferUploadResponse{
			ID:      task.ID,
			Message: "Failed to write file",
		})
		slog.Error("Failed to write file", "error", err, "component", "transfer")
		task.Status = TransferStatusError
		task.ErrorMsg = fmt.Errorf("failed to write file: %v", err).Error()
		s.transferList.Store(task.ID, *task)
		// 通知前端传输失败
		s.app.Event.Emit("transfer:refreshList")
		return
	}

	c.JSON(http.StatusOK, TransferUploadResponse{
		ID:      task.ID,
		Message: "File received successfully",
	})
	// 传输成功，任务结束
	task.Status = TransferStatusCompleted
	s.transferList.Store(task.ID, *task)
	s.app.Event.Emit("transfer:refreshList")
}

func (s *Service) receiveFolder(c *gin.Context, savePath string, task *Transfer) {
	// 创建根目录
	destPath := filepath.Join(savePath, task.FileName)
	if err := os.MkdirAll(destPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, TransferUploadResponse{
			ID:      task.ID,
			Message: "Receiver failed to create folder",
		})
		return
	}

	// 包装 reader，用于计算进度
	reader := &PassThroughReader{
		Reader: c.Request.Body,
		total:  task.FileSize,
		callback: func(current, total int64, speed float64) {
			task.Progress = Progress{
				Current: current,
				Total:   total,
				Speed:   speed,
			}
			s.transferList.Store(task.ID, *task)
			s.app.Event.Emit("transfer:refreshList")
		},
	}

	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, TransferUploadResponse{
				ID:      task.ID,
				Message: "Stream error",
			})
			slog.Error("Tar stream error", "error", err)
			return
		}

		target := filepath.Join(destPath, header.Name)
		// 确保路径没有越界
		if !strings.HasPrefix(target, filepath.Clean(destPath)+string(os.PathSeparator)) {
			// 非法路径
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				slog.Error("Failed to create dir", "path", target, "error", err)
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				slog.Error("Failed to create file", "path", target, "error", err)
				continue
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				slog.Error("Failed to write file", "path", target, "error", err)
				c.JSON(http.StatusInternalServerError, TransferUploadResponse{
					ID:      task.ID,
					Message: "Write error",
				})
				return
			}
			f.Close()
		}
	}

	c.JSON(http.StatusOK, TransferUploadResponse{
		ID:      task.ID,
		Message: "Folder received successfully",
	})
	task.Progress.Total = task.FileSize
	task.Progress.Current = task.FileSize
	task.Status = TransferStatusCompleted
	s.transferList.Store(task.ID, *task)
	s.app.Event.Emit("transfer:refreshList")
}

func (s *Service) GetTransferList() []Transfer {
	var requests []Transfer
	s.transferList.Range(func(key, value any) bool {
		requests = append(requests, value.(Transfer))
		return true
	})
	return requests
}
