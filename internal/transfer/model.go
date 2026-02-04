package transfer

type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "pending"
	TransferStatusAccepted  TransferStatus = "accepted"
	TransferStatusRejected  TransferStatus = "rejected"
	TransferStatusCompleted TransferStatus = "completed"
	TransferStatusError     TransferStatus = "error"
	TransferStatusCanceled  TransferStatus = "canceled"
	TransferStatusActive    TransferStatus = "active"
)

type TransferType string

const (
	TransferTypeSend    TransferType = "send"
	TransferTypeReceive TransferType = "receive"
)

type ContentType string

const (
	ContentTypeFile   ContentType = "file"
	ContentTypeText   ContentType = "text"
	ContentTypeFolder ContentType = "folder"
)

// Transfer
type Transfer struct {
	ID           string         `json:"id" binding:"required"`     // 传输会话 ID
	Sender       Sender         `json:"sender" binding:"required"` // 发送者
	FileName     string         `json:"file_name"`                 // 文件名
	FileSize     int64          `json:"file_size"`                 // 文件大小 (字节)
	SavePath     string         `json:"savePath"`                  // 保存路径
	Status       TransferStatus `json:"status"`                    // 传输状态
	Progress     Progress       `json:"progress"`                  // 传输进度
	Type         TransferType   `json:"type"`                      // 进度类型
	ContentType  ContentType    `json:"content_type"`              // 内容类型
	Text         string         `json:"text"`                      // 文本内容
	ErrorMsg     string         `json:"error_msg"`                 // 错误信息
	Token        string         `json:"token"`                     // 用于上传的凭证
	DecisionChan chan Decision  `json:"-"`                         // 用户决策通道
}

type Sender struct {
	ID   string `json:"id" binding:"required"`   // 发送者 ID
	Name string `json:"name" binding:"required"` // 发送者名称
}

// Progress 用户前端传输进度
type Progress struct {
	Current int64   `json:"current"` // 当前进度
	Total   int64   `json:"total"`   // 总进度
	Speed   float64 `json:"speed"`   // 速度
}

// Decision 用户前端决策
type Decision struct {
	ID       string `json:"id"` // 传输会话 ID
	Accepted bool   `json:"accepted"`
	SavePath string `json:"save_path"`
}

// TransferAskResponse 握手回应
type TransferAskResponse struct {
	ID       string `json:"id"` // 传输会话 ID
	Accepted bool   `json:"accepted"`
	Token    string `json:"token,omitempty"`   // 用于上传的凭证
	Message  string `json:"message,omitempty"` // 错误信息
}

// TransferUploadResponse 上传回应
type TransferUploadResponse struct {
	ID      string         `json:"id"` // 传输会话 ID
	Message string         `json:"message"`
	Status  TransferStatus `json:"status"`
}
