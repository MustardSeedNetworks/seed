// Package survey provides WiFi site survey functionality.
// This file implements AirMapper .amp file parsing.
package survey

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// AirMapperFile represents a parsed .amp file from NetAlly AirMapper.
type AirMapperFile struct {
	Serial            *SerialMetadata `json:"serial"`
	FloorPlan         []byte          `json:"floorPlanData"` // Raw JPEG/PNG data
	FloorPlanFilename string          `json:"floorPlanFilename"`
}

// SerialMetadata contains metadata from the .serial JSON file in an AirMapper archive.
type SerialMetadata struct {
	FileName          string         `json:"fileName"`
	FloorPlanScalePpf float64        `json:"floorPlanScalePpf"` // pixels per foot
	Propagation       float64        `json:"propagation"`
	PropagationUnit   string         `json:"propagationUnit"`
	SurveyPointCount  int            `json:"surveyPointCount"`
	SurveyItemsCount  int            `json:"surveyItemsCount"`
	Locations         *LocationsData `json:"locations,omitempty"`
	InsitesLimits     []InsitesLimit `json:"insitesLimits,omitempty"`
	Views             []ViewConfig   `json:"views,omitempty"`
}

// LocationsData contains AP and client location data.
type LocationsData struct {
	APLocations     []APLocationData     `json:"aps,omitempty"`
	ClientLocations []ClientLocationData `json:"clients,omitempty"`
}

// APLocationData represents a placed AP location from AirMapper.
type APLocationData struct {
	BSSID   string  `json:"bssid"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Label   string  `json:"label,omitempty"`
	Channel int     `json:"channel,omitempty"`
	Band    string  `json:"band,omitempty"`
}

// ClientLocationData represents a client location from AirMapper.
type ClientLocationData struct {
	MAC   string  `json:"mac"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Label string  `json:"label,omitempty"`
}

// InsitesLimit represents a pass/fail criterion from AirMapper.
type InsitesLimit struct {
	Option  string  `json:"option"`
	Name    string  `json:"name,omitempty"`
	Limit   float64 `json:"limit"`
	Suffix  string  `json:"suffix"`
	Enabled bool    `json:"enabled"`
	Mode    string  `json:"mode"`         // "passive", "active"
	AP      int     `json:"ap,omitempty"` // AP index for nth-signal tests
}

// ViewConfig represents a view configuration from AirMapper.
type ViewConfig struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

// AirMapperCalibration contains scale and propagation settings.
type AirMapperCalibration struct {
	ScaleM       float64 `json:"scaleM"`       // meters per pixel
	PropagationM float64 `json:"propagationM"` // signal propagation radius in meters
}

// AirMapperImportResult is the result of parsing an AirMapper file.
type AirMapperImportResult struct {
	FloorPlanImage    string               `json:"floorPlanImage"` // Base64 data URL
	FloorPlanFilename string               `json:"floorPlanFilename"`
	Calibration       AirMapperCalibration `json:"calibration"`
	APLocations       []APLocationData     `json:"apLocations,omitempty"`
	ClientLocations   []ClientLocationData `json:"clientLocations,omitempty"`
	PassFailCriteria  []InsitesLimit       `json:"passFailCriteria,omitempty"`
	SurveyPointCount  int                  `json:"surveyPointCount"`
	SurveyItemsCount  int                  `json:"surveyItemsCount"`
	Warnings          []string             `json:"warnings,omitempty"`
}

// ParseAirMapperFile parses an AirMapper .amp archive file.
func ParseAirMapperFile(data []byte) (*AirMapperFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip archive: %w", err)
	}

	result := &AirMapperFile{}
	var serialFound, floorPlanFound bool

	for _, file := range reader.File {
		name := file.Name
		ext := strings.ToLower(filepath.Ext(name))

		switch {
		case strings.HasSuffix(name, ".serial"):
			serial, parseErr := parseSerialFile(file)
			if parseErr != nil {
				return nil, fmt.Errorf("failed to parse .serial file: %w", parseErr)
			}
			result.Serial = serial
			serialFound = true

		case ext == ".jpg" || ext == ".jpeg" || ext == ".png":
			imgData, readErr := readZipFile(file)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read floor plan image: %w", readErr)
			}
			result.FloorPlan = imgData
			result.FloorPlanFilename = filepath.Base(name)
			floorPlanFound = true

			// .SurveyResult binary parsing is not yet implemented
			// It contains the actual survey sample data in a protobuf-like format
		}
	}

	if !serialFound {
		return nil, errors.New("no .serial file found in archive")
	}

	if !floorPlanFound {
		return nil, errors.New("no floor plan image found in archive")
	}

	return result, nil
}

// parseSerialFile parses the .serial JSON file from an AirMapper archive.
func parseSerialFile(file *zip.File) (*SerialMetadata, error) {
	data, err := readZipFile(file)
	if err != nil {
		return nil, err
	}

	var serial SerialMetadata
	if unmarshalErr := json.Unmarshal(data, &serial); unmarshalErr != nil {
		return nil, fmt.Errorf("invalid JSON in .serial file: %w", unmarshalErr)
	}

	return &serial, nil
}

// readZipFile reads the contents of a file from a ZIP archive.
func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	return io.ReadAll(rc)
}

// ToImportResult converts an AirMapperFile to an AirMapperImportResult.
func (a *AirMapperFile) ToImportResult() (*AirMapperImportResult, error) {
	if a.Serial == nil {
		return nil, errors.New("no serial metadata available")
	}

	result := &AirMapperImportResult{
		Warnings: make([]string, 0),
	}

	// Convert floor plan to base64 data URL
	if len(a.FloorPlan) > 0 {
		// Detect image type from magic bytes
		mimeType := "image/jpeg"
		if len(a.FloorPlan) > 8 && string(a.FloorPlan[:8]) == "\x89PNG\r\n\x1a\n" {
			mimeType = "image/png"
		}

		result.FloorPlanImage = fmt.Sprintf("data:%s;base64,%s",
			mimeType, base64.StdEncoding.EncodeToString(a.FloorPlan))
		result.FloorPlanFilename = a.FloorPlanFilename
	} else {
		result.Warnings = append(result.Warnings, "No floor plan image found")
	}

	// Convert scale from pixels per foot to meters per pixel
	if a.Serial.FloorPlanScalePpf > 0 {
		// ppf = pixels per foot
		// We need meters per pixel = 1 / (ppf * 3.28084) = feet_per_pixel * 0.3048
		feetPerPixel := 1.0 / a.Serial.FloorPlanScalePpf
		result.Calibration.ScaleM = feetPerPixel * 0.3048
	} else {
		result.Calibration.ScaleM = 0.1 // Default 10cm per pixel
		result.Warnings = append(result.Warnings, "No scale calibration found, using default")
	}

	// Convert propagation from feet to meters
	if a.Serial.Propagation > 0 {
		switch a.Serial.PropagationUnit {
		case "ft", "":
			result.Calibration.PropagationM = a.Serial.Propagation * 0.3048
		case "m":
			result.Calibration.PropagationM = a.Serial.Propagation
		default:
			result.Calibration.PropagationM = a.Serial.Propagation * 0.3048 // Default to feet
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf("Unknown propagation unit: %s, assuming feet", a.Serial.PropagationUnit),
			)
		}
	} else {
		result.Calibration.PropagationM = 10 // Default 10m
	}

	// Copy locations if available
	if a.Serial.Locations != nil {
		result.APLocations = a.Serial.Locations.APLocations
		result.ClientLocations = a.Serial.Locations.ClientLocations
	}

	// Copy pass/fail criteria
	result.PassFailCriteria = a.Serial.InsitesLimits

	// Copy counts
	result.SurveyPointCount = a.Serial.SurveyPointCount
	result.SurveyItemsCount = a.Serial.SurveyItemsCount

	return result, nil
}

// GetCalibration extracts calibration data from an AirMapper file.
func (a *AirMapperFile) GetCalibration() AirMapperCalibration {
	cal := AirMapperCalibration{
		ScaleM:       0.1, // Default
		PropagationM: 10,  // Default
	}

	if a.Serial == nil {
		return cal
	}

	// Convert scale
	if a.Serial.FloorPlanScalePpf > 0 {
		feetPerPixel := 1.0 / a.Serial.FloorPlanScalePpf
		cal.ScaleM = feetPerPixel * 0.3048
	}

	// Convert propagation
	if a.Serial.Propagation > 0 {
		if a.Serial.PropagationUnit == "m" {
			cal.PropagationM = a.Serial.Propagation
		} else {
			cal.PropagationM = a.Serial.Propagation * 0.3048
		}
	}

	return cal
}
