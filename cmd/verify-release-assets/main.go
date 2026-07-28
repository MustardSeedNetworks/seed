package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	sha256HexLength    = 64
	versionPlaceholder = "{version}"
)

type releaseManifest struct {
	PrimaryAssets []primaryAsset `json:"primaryAssets"`
}

type primaryAsset struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	Format       string `json:"format"`
	NameTemplate string `json:"nameTemplate"`
}

type artifact struct {
	Name   string        `json:"name"`
	Type   string        `json:"type"`
	GOOS   string        `json:"goos"`
	GOARCH string        `json:"goarch"`
	Extra  artifactExtra `json:"extra"`
}

type artifactExtra struct {
	Checksum string `json:"Checksum"`
	Format   string `json:"Format"`
	ID       string `json:"ID"`
}

type subject struct {
	Name     string
	Checksum string
}

func main() {
	rootPath := flag.String("root", ".", "release workspace root")
	manifestPath := flag.String(
		"manifest",
		"release/asset-manifest.json",
		"frozen release asset manifest",
	)
	artifactsPath := flag.String("artifacts", "dist/artifacts.json", "GoReleaser artifact catalog")
	checksumsPath := flag.String("checksums", "dist/checksums.txt", "GoReleaser checksum file")
	distPath := flag.String("dist", "dist", "GoReleaser output directory")
	subjectsPath := flag.String(
		"subjects",
		"dist/provenance-subjects.txt",
		"validated provenance subjects output",
	)
	flag.Parse()

	if err := run(
		*rootPath,
		*manifestPath,
		*artifactsPath,
		*checksumsPath,
		*distPath,
		*subjectsPath,
	); err != nil {
		fmt.Fprintf(os.Stderr, "release asset verification failed: %v\n", err)
		os.Exit(1)
	}
}

func run(
	rootPath, manifestPath, artifactsPath, checksumsPath, distPath, subjectsPath string,
) error {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return fmt.Errorf("open release workspace: %w", err)
	}
	defer root.Close()

	manifest, err := readManifest(root.FS(), manifestPath)
	if err != nil {
		return err
	}
	catalog, err := readArtifacts(root.FS(), artifactsPath)
	if err != nil {
		return err
	}
	checksums, err := readChecksums(root.FS(), checksumsPath)
	if err != nil {
		return err
	}
	files, err := listFiles(root.FS(), distPath)
	if err != nil {
		return err
	}

	subjects, err := verifyRelease(manifest, catalog, checksums, files)
	if err != nil {
		return err
	}
	writeErr := writeSubjects(root, subjectsPath, subjects)
	if writeErr != nil {
		return writeErr
	}

	fmt.Fprintf(
		os.Stdout,
		"verified %d primary release assets; provenance subjects: %s\n",
		len(subjects),
		subjectsPath,
	)
	return nil
}

func readManifest(filesystem fs.FS, path string) (releaseManifest, error) {
	file, err := filesystem.Open(path)
	if err != nil {
		return releaseManifest{}, fmt.Errorf("open release manifest: %w", err)
	}
	defer file.Close()

	var manifest releaseManifest
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	decodeErr := decoder.Decode(&manifest)
	if decodeErr != nil {
		return releaseManifest{}, fmt.Errorf("decode release manifest: %w", decodeErr)
	}
	eofErr := requireEOF(decoder)
	if eofErr != nil {
		return releaseManifest{}, fmt.Errorf("decode release manifest: %w", eofErr)
	}
	validationErr := validateManifest(manifest)
	if validationErr != nil {
		return releaseManifest{}, validationErr
	}
	return manifest, nil
}

func readArtifacts(filesystem fs.FS, path string) ([]artifact, error) {
	file, err := filesystem.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open GoReleaser artifacts: %w", err)
	}
	defer file.Close()

	var catalog []artifact
	decoder := json.NewDecoder(file)
	decodeErr := decoder.Decode(&catalog)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode GoReleaser artifacts: %w", decodeErr)
	}
	eofErr := requireEOF(decoder)
	if eofErr != nil {
		return nil, fmt.Errorf("decode GoReleaser artifacts: %w", eofErr)
	}
	return catalog, nil
}

func requireEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("unexpected trailing JSON value")
}

func readChecksums(filesystem fs.FS, path string) (map[string]string, error) {
	file, err := filesystem.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open checksums: %w", err)
	}
	defer file.Close()

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || !validSHA256(fields[0]) {
			return nil, fmt.Errorf("invalid checksum line %q", scanner.Text())
		}
		if _, exists := checksums[fields[1]]; exists {
			return nil, fmt.Errorf("duplicate checksum for %q", fields[1])
		}
		checksums[fields[1]] = fields[0]
	}
	scanErr := scanner.Err()
	if scanErr != nil {
		return nil, fmt.Errorf("read checksums: %w", scanErr)
	}
	return checksums, nil
}

func listFiles(filesystem fs.FS, path string) (map[string]struct{}, error) {
	entries, err := fs.ReadDir(filesystem, path)
	if err != nil {
		return nil, fmt.Errorf("read release directory: %w", err)
	}
	files := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			files[entry.Name()] = struct{}{}
		}
	}
	return files, nil
}

func validateManifest(manifest releaseManifest) error {
	if len(manifest.PrimaryAssets) == 0 {
		return errors.New("release manifest has no primary assets")
	}
	seen := make(map[string]struct{}, len(manifest.PrimaryAssets))
	for _, expected := range manifest.PrimaryAssets {
		if expected.ID == "" || expected.Type == "" || expected.GOOS == "" ||
			expected.GOARCH == "" || expected.Format == "" {
			return fmt.Errorf(
				"release manifest entry %q has empty identity fields",
				expected.NameTemplate,
			)
		}
		if strings.Count(expected.NameTemplate, versionPlaceholder) != 1 {
			return fmt.Errorf(
				"release manifest entry %q must contain one %s placeholder",
				expected.NameTemplate,
				versionPlaceholder,
			)
		}
		if _, exists := seen[expected.NameTemplate]; exists {
			return fmt.Errorf("release manifest has duplicate entry %q", expected.NameTemplate)
		}
		seen[expected.NameTemplate] = struct{}{}
	}
	return nil
}

func verifyRelease(
	manifest releaseManifest,
	catalog []artifact,
	checksums map[string]string,
	files map[string]struct{},
) ([]subject, error) {
	validationErr := validateManifest(manifest)
	if validationErr != nil {
		return nil, validationErr
	}
	covered, err := selectPrimaryAssets(manifest, catalog)
	if err != nil {
		return nil, err
	}
	return verifyCoverage(covered, checksums, files)
}

func selectPrimaryAssets(manifest releaseManifest, catalog []artifact) ([]artifact, error) {
	primary := make([]artifact, 0, len(manifest.PrimaryAssets))
	for _, candidate := range catalog {
		if candidate.Type == "Archive" || candidate.Type == "Linux Package" {
			primary = append(primary, candidate)
		}
	}
	matched := make([]bool, len(primary))
	versions := make(map[string]struct{})
	covered := make([]artifact, 0, len(manifest.PrimaryAssets))

	for _, expected := range manifest.PrimaryAssets {
		match, version, err := findPrimaryMatch(expected, primary, matched)
		if err != nil {
			return nil, err
		}
		matched[match] = true
		versions[version] = struct{}{}
		covered = append(covered, primary[match])
	}

	for index, candidate := range primary {
		if !matched[index] {
			return nil, fmt.Errorf(
				"orphan primary asset %q is not in the frozen manifest",
				candidate.Name,
			)
		}
	}
	if len(versions) != 1 {
		return nil, fmt.Errorf(
			"primary assets contain %d release versions, want one",
			len(versions),
		)
	}
	return covered, nil
}

func findPrimaryMatch(
	expected primaryAsset,
	primary []artifact,
	matched []bool,
) (int, string, error) {
	match := -1
	version := ""
	for index, candidate := range primary {
		if matched[index] {
			continue
		}
		candidateVersion, ok := matches(expected, candidate)
		if !ok {
			continue
		}
		if match != -1 {
			return 0, "", fmt.Errorf("multiple primary assets match %q", expected.NameTemplate)
		}
		match = index
		version = candidateVersion
	}
	if match == -1 {
		return 0, "", fmt.Errorf("missing primary asset %q", expected.NameTemplate)
	}
	return match, version, nil
}

func verifyCoverage(
	covered []artifact,
	checksums map[string]string,
	files map[string]struct{},
) ([]subject, error) {
	subjects := make([]subject, 0, len(covered))
	expectedPrimary := make(map[string]struct{}, len(covered))
	expectedSBOMs := make(map[string]struct{}, len(covered))
	for _, candidate := range covered {
		if _, assetExists := files[candidate.Name]; !assetExists {
			return nil, fmt.Errorf("missing primary asset file %q", candidate.Name)
		}
		artifactChecksum, hasPrefix := strings.CutPrefix(candidate.Extra.Checksum, "sha256:")
		if !hasPrefix || !validSHA256(artifactChecksum) {
			return nil, fmt.Errorf(
				"primary asset %q has invalid artifact checksum %q",
				candidate.Name,
				candidate.Extra.Checksum,
			)
		}
		checksum, checksumExists := checksums[candidate.Name]
		if !checksumExists {
			return nil, fmt.Errorf("missing checksum for primary asset %q", candidate.Name)
		}
		if checksum != artifactChecksum {
			return nil, fmt.Errorf("checksum mismatch for primary asset %q", candidate.Name)
		}
		if _, signatureExists := files[candidate.Name+".cosign.bundle"]; !signatureExists {
			return nil, fmt.Errorf("missing Cosign bundle for primary asset %q", candidate.Name)
		}
		if _, sbomExists := files[candidate.Name+".sbom.json"]; !sbomExists {
			return nil, fmt.Errorf("missing SBOM for primary asset %q", candidate.Name)
		}
		expectedPrimary[candidate.Name] = struct{}{}
		expectedSBOMs[candidate.Name+".sbom.json"] = struct{}{}
		subjects = append(subjects, subject{Name: candidate.Name, Checksum: artifactChecksum})
	}
	if err := verifyNoOrphanSidecars(files, expectedPrimary, expectedSBOMs); err != nil {
		return nil, err
	}

	sort.Slice(subjects, func(i, j int) bool { return subjects[i].Name < subjects[j].Name })
	return subjects, nil
}

func verifyNoOrphanSidecars(files, expectedPrimary, expectedSBOMs map[string]struct{}) error {
	for name := range files {
		if strings.HasSuffix(name, ".sbom.json") {
			if _, ok := expectedSBOMs[name]; !ok {
				return fmt.Errorf("orphan SBOM %q has no primary asset", name)
			}
		}
		if !strings.HasSuffix(name, ".cosign.bundle") {
			continue
		}
		signed := strings.TrimSuffix(name, ".cosign.bundle")
		if signed == "checksums.txt" {
			continue
		}
		if _, ok := expectedPrimary[signed]; ok {
			continue
		}
		if _, ok := expectedSBOMs[signed]; !ok {
			return fmt.Errorf("orphan Cosign bundle %q has no release artifact", name)
		}
	}
	return nil
}

func matches(expected primaryAsset, candidate artifact) (string, bool) {
	if expected.Type != candidate.Type || expected.GOOS != candidate.GOOS ||
		expected.GOARCH != candidate.GOARCH || expected.ID != candidate.Extra.ID {
		return "", false
	}
	format := strings.TrimPrefix(candidate.Extra.Format, ".")
	if format == "" {
		format = formatFromName(candidate.Name)
	}
	if expected.Format != format {
		return "", false
	}

	prefix, suffix, _ := strings.Cut(expected.NameTemplate, versionPlaceholder)
	if !strings.HasPrefix(candidate.Name, prefix) || !strings.HasSuffix(candidate.Name, suffix) {
		return "", false
	}
	version := strings.TrimSuffix(strings.TrimPrefix(candidate.Name, prefix), suffix)
	if version == "" || strings.ContainsAny(version, `/\\`) {
		return "", false
	}
	return version, true
}

func formatFromName(name string) string {
	switch {
	case strings.HasSuffix(name, ".tar.gz"):
		return "tar.gz"
	case strings.HasSuffix(name, ".zip"):
		return "zip"
	case strings.HasSuffix(name, ".deb"):
		return "deb"
	case strings.HasSuffix(name, ".rpm"):
		return "rpm"
	default:
		return ""
	}
}

func validSHA256(value string) bool {
	if len(value) != sha256HexLength {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func writeSubjects(root *os.Root, path string, subjects []subject) error {
	file, err := root.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("create provenance subjects: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, item := range subjects {
		_, writeErr := fmt.Fprintf(writer, "%s  %s\n", item.Checksum, item.Name)
		if writeErr != nil {
			return fmt.Errorf("write provenance subjects: %w", writeErr)
		}
	}
	flushErr := writer.Flush()
	if flushErr != nil {
		return fmt.Errorf("flush provenance subjects: %w", flushErr)
	}
	return nil
}
