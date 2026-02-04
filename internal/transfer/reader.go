package transfer

import (
	"context"
	"io"
)

// ContextReader 带有 Context 的 Reader
type ContextReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *ContextReader) Read(p []byte) (n int, err error) {
	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err() // 返回 context.Canceled 错误
	default:
		return cr.r.Read(p)
	}
}
