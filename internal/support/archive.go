package support

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/fsutil"
)

// Backup archive entries. The archive never carries a path from the machine
// that wrote it: each entry names a role, and restore maps the role to a
// destination under the layout of the machine reading it. That is what makes a
// hostile or corrupted archive unable to land a file anywhere unexpected.
const (
	EntryConfig      = "config/seed.json"
	EntryKeyring     = "config/credential.key"
	EntryDatabase    = "data/seed.db"
	EntryDatabaseWAL = "data/seed.db-wal"
	EntryDatabaseSHM = "data/seed.db-shm"
	EntryCertificate = "certs/server.crt"
	EntryKey         = "certs/server.key"

	// ManifestEntry is the archive's first entry: format version and contents.
	ManifestEntry = "backup.json"
)

const (
	manifestVersion = 1
	// maxManifestBytes bounds the manifest read; it is a few hundred bytes.
	maxManifestBytes = 1 << 20
	// maxEntryBytes bounds a single restored entry. A SQLite file past this is
	// not a seed database; refusing it keeps a crafted archive from filling
	// the disk.
	maxEntryBytes = 64 << 30

	dirMode        = 0o700
	privateMode    = 0o600
	publicCertMode = 0o644
	tmpSuffix      = ".restore-tmp"
)

// ErrArchive marks an archive restore refused as unsafe or malformed. Nothing
// has been written when it is returned.
var ErrArchive = errors.New("refusing archive")

// Member is one file to carry in a backup.
type Member struct {
	Entry string
	Path  string
}

// Target is where one archive entry lands on restore. RemoveIfAbsent deletes
// the destination when the archive has no such entry: a restored database must
// not keep the old database's write-ahead log, which would replay onto it.
type Target struct {
	Entry          string
	Path           string
	RemoveIfAbsent bool
}

type manifest struct {
	Version   int      `json:"version"`
	CreatedAt string   `json:"createdAt"`
	Seed      string   `json:"seedVersion"`
	Entries   []string `json:"entries"`
}

func knownEntry(name string) bool {
	switch name {
	case EntryConfig, EntryKeyring, EntryDatabase, EntryDatabaseWAL, EntryDatabaseSHM,
		EntryCertificate, EntryKey:
		return true
	default:
		return false
	}
}

// Backup writes members to w as a gzipped tar led by a manifest. Every source
// is opened through an [os.Root] on its own directory, so a source path that
// is, or contains, a symlink cannot be followed out of it.
func Backup(w io.Writer, members []Member, seedVersion string, now time.Time) error {
	sorted := append([]Member(nil), members...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Entry < sorted[j].Entry })

	entries := make([]string, 0, len(sorted))
	for _, member := range sorted {
		if !knownEntry(member.Entry) {
			return fmt.Errorf("backup entry %q is not a known role", member.Entry)
		}
		entries = append(entries, member.Entry)
	}
	doc, err := json.MarshalIndent(manifest{
		Version:   manifestVersion,
		CreatedAt: now.UTC().Format(time.RFC3339),
		Seed:      seedVersion,
		Entries:   entries,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("render manifest: %w", err)
	}

	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	if writeErr := writeBytes(tw, ManifestEntry, append(doc, '\n'), now); writeErr != nil {
		return writeErr
	}
	for _, member := range sorted {
		if copyErr := copyMember(tw, member, now); copyErr != nil {
			return copyErr
		}
	}
	if closeErr := tw.Close(); closeErr != nil {
		return fmt.Errorf("close tar: %w", closeErr)
	}
	if closeErr := gz.Close(); closeErr != nil {
		return fmt.Errorf("close gzip: %w", closeErr)
	}
	return nil
}

func writeBytes(tw *tar.Writer, name string, data []byte, now time.Time) error {
	header := &tar.Header{
		Name: name, Mode: privateMode, Size: int64(len(data)), ModTime: now,
		Typeflag: tar.TypeReg, Format: tar.FormatPAX,
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write header %s: %w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func copyMember(tw *tar.Writer, member Member, now time.Time) error {
	root, base, err := fsutil.RootAt(member.Path)
	if err != nil {
		return fmt.Errorf("open %s: %w", member.Path, err)
	}
	defer func() { _ = root.Close() }()

	file, openErr := root.Open(base)
	if openErr != nil {
		return fmt.Errorf("open %s: %w", member.Path, openErr)
	}
	defer func() { _ = file.Close() }()

	info, statErr := file.Stat()
	if statErr != nil {
		return fmt.Errorf("stat %s: %w", member.Path, statErr)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", member.Path)
	}
	header := &tar.Header{
		Name: member.Entry, Mode: privateMode, Size: info.Size(), ModTime: now,
		Typeflag: tar.TypeReg, Format: tar.FormatPAX,
	}
	if headerErr := tw.WriteHeader(header); headerErr != nil {
		return fmt.Errorf("write header %s: %w", member.Entry, headerErr)
	}
	if _, copyErr := io.CopyN(tw, file, info.Size()); copyErr != nil {
		return fmt.Errorf("copy %s: %w", member.Path, copyErr)
	}
	return nil
}

// Restore validates the whole archive, then replaces each target. Nothing is
// written until every entry has been read to its end and passed the checks, so
// a truncated, corrupt or hostile archive leaves the install exactly as it was.
// beforeInstall runs after validation and before the first write, which is
// where a caller keeps a copy of what is about to be replaced; its error
// aborts the restore. It returns the entries it restored.
func Restore(src io.ReadSeeker, targets []Target, beforeInstall func() error) ([]string, error) {
	byEntry := make(map[string]Target, len(targets))
	for _, target := range targets {
		byEntry[target.Entry] = target
	}
	present, err := validate(src, byEntry)
	if err != nil {
		return nil, err
	}
	if beforeInstall != nil {
		if hookErr := beforeInstall(); hookErr != nil {
			return nil, fmt.Errorf("before restoring: %w", hookErr)
		}
	}
	if _, seekErr := src.Seek(0, io.SeekStart); seekErr != nil {
		return nil, fmt.Errorf("rewind archive: %w", seekErr)
	}
	return install(src, byEntry, present)
}

func openTar(src io.Reader) (*tar.Reader, error) {
	gz, err := gzip.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("%w: not a gzip archive: %w", ErrArchive, err)
	}
	return tar.NewReader(gz), nil
}

// validate reads every entry to its end, checking names and types, and returns
// the set of target entries the archive carries.
func validate(src io.Reader, targets map[string]Target) (map[string]bool, error) {
	tr, err := openTar(src)
	if err != nil {
		return nil, err
	}
	present := make(map[string]bool)
	sawManifest := false
	for {
		header, nextErr := tr.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrArchive, nextErr)
		}
		if checkErr := checkHeader(header, targets, present, sawManifest); checkErr != nil {
			return nil, checkErr
		}
		if header.Name == ManifestEntry {
			if manifestErr := readManifest(tr); manifestErr != nil {
				return nil, manifestErr
			}
			sawManifest = true
			continue
		}
		if _, copyErr := io.Copy(io.Discard, tr); copyErr != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrArchive, header.Name, copyErr)
		}
		present[header.Name] = true
	}
	if !sawManifest {
		return nil, fmt.Errorf("%w: no %s; this is not a seed backup", ErrArchive, ManifestEntry)
	}
	if len(present) == 0 {
		return nil, fmt.Errorf("%w: the archive holds no files", ErrArchive)
	}
	return present, nil
}

func checkHeader(header *tar.Header, targets map[string]Target, present map[string]bool, sawManifest bool) error {
	name := header.Name
	if header.Typeflag != tar.TypeReg {
		return fmt.Errorf("%w: %q is not a regular file", ErrArchive, name)
	}
	if !LocalName(name) {
		return fmt.Errorf("%w: %q escapes the install", ErrArchive, name)
	}
	if header.Size < 0 || header.Size > maxEntryBytes {
		return fmt.Errorf("%w: %q has an implausible size", ErrArchive, name)
	}
	if name == ManifestEntry {
		if sawManifest {
			return fmt.Errorf("%w: duplicate %s", ErrArchive, ManifestEntry)
		}
		return nil
	}
	if !sawManifest {
		return fmt.Errorf("%w: %s must lead the archive", ErrArchive, ManifestEntry)
	}
	if _, ok := targets[name]; !ok || !knownEntry(name) {
		return fmt.Errorf("%w: unexpected entry %q", ErrArchive, name)
	}
	if present[name] {
		return fmt.Errorf("%w: duplicate entry %q", ErrArchive, name)
	}
	return nil
}

// LocalName reports whether name is a clean, relative, forward-slash path that
// stays inside the directory it is unpacked into.
func LocalName(name string) bool {
	return filepath.IsLocal(name) && !strings.Contains(name, `\`) && path.Clean(name) == name
}

func readManifest(r io.Reader) error {
	var doc manifest
	if err := json.NewDecoder(io.LimitReader(r, maxManifestBytes)).Decode(&doc); err != nil {
		return fmt.Errorf("%w: unreadable %s: %w", ErrArchive, ManifestEntry, err)
	}
	if doc.Version != manifestVersion {
		return fmt.Errorf("%w: backup format %d is not supported (want %d)", ErrArchive, doc.Version, manifestVersion)
	}
	return nil
}

type staged struct {
	target Target
	dir    *os.Root
	tmp    string
	base   string
}

// install writes every entry to a temporary name beside its destination, then
// renames them all into place. The window in which an install is half
// replaced is the renames alone.
func install(src io.Reader, targets map[string]Target, present map[string]bool) ([]string, error) {
	tr, err := openTar(src)
	if err != nil {
		return nil, err
	}
	var pending []staged
	defer func() {
		for _, item := range pending {
			_ = item.dir.Remove(item.tmp)
			_ = item.dir.Close()
		}
	}()
	for {
		header, nextErr := tr.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrArchive, nextErr)
		}
		if header.Name == ManifestEntry {
			continue
		}
		item, stageErr := stage(tr, header, targets[header.Name])
		if stageErr != nil {
			return nil, stageErr
		}
		pending = append(pending, item)
	}
	restored := make([]string, 0, len(pending))
	for _, item := range pending {
		if renameErr := item.dir.Rename(item.tmp, item.base); renameErr != nil {
			return restored, fmt.Errorf("replace %s: %w", item.target.Path, renameErr)
		}
		restored = append(restored, item.target.Entry)
	}
	pending = nil
	return restored, removeAbsent(targets, present)
}

func stage(tr io.Reader, header *tar.Header, target Target) (staged, error) {
	root, base, err := openParent(target.Path)
	if err != nil {
		return staged{}, fmt.Errorf("prepare %s: %w", target.Path, err)
	}
	tmp := base + tmpSuffix
	mode := fs.FileMode(privateMode)
	if target.Entry == EntryCertificate {
		mode = publicCertMode
	}
	file, openErr := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if openErr != nil {
		_ = root.Close()
		return staged{}, fmt.Errorf("create %s: %w", target.Path, openErr)
	}
	_, copyErr := io.CopyN(file, tr, header.Size)
	closeErr := file.Close()
	if failed := errors.Join(copyErr, closeErr); failed != nil {
		_ = root.Remove(tmp)
		_ = root.Close()
		return staged{}, fmt.Errorf("write %s: %w", target.Path, failed)
	}
	return staged{target: target, dir: root, tmp: tmp, base: base}, nil
}

func removeAbsent(targets map[string]Target, present map[string]bool) error {
	for entry, target := range targets {
		if present[entry] || !target.RemoveIfAbsent {
			continue
		}
		root, base, err := fsutil.RootAt(target.Path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("prepare %s: %w", target.Path, err)
		}
		removeErr := root.Remove(base)
		_ = root.Close()
		if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return fmt.Errorf("remove stale %s: %w", target.Path, removeErr)
		}
	}
	return nil
}

// openParent opens an [os.Root] on the directory that will hold p, creating
// the directory when it does not exist yet (a fresh host has no certs/).
func openParent(p string) (*os.Root, string, error) {
	dir := filepath.Dir(p)
	if err := ensureDir(dir); err != nil {
		return nil, "", err
	}
	return fsutil.RootAt(p)
}

// ensureDir creates dir through an [os.Root] on its nearest existing ancestor,
// so the creation cannot be redirected by a symlink inside the new part.
func ensureDir(dir string) error {
	ancestor := dir
	var missing []string
	for {
		if _, err := os.Stat(ancestor); err == nil {
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return fmt.Errorf("no existing ancestor of %s", dir)
		}
		missing = append([]string{filepath.Base(ancestor)}, missing...)
		ancestor = parent
	}
	if len(missing) == 0 {
		return nil
	}
	root, err := os.OpenRoot(ancestor)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return root.MkdirAll(filepath.Join(missing...), dirMode)
}
