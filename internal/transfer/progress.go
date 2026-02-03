package transfer

import (
	"io"
	"time"
)

type ProgressCallback func(current int64, total int64, speed float64)

const (
	ProgressInterval = 100 * time.Millisecond
)

// PassThroughReader 包装 io.Reader 以计算读取字节数
type PassThroughReader struct {
	io.Reader
	total      int64
	currentLen int64
	lastTime   time.Time
	lastLen    int64
	callback   ProgressCallback
}

func (pt *PassThroughReader) Read(p []byte) (int, error) {
	n, err := pt.Reader.Read(p)
	pt.currentLen += int64(n)

	if time.Since(pt.lastTime) > ProgressInterval || err == io.EOF {
		// 计算速度，单位为字节/秒
		speed := float64(pt.currentLen-pt.lastLen) / time.Since(pt.lastTime).Seconds()
		pt.callback(pt.currentLen, pt.total, speed)
		pt.lastTime = time.Now()
		pt.lastLen = pt.currentLen
	}

	return n, err
}
