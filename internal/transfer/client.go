package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
