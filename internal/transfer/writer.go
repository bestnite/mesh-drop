package transfer

import "io"

type Writer struct {
	w        io.Writer
	filePath string
}

func (w Writer) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}

func (w Writer) GetFilePath() string {
	return w.filePath
}
