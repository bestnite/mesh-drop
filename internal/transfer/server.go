package transfer

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// handleAsk 处理接收文件请求
func (s *Service) handleAsk(c *gin.Context) {
	defer s.NotifyTransferListUpdate()
	var task Transfer

	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, TransferAskResponse{
			ID:      task.ID,
			Message: "Invalid request",
		})
		return
	}

	// 检查是否已经存在
	if _, exists := s.transfers.Load(task.ID); exists {
		// 如果已经存在，说明是网络重试，直接忽略
		return
	}

	// 存储请求
	task.Type = TransferTypeReceive
	task.Status = TransferStatusPending
	task.DecisionChan = make(chan Decision, 1)
	s.StoreTransferToList(&task)

	// 从本地获取 peer 检查是否 mismatch
	peer, ok := s.discoveryService.GetPeerByID(task.Sender.ID)
	if ok {
		task.Sender.TrustMismatch = peer.TrustMismatch
	}

	if s.config.GetAutoAccept() || (s.config.IsTrusted(task.Sender.ID) && !task.Sender.TrustMismatch) {
		task.DecisionChan <- Decision{
			ID:       task.ID,
			Accepted: true,
			SavePath: s.config.GetSavePath(),
		}
	} else {
		// 发送系统通知
		_ = s.notifier.SendNotification(notifications.NotificationOptions{
			ID:    uuid.New().String(),
			Title: "File Transfer Request",
			Body:  fmt.Sprintf("%s wants to transfer %s", task.Sender.Name, task.FileName),
		})
	}

	// 等待用户决策或发送端放弃
	select {
	case decision := <-task.DecisionChan:
		// 用户决策
		if decision.Accepted {
			task.Status = TransferStatusAccepted
			task.SavePath = decision.SavePath
			token := uuid.New().String()
			task.Token = token
			c.JSON(http.StatusOK, TransferAskResponse{
				ID:       task.ID,
				Accepted: decision.Accepted,
				Token:    task.Token,
			})
		} else {
			task.Status = TransferStatusRejected
			c.JSON(http.StatusOK, TransferAskResponse{
				ID:       task.ID,
				Accepted: false,
				Message:  "Transfer rejected",
			})
		}
	case <-c.Request.Context().Done():
		// 发送端放弃
		task.Status = TransferStatusCanceled
	}
}

// ResolvePendingRequest 外部调用，解决待处理的传输请求
// 返回 true 表示成功处理，false 表示未找到该 ID 的请求
func (s *Service) ResolvePendingRequest(id string, accept bool, savePath string) bool {
	task, ok := s.GetTransfer(id)
	if !ok {
		return false
	}
	task.DecisionChan <- Decision{
		ID:       id,
		Accepted: accept,
		SavePath: savePath,
	}
	return true
}

// handleUpload 处理接收文件请求
func (s *Service) handleUpload(c *gin.Context) {
	defer s.NotifyTransferListUpdate()
	id := c.Param("id")
	token := c.Query("token")

	if id == "" || token == "" {
		c.JSON(http.StatusBadRequest, TransferUploadResponse{
			ID:      id,
			Message: "Invalid request: missing id or token",
			Status:  TransferStatusError,
		})
		return
	}

	// 获取传输任务
	task, ok := s.GetTransfer(id)
	if !ok {
		c.JSON(http.StatusUnauthorized, TransferUploadResponse{
			ID:      id,
			Message: "Invalid request: task not found",
			Status:  TransferStatusError,
		})
		return
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	s.cancelMap.Store(task.ID, cancel)
	defer func() {
		s.cancelMap.Delete(task.ID)
		cancel()
	}()

	// 校验 token
	if task.Token != token {
		c.JSON(http.StatusUnauthorized, TransferUploadResponse{
			ID:      id,
			Message: "Token mismatch",
			Status:  TransferStatusError,
		})
		return
	}

	// 校验状态
	if task.Status != TransferStatusAccepted {
		c.JSON(http.StatusForbidden, TransferUploadResponse{
			ID:      id,
			Message: "Invalid task status",
			Status:  TransferStatusError,
		})
		return
	}

	// 更新状态为 active
	task.Status = TransferStatusActive

	savePath := task.SavePath
	if savePath == "" {
		savePath = s.config.GetSavePath()
	}

	ctxReader := &ContextReader{
		ctx: ctx,
		r:   c.Request.Body,
	}

	switch task.ContentType {
	case ContentTypeFile:
		destPath := filepath.Join(savePath, task.FileName)
		// 如果文件已存在则在文件名后追加序号
		_, err := os.Stat(destPath)
		counter := 1
		for err == nil {
			destPath = filepath.Join(savePath, fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(task.FileName, filepath.Ext(task.FileName)), counter, filepath.Ext(task.FileName)))
			counter++
			_, err = os.Stat(destPath)
		}
		file, err := os.Create(destPath)
		if err != nil {
			// 接收方无法创建文件，直接报错，任务结束
			c.JSON(http.StatusInternalServerError, TransferUploadResponse{
				ID:      task.ID,
				Message: "Receiver failed to create file",
				Status:  TransferStatusError,
			})
			slog.Error("Failed to create file", "error", err, "component", "transfer")
			task.Status = TransferStatusError
			task.ErrorMsg = fmt.Errorf("receiver failed to create file: %v", err).Error()
			return
		}
		defer file.Close()
		s.receive(c, task, Writer{w: file, filePath: destPath}, ctxReader)
	case ContentTypeText:
		var buf bytes.Buffer
		s.receive(c, task, Writer{w: &buf, filePath: ""}, ctxReader)
		task.Text = buf.String()
	case ContentTypeFolder:
		s.receiveFolder(c, savePath, task, ctxReader)
	}
}

func (s *Service) receive(c *gin.Context, task *Transfer, writer Writer, ctxReader io.Reader) {
	// 包装 reader，用于计算进度
	reader := &PassThroughReader{
		Reader: ctxReader,
		total:  task.FileSize,
		callback: func(current, total int64, speed float64) {
			task.Progress = Progress{
				Current: current,
				Total:   total,
				Speed:   speed,
			}
			task.Status = TransferStatusActive
			s.NotifyTransferListUpdate()
		},
	}

	_, err := io.Copy(writer, reader)
	if err != nil {
		// 发送端断线，任务取消
		if c.Request.Context().Err() != nil {
			slog.Info("Sender canceled transfer (Network/Context disconnected)", "id", task.ID, "raw_err", err)
			task.ErrorMsg = "Sender disconnected"
			task.Status = TransferStatusCanceled
			return
		}

		// 用户取消传输
		if errors.Is(err, context.Canceled) {
			slog.Info("User canceled transfer", "component", "transfer")
			task.ErrorMsg = "User canceled transfer"
			task.Status = TransferStatusCanceled
			// 通知发送端
			c.JSON(http.StatusOK, TransferUploadResponse{
				ID:      task.ID,
				Message: "File transfer canceled",
				Status:  TransferStatusCanceled,
			})
			return
		}

		// 接收端写文件失败
		c.JSON(http.StatusInternalServerError, TransferUploadResponse{
			ID:      task.ID,
			Message: "Failed to write file",
			Status:  TransferStatusError,
		})
		slog.Error("Failed to write file", "error", err, "component", "transfer")
		task.Status = TransferStatusError
		task.ErrorMsg = fmt.Errorf("failed to write file: %v", err).Error()

		// 删除文件
		if task.ContentType == ContentTypeFile && writer.GetFilePath() != "" {
			_ = os.Remove(writer.GetFilePath())
		}
		return
	}

	c.JSON(http.StatusOK, TransferUploadResponse{
		ID:      task.ID,
		Message: "File received successfully",
		Status:  TransferStatusCompleted,
	})
	// 传输成功，任务结束
	task.Status = TransferStatusCompleted
}

func (s *Service) receiveFolder(c *gin.Context, savePath string, task *Transfer, ctxReader io.Reader) {
	defer s.NotifyTransferListUpdate()

	// 创建根目录
	destPath := filepath.Join(savePath, task.FileName)
	// 如果文件已存在则在文件名后追加序号
	_, err := os.Stat(destPath)
	counter := 1
	for err == nil {
		destPath = filepath.Join(savePath, fmt.Sprintf("%s (%d)", task.FileName, counter))
		counter++
		_, err = os.Stat(destPath)
	}
	if err := os.MkdirAll(destPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, TransferUploadResponse{
			ID:      task.ID,
			Message: "Receiver failed to create folder",
			Status:  TransferStatusError,
		})
		slog.Error("Failed to create folder", "error", err, "component", "transfer")
		task.Status = TransferStatusError
		task.ErrorMsg = fmt.Errorf("receiver failed to create folder: %v", err).Error()
		return
	}

	// 包装 reader，用于计算进度
	reader := &PassThroughReader{
		Reader: ctxReader,
		total:  task.FileSize,
		callback: func(current, total int64, speed float64) {
			task.Progress = Progress{
				Current: current,
				Total:   total,
				Speed:   speed,
			}
			task.Status = TransferStatusActive
			s.NotifyTransferListUpdate()
		},
	}

	handleError := func(err error, stage string) bool {
		if err == nil {
			return false
		}
		if c.Request.Context().Err() != nil {
			slog.Info("Transfer canceled by sender (Network disconnect)", "id", task.ID, "stage", stage)
			task.Status = TransferStatusCanceled
			task.ErrorMsg = "Sender disconnected"
			// 发送端已断开，无需也不应再发送 c.JSON
			return true
		}

		if errors.Is(err, context.Canceled) {
			slog.Info("Transfer canceled by user", "id", task.ID, "stage", stage)
			task.Status = TransferStatusCanceled
			task.ErrorMsg = "User canceled transfer"
			// 通知发送端（虽然此时连接可能即将关闭，但尽力通知）
			c.JSON(http.StatusOK, TransferUploadResponse{
				ID:      task.ID,
				Message: "File transfer canceled",
				Status:  TransferStatusCanceled,
			})
			return true
		}

		slog.Error("Transfer failed", "error", err, "stage", stage)
		task.Status = TransferStatusError
		task.ErrorMsg = fmt.Sprintf("Failed at %s: %v", stage, err)

		c.JSON(http.StatusInternalServerError, TransferUploadResponse{
			ID:      task.ID,
			Message: fmt.Sprintf("Transfer failed: %v", err),
			Status:  TransferStatusError,
		})
		return true
	}

	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if handleError(err, "read_tar_header") {
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
				if handleError(err, "write_file_content") {
					return
				}
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
}
