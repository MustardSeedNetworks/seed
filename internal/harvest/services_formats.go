package harvest

// services_formats.go contains the non-PDF report renderers: HTML (a single
// inline-styled template), CSV (label/value rows), and JSON.

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// generateHTML creates an HTML report.
func (s *GeneratorService) generateHTML(report *Report, data *AggregatedData) ([]byte, error) {
	html := fmt.Sprintf(
		`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; border-bottom: 2px solid #d4a017; padding-bottom: 10px; }
        h2 { color: #666; margin-top: 30px; }
        .metric { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee; }
        .metric-label { color: #666; }
        .metric-value { font-weight: bold; }
        .severity-critical { color: #dc3545; }
        .severity-high { color: #ff8000; }
        .severity-medium { color: #ffc107; }
        .severity-low { color: #28a745; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <p>Generated: %s</p>
    <p>Period: %s to %s</p>

    <h2>Executive Summary</h2>
    <div class="metric"><span class="metric-label">Total Devices</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label">Total Vulnerabilities</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label">Average Latency</span><span class="metric-value">%.2f ms</span></div>
    <div class="metric"><span class="metric-label">Uptime</span><span class="metric-value">%.2f%%</span></div>

    <h2>Vulnerability Summary</h2>
    <div class="metric"><span class="metric-label severity-critical">Critical</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label severity-high">High</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label severity-medium">Medium</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label severity-low">Low</span><span class="metric-value">%d</span></div>

    <h2>Performance Metrics</h2>
    <div class="metric"><span class="metric-label">Average Latency</span><span class="metric-value">%.2f ms</span></div>
    <div class="metric"><span class="metric-label">Packet Loss</span><span class="metric-value">%.2f%%</span></div>
    <div class="metric"><span class="metric-label">Bandwidth</span><span class="metric-value">%.2f Mbps</span></div>
</body>
</html>`,
		report.Name,
		report.Name,
		time.Now().Format("January 2, 2006 15:04"),
		data.StartDate.Format("Jan 2"),
		data.EndDate.Format("Jan 2, 2006"),
		data.DeviceCount,
		data.VulnCount.Total,
		data.Performance.AvgLatencyMs,
		data.Performance.UptimePercent,
		data.VulnCount.Critical,
		data.VulnCount.High,
		data.VulnCount.Medium,
		data.VulnCount.Low,
		data.Performance.AvgLatencyMs,
		data.Performance.AvgPacketLoss,
		data.Performance.AvgBandwidthMbps,
	)

	return []byte(html), nil
}

// generateCSV creates a CSV report.
func (s *GeneratorService) generateCSV(report *Report, data *AggregatedData) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Header
	_ = writer.Write([]string{"Metric", "Value"})

	// Data rows
	rows := [][]string{
		{"Report Name", report.Name},
		{"Generated", time.Now().Format(time.RFC3339)},
		{"Period Start", data.StartDate.Format(time.RFC3339)},
		{"Period End", data.EndDate.Format(time.RFC3339)},
		{"Total Devices", strconv.Itoa(data.DeviceCount)},
		{"Total Vulnerabilities", strconv.Itoa(data.VulnCount.Total)},
		{"Critical Vulnerabilities", strconv.Itoa(data.VulnCount.Critical)},
		{"High Vulnerabilities", strconv.Itoa(data.VulnCount.High)},
		{"Medium Vulnerabilities", strconv.Itoa(data.VulnCount.Medium)},
		{"Low Vulnerabilities", strconv.Itoa(data.VulnCount.Low)},
		{"Average Latency (ms)", fmt.Sprintf("%.2f", data.Performance.AvgLatencyMs)},
		{"Packet Loss (%)", fmt.Sprintf("%.2f", data.Performance.AvgPacketLoss)},
		{"Bandwidth (Mbps)", fmt.Sprintf("%.2f", data.Performance.AvgBandwidthMbps)},
		{"Uptime (%)", fmt.Sprintf("%.2f", data.Performance.UptimePercent)},
	}

	for _, row := range rows {
		_ = writer.Write(row)
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// generateJSON creates a JSON report.
func (s *GeneratorService) generateJSON(report *Report, data *AggregatedData) ([]byte, error) {
	output := map[string]any{
		"report": map[string]any{
			"id":        report.ID,
			"name":      report.Name,
			"type":      report.Type,
			"generated": time.Now().Format(time.RFC3339),
		},
		"data": data,
	}

	result, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON report: %w", err)
	}
	return result, nil
}
