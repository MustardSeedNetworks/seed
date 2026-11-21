package support

import (
	"bytes"
	"fmt"
	"io"
)

// TailLines returns the last n lines of r, reading no more than maxBytes from
// its end. The window opens on a line boundary, so a truncated read never
// starts mid-line.
func TailLines(r io.ReadSeeker, n int, maxBytes int64) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}
	data, err := TailBytes(r, maxBytes)
	if err != nil {
		return nil, err
	}
	text := string(bytes.TrimRight(data, "\n"))
	if text == "" {
		return nil, nil
	}
	lines := bytes.Split([]byte(text), []byte("\n"))
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = string(line)
	}
	return out, nil
}

// TailBytes returns the last maxBytes of r, starting at the first newline when
// it had to cut, so the tail never opens mid-line.
func TailBytes(r io.ReadSeeker, maxBytes int64) ([]byte, error) {
	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}
	truncated := size > maxBytes
	start := int64(0)
	if truncated {
		start = size - maxBytes
	}
	if _, seekErr := r.Seek(start, io.SeekStart); seekErr != nil {
		return nil, fmt.Errorf("seek: %w", seekErr)
	}
	data, readErr := io.ReadAll(io.LimitReader(r, maxBytes))
	if readErr != nil {
		return nil, fmt.Errorf("read: %w", readErr)
	}
	if truncated {
		if cut := bytes.IndexByte(data, '\n'); cut >= 0 {
			data = data[cut+1:]
		}
	}
	return data, nil
}
