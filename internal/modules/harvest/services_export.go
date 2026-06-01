package harvest

// services_export.go contains the bulk data-export path on GeneratorService:
// Export dispatches on req.Type (devices / vulnerabilities), queries the
// database, then serializes the result via JSON or CSV writers.

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Export exports data in the specified format.
func (s *GeneratorService) Export(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	if req == nil {
		return nil, errors.New("export request is nil")
	}

	// Create export ID
	exportID := uuid.New().String()

	// Query data based on type
	var data any
	var recordCount int
	var err error

	switch req.Type {
	case entityDevices:
		data, recordCount, err = s.exportDevices(ctx, req)
	case "vulnerabilities":
		data, recordCount, err = s.exportVulnerabilities(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported export type: %s", req.Type)
	}

	if err != nil {
		return nil, err
	}

	// Generate export file
	var content []byte
	switch req.Format {
	case FormatJSON:
		content, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling export data: %w", err)
		}
	case FormatCSV:
		content, err = s.dataToCSV(data)
	case FormatPDF, FormatHTML, FormatExcel, FormatMarkdown:
		err = fmt.Errorf("unsupported export format: %s", req.Format)
	}

	if err != nil {
		return nil, err
	}

	// Save file
	filename := fmt.Sprintf("export-%s.%s", exportID, req.Format)
	filepath := filepath.Join(s.reportsPath, filename)

	if mkdirErr := os.MkdirAll(s.reportsPath, 0o750); mkdirErr != nil {
		return nil, fmt.Errorf("creating export directory: %w", mkdirErr)
	}
	if writeErr := os.WriteFile(filepath, content, 0o600); writeErr != nil {
		return nil, fmt.Errorf("writing export file: %w", writeErr)
	}

	return &ExportResult{
		ID:          exportID,
		FilePath:    filepath,
		FileSize:    int64(len(content)),
		RecordCount: recordCount,
		Format:      req.Format,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(DefaultReportTTL),
	}, nil
}

func (s *GeneratorService) exportDevices(ctx context.Context, _ *ExportRequest) (any, int, error) {
	devices, err := s.export.ExportDevices(ctx)
	if err != nil {
		return nil, 0, err
	}
	return devices, len(devices), nil
}

func (s *GeneratorService) exportVulnerabilities(
	ctx context.Context,
	_ *ExportRequest,
) (any, int, error) {
	vulns, err := s.export.ExportVulnerabilities(ctx)
	if err != nil {
		return nil, 0, err
	}
	return vulns, len(vulns), nil
}

func (s *GeneratorService) dataToCSV(data any) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if d, ok := data.([]map[string]any); ok && len(d) > 0 {
		// Get headers from first record
		var headers []string
		for k := range d[0] {
			headers = append(headers, k)
		}
		sort.Strings(headers)
		_ = writer.Write(headers)

		// Write rows
		for _, record := range d {
			var row []string
			for _, h := range headers {
				row = append(row, fmt.Sprintf("%v", record[h]))
			}
			_ = writer.Write(row)
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}
