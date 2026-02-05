package transfer

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mesh-drop/internal/discovery"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func (s *Service) SendFiles(target *discovery.Peer, targetIP string, filePaths []string) {
	for _, filePath := range filePaths {
		s.SendFile(target, targetIP, filePath)
	}
}

func (s *Service) SendFile(target *discovery.Peer, targetIP string, filePath string) {
	taskID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelMap.Store(taskID, cancel)

	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("Failed to open file", "path", filePath, "error", err, "component", "transfer-client")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return
	}

	task := NewTransfer(
		taskID,
		NewSender(
			s.discoveryService.GetID(),
			s.config.GetHostName(),
			WithReceiverIP(targetIP, s.discoveryService),
		),
		WithFileName(filepath.Base(filePath)),
		WithFileSize(stat.Size()),
		WithType(TransferTypeSend),
		WithContentType(ContentTypeFile),
	)

	s.StoreTransferToList(task)

	go func() {
		// 任务结束后清理 ctx
		defer func() {
			s.cancelMap.Delete(taskID)
			cancel()
			s.NotifyTransferListUpdate()
		}()

		askResp, err := s.ask(ctx, target, targetIP, task)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				task.Status = TransferStatusCanceled
			} else {
				// 如果请求发送失败，更新状态为 Error
				task.Status = TransferStatusError
				task.ErrorMsg = fmt.Sprintf("Failed to connect to receiver: %v", err)
			}
			return
		}
		if askResp.Accepted {
			s.processTransfer(ctx, askResp, target, targetIP, task, file)
		} else {
			// 接收方拒绝
			task.Status = TransferStatusRejected
			return
		}
	}()
}

func (s *Service) SendFolder(target *discovery.Peer, targetIP string, folderPath string) {
	taskID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelMap.Store(taskID, cancel)

	// 任务结束后清理 ctx
	defer func() {
		s.cancelMap.Delete(taskID)
		cancel()
		s.NotifyTransferListUpdate()
	}()

	size, err := calculateTarSize(ctx, folderPath)
	if err != nil {
		slog.Error("Failed to calculate folder size", "path", folderPath, "error", err, "component", "transfer-client")
		return
	}

	task := NewTransfer(
		taskID,
		NewSender(
			s.discoveryService.GetID(),
			s.config.GetHostName(),
			WithReceiverIP(targetIP, s.discoveryService),
		),
		WithFileName(filepath.Base(folderPath)),
		WithFileSize(size),
		WithType(TransferTypeSend),
		WithContentType(ContentTypeFolder),
	)

	s.StoreTransferToList(task)

	askResp, err := s.ask(ctx, target, targetIP, task)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			task.Status = TransferStatusCanceled
		} else {
			// 如果请求发送失败，更新状态为 Error
			task.Status = TransferStatusError
			task.ErrorMsg = fmt.Sprintf("Failed to connect to receiver: %v", err)
		}
		return
	}
	if askResp.Accepted {
		r, w := io.Pipe()
		go func(ctx context.Context) {
			defer w.Close()
			if err := streamFolderToTar(ctx, w, folderPath); err != nil {
				slog.Error("Failed to stream folder to tar", "error", err, "component", "transfer-client")
				w.CloseWithError(err)
			}
		}(ctx)
		s.processTransfer(ctx, askResp, target, targetIP, task, r)
	} else {
		// 接收方拒绝
		task.Status = TransferStatusRejected
	}
}

func (s *Service) SendText(target *discovery.Peer, targetIP string, text string) {
	taskID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelMap.Store(taskID, cancel)

	r := bytes.NewReader([]byte(text))
	task := NewTransfer(
		taskID,
		NewSender(
			s.discoveryService.GetID(),
			s.config.GetHostName(),
			WithReceiverIP(targetIP, s.discoveryService),
		),
		WithFileSize(int64(len(text))),
		WithType(TransferTypeSend),
		WithContentType(ContentTypeText),
	)

	s.StoreTransferToList(task)

	go func() {
		// 任务结束后清理 ctx
		defer func() {
			s.cancelMap.Delete(taskID)
			cancel()
			s.NotifyTransferListUpdate()
		}()

		askResp, err := s.ask(ctx, target, targetIP, task)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				task.Status = TransferStatusCanceled
			} else {
				// 如果请求发送失败，更新状态为 Error
				task.Status = TransferStatusError
				task.ErrorMsg = fmt.Sprintf("Failed to connect to receiver: %v", err)
			}
			return
		}
		if askResp.Accepted {
			s.processTransfer(ctx, askResp, target, targetIP, task, r)
		} else {
			// 接收方拒绝
			task.Status = TransferStatusRejected
			return
		}
	}()
}

// ask 向接收端发送传输请求
func (s *Service) ask(ctx context.Context, target *discovery.Peer, targetIP string, task *Transfer) (TransferAskResponse, error) {
	if err := ctx.Err(); err != nil {
		return TransferAskResponse{}, err
	}

	// 发送请求
	askBody, _ := json.Marshal(task)

	askUrl := fmt.Sprintf("https://%s:%d/transfer/ask", targetIP, target.Port)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, askUrl, bytes.NewReader(askBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return TransferAskResponse{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return TransferAskResponse{}, err
	}
	defer resp.Body.Close()

	var askResp TransferAskResponse
	if err := json.NewDecoder(resp.Body).Decode(&askResp); err != nil {
		return TransferAskResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return TransferAskResponse{}, errors.New(askResp.Message)
	}
	return askResp, nil
}

// processTransfer 传输数据
func (s *Service) processTransfer(ctx context.Context, askResp TransferAskResponse, target *discovery.Peer, targetIP string, task *Transfer, payload io.Reader) {
	defer func() {
		s.NotifyTransferListUpdate()
	}()

	if err := ctx.Err(); err != nil {
		return
	}
	uploadUrl, _ := url.Parse(fmt.Sprintf("https://%s:%d/transfer/upload/%s", targetIP, target.Port, task.ID))
	query := uploadUrl.Query()
	query.Add("token", askResp.Token)
	uploadUrl.RawQuery = query.Encode()

	reader := &PassThroughReader{
		Reader: payload,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadUrl.String(), reader)
	if err != nil {
		return
	}
	req.ContentLength = task.FileSize
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			task.Status = TransferStatusCanceled
		} else {
			task.Status = TransferStatusError
			task.ErrorMsg = fmt.Sprintf("Failed to upload file: %v", err)
			slog.Error("Failed to upload file", "url", uploadUrl.String(), "error", err, "component", "transfer-client")
		}
		return
	}
	defer resp.Body.Close()

	var uploadResp TransferUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		task.Status = TransferStatusError
		task.ErrorMsg = uploadResp.Message
		return
	}

	// 判断任务完成还是被接收端取消
	if uploadResp.Status == TransferStatusCanceled {
		task.Status = TransferStatusCanceled
		task.ErrorMsg = uploadResp.Message
		return
	}

	// 传输成功，任务结束
	task.Status = TransferStatusCompleted
}

type countWriter struct {
	n int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

func calculateTarSize(ctx context.Context, srcPath string) (int64, error) {
	var size int64
	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 计算相对路径
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// 使用 tar.FileInfoHeader 计算 header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// 保持与 streamFolderToTar 一致
		header.Name = filepath.ToSlash(relPath)
		if info.IsDir() {
			header.Name += "/"
		}

		cw := &countWriter{}
		tw := tar.NewWriter(cw)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// tw.WriteHeader 写入 header blocks（包括扩展头）
		size += cw.n

		if !info.IsDir() {
			// 文件内容大小 + 填充
			fileSize := info.Size()
			blocks := math.Ceil(float64(fileSize) / 512)
			size += int64(blocks) * 512
		}

		return nil
	})

	// 两个 512 字节的空块作为结束标记
	size += 1024

	return size, err
}

func streamFolderToTar(ctx context.Context, w io.Writer, srcPath string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		slog.Debug("Processing file", "path", path, "relPath", relPath, "component", "transfer-client")

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// tar 文件名使用正斜杠
		header.Name = filepath.ToSlash(relPath)
		if info.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})
}
