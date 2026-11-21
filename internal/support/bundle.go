package support

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"time"
)

// bundleFileMode is the permission bits recorded on every bundle entry.
const bundleFileMode = 0o600

// File is one already-redacted document in a support bundle.
type File struct {
	Name string
	Data []byte
}

// WriteBundle writes files to w as a gzipped tar. Callers redact before they
// hand a file over: this package's job here is only to refuse a name that
// could escape the directory the bundle is unpacked into.
func WriteBundle(w io.Writer, files []File, now time.Time) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	for _, file := range files {
		if !LocalName(file.Name) {
			return fmt.Errorf("bundle entry %q is not a relative path", file.Name)
		}
		header := &tar.Header{
			Name:     file.Name,
			Mode:     bundleFileMode,
			Size:     int64(len(file.Data)),
			ModTime:  now,
			Typeflag: tar.TypeReg,
			Format:   tar.FormatPAX,
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("write header %s: %w", file.Name, err)
		}
		if _, err := tw.Write(file.Data); err != nil {
			return fmt.Errorf("write %s: %w", file.Name, err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	return nil
}
