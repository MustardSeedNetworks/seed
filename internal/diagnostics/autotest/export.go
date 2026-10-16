package autotest

// export.go renders a record as CSV and PDF (P-E2). JSON needs no renderer:
// the export is the Report, which ParseRecord reads back.

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// CSV renders r with one row per stage, each carrying the record's fields so
// a spreadsheet of many exports can be filtered by jack or asset. A record
// with no stages is one row with the stage columns empty.
func CSV(r *Report) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	record := []string{
		r.ID, r.Interface, r.JackLabel, r.AssetTag, r.StartedAt.UTC().Format(time.RFC3339),
		string(r.Verdict), strconv.FormatInt(r.DurationMs, 10),
	}
	rows := [][]string{{
		"id", "interface", "jackLabel", "assetTag", "startedAt", "verdict", "durationMs",
		"stage", "outcome", "reason", "message", "readings", "stageDurationMs",
	}}
	if len(r.Stages) == 0 {
		rows = append(rows, append(record, "", "", "", "", "", ""))
	}
	for _, st := range r.Stages {
		rows = append(rows, append(slices.Clone(record),
			string(st.Stage), string(st.Outcome), string(st.Reason), st.Message,
			readingsText(st.Readings), strconv.FormatInt(st.DurationMs, 10)))
	}
	for _, row := range rows {
		for i := range row {
			row[i] = neutralizeFormula(row[i])
		}
	}
	if err := w.WriteAll(rows); err != nil {
		return nil, fmt.Errorf("write csv: %w", err)
	}
	return buf.Bytes(), nil
}

func readingsText(rs []Reading) string {
	parts := make([]string, 0, len(rs))
	for _, rd := range rs {
		parts = append(parts, rd.Label+"="+rd.Value)
	}
	return strings.Join(parts, "; ")
}

// neutralizeFormula keeps a spreadsheet from evaluating a cell an operator
// typed (a jack label) or a check echoed (an error message) as a formula, by
// the OWASP CSV-injection rule: a leading = + - @ tab or CR gets a quote.
func neutralizeFormula(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// PDF layout, in millimetres on A4 portrait.
const (
	pdfMargin      = 15.0
	pdfLineH       = 6.0
	pdfTitleH      = 12.0
	pdfDetailH     = 4.0
	pdfLabelW      = 35.0
	pdfStageW      = 25.0
	pdfOutcomeW    = 22.0
	pdfReasonW     = 35.0
	pdfDurationW   = 22.0
	pdfTitleSize   = 18.0
	pdfHeadingSize = 12.0
	pdfBodySize    = 10.0
	pdfDetailSize  = 8.0
)

// PDF renders r as a one-page printable result: the labels first, since they
// are what ties the sheet to a port, then the verdict and every stage.
func PDF(r *Report) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(true, pdfMargin)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", pdfTitleSize)
	pdf.CellFormat(0, pdfTitleH, "AutoTest result", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", pdfBodySize)
	for _, f := range [][2]string{
		{"Jack label", r.JackLabel},
		{"Asset tag", r.AssetTag},
		{"Interface", r.Interface},
		{"Started", r.StartedAt.UTC().Format("2006-01-02 15:04:05 UTC")},
		{"Duration", formatMs(r.DurationMs)},
		{"Verdict", strings.ToUpper(string(r.Verdict))},
		{"Record", r.ID},
	} {
		pdf.CellFormat(pdfLabelW, pdfLineH, f[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(0, pdfLineH, tr(f[1]), "", 1, "L", false, 0, "")
	}

	pdf.Ln(pdfLineH)
	pdf.SetFont("Helvetica", "B", pdfHeadingSize)
	pdf.CellFormat(0, pdfLineH, "Stages", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", pdfBodySize)
	for _, h := range []struct {
		w float64
		s string
	}{{pdfStageW, "Stage"}, {pdfOutcomeW, "Outcome"}, {pdfReasonW, "Reason"}, {pdfDurationW, "Duration"}} {
		pdf.CellFormat(h.w, pdfLineH, h.s, "B", 0, "L", false, 0, "")
	}
	pdf.Ln(pdfLineH)

	for _, st := range r.Stages {
		pdf.SetFont("Helvetica", "", pdfBodySize)
		pdf.CellFormat(pdfStageW, pdfLineH, string(st.Stage), "", 0, "L", false, 0, "")
		pdf.CellFormat(pdfOutcomeW, pdfLineH, string(st.Outcome), "", 0, "L", false, 0, "")
		pdf.CellFormat(pdfReasonW, pdfLineH, string(st.Reason), "", 0, "L", false, 0, "")
		pdf.CellFormat(pdfDurationW, pdfLineH, formatMs(st.DurationMs), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", pdfDetailSize)
		for _, rd := range st.Readings {
			pdf.SetX(pdfMargin + pdfStageW)
			pdf.MultiCell(0, pdfDetailH, tr(rd.Label+": "+rd.Value), "", "L", false)
		}
		if st.Message != "" {
			pdf.SetX(pdfMargin + pdfStageW)
			pdf.MultiCell(0, pdfDetailH, tr(st.Message), "", "L", false)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("render pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func formatMs(ms int64) string {
	return (time.Duration(ms) * time.Millisecond).String()
}
