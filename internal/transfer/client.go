package transfer

import (
	"archive/tar"
	"bytes"
	"encoding/json"
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

func (s *Service) SendFile(target *discovery.Peer, targetIP string, filePath string) {
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

	task := Transfer{
		ID:       uuid.New().String(),
		FileName: filepath.Base(filePath),
		FileSize: stat.Size(),
		Sender: Sender{
			ID:   s.discoveryService.GetID(),
			Name: s.discoveryService.GetName(),
		},
		Type:        TransferTypeSend,
		Status:      TransferStatusPending,
		ContentType: ContentTypeFile,
	}

	s.processTransfer(target, targetIP, task, file)
}

func (s *Service) SendFolder(target *discovery.Peer, targetIP string, folderPath string) {
	size, err := calculateTarSize(folderPath)
	if err != nil {
		slog.Error("Failed to calculate folder size", "path", folderPath, "error", err, "component", "transfer-client")
		return
	}

	r, w := io.Pipe()

	go func() {
		defer w.Close()
		if err := streamFolderToTar(w, folderPath); err != nil {
			slog.Error("Failed to stream folder to tar", "error", err, "component", "transfer-client")
			w.CloseWithError(err)
		}
	}()

	task := Transfer{
		ID:       uuid.New().String(),
		FileName: filepath.Base(folderPath),
		FileSize: size,
		Sender: Sender{
			ID:   s.discoveryService.GetID(),
			Name: s.discoveryService.GetName(),
		},
		Type:        TransferTypeSend,
		Status:      TransferStatusPending,
		ContentType: ContentTypeFolder,
	}

	s.processTransfer(target, targetIP, task, r)
}

func (s *Service) SendText(target *discovery.Peer, targetIP string, text string) {
	reader := bytes.NewReader([]byte(text))
	task := Transfer{
		ID:       uuid.New().String(),
		FileName: "",
		FileSize: int64(len(text)),
		Sender: Sender{
			ID:   s.discoveryService.GetID(),
			Name: s.discoveryService.GetName(),
		},
		Type:        TransferTypeSend,
		Status:      TransferStatusPending,
		ContentType: ContentTypeText,
	}

	s.processTransfer(target, targetIP, task, reader)
}

type countWriter struct {
	n int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

func calculateTarSize(srcPath string) (int64, error) {
	var size int64
	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

func streamFolderToTar(w io.Writer, srcPath string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

func (s *Service) processTransfer(target *discovery.Peer, targetIP string, task Transfer, payload io.Reader) {
	s.transferList.Store(task.ID, task)
	s.app.Event.Emit("transfer:refreshList")

	// 发送请求
	askBody, _ := json.Marshal(task)

	askUrl := fmt.Sprintf("http://%s:%d/transfer/ask", targetIP, target.Port)

	resp, err := http.Post(askUrl, "application/json", bytes.NewReader(askBody))
	if err != nil {
		slog.Error("Failed to send ask request", "url", askUrl, "error", err, "component", "transfer-client")
		// 如果请求发送失败，更新状态为 Error
		task.Status = TransferStatusError
		task.ErrorMsg = fmt.Sprintf("Failed to connect to receiver: %v", err)
		s.transferList.Store(task.ID, task)
		s.app.Event.Emit("transfer:refreshList")
		return
	}
	defer resp.Body.Close()

	var askResp TransferAskResponse
	if err := json.NewDecoder(resp.Body).Decode(&askResp); err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		task.Status = TransferStatusError
		task.ErrorMsg = askResp.Message
		s.transferList.Store(task.ID, task)
		s.app.Event.Emit("transfer:refreshList")
		return
	}

	if !askResp.Accepted {
		// 接收方拒绝
		task.Status = TransferStatusRejected
		s.transferList.Store(task.ID, task)
		s.app.Event.Emit("transfer:refreshList")
		return
	}

	// 上传
	uploadUrl, _ := url.Parse(fmt.Sprintf("http://%s:%d/transfer/upload/%s", targetIP, target.Port, task.ID))
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
			s.transferList.Store(task.ID, task)
			s.app.Event.Emit("transfer:refreshList")
		},
	}

	req, err := http.NewRequest(http.MethodPut, uploadUrl.String(), reader)
	if err != nil {
		return
	}
	req.ContentLength = task.FileSize
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Failed to upload file", "url", uploadUrl.String(), "error", err, "component", "transfer-client")
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
		s.transferList.Store(task.ID, task)
		s.app.Event.Emit("transfer:refreshList")
		return
	}

	// 传输成功，任务结束
	task.Status = TransferStatusCompleted
	s.transferList.Store(task.ID, task)
	s.app.Event.Emit("transfer:refreshList")
}
