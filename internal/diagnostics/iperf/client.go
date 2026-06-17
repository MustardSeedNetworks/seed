package iperf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// RunClient runs an iperf3 client test.
func (m *Manager) RunClient(ctx context.Context, config *ClientConfig) (*Result, error) {
	if err := validateServer(config.Server); err != nil {
		return nil, err
	}
	config.Protocol = strings.ToLower(config.Protocol)
	config.Direction = strings.ToLower(config.Direction)

	if err := m.startClientTest(); err != nil {
		return nil, err
	}
	defer m.resetClientStatus()

	binaryPath, err := findIperf3Binary()
	if err != nil {
		return nil, err
	}

	setClientDefaults(config)
	direction := normalizeDirection(config)
	args := buildClientArgs(config, direction)

	m.updateClientProgress("testing", progressWarningThreshold)

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		var iperfOut iperfJSON
		if jsonErr := json.Unmarshal(output, &iperfOut); jsonErr == nil && iperfOut.Error != "" {
			return nil, fmt.Errorf("iperf3 error: %s", iperfOut.Error)
		}
		return nil, fmt.Errorf("iperf3 failed: %w", err)
	}

	m.updateClientProgress("parsing", progressCriticalThreshold)

	var iperfOut iperfJSON
	if unmarshalErr := json.Unmarshal(output, &iperfOut); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse iperf3 output: %w", unmarshalErr)
	}
	if iperfOut.Error != "" {
		return nil, fmt.Errorf("iperf3 error: %s", iperfOut.Error)
	}

	result := parseClientResult(&iperfOut, config, direction)

	m.mu.Lock()
	m.lastResult = result
	m.clientStatus.Phase = "complete"
	m.clientStatus.Progress = progressMaxPercent
	m.mu.Unlock()

	return result, nil
}

func (m *Manager) startClientTest() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.clientStatus.Running {
		return errors.New("test already in progress")
	}
	m.clientStatus = ClientStatus{Running: true, Phase: "connecting", Progress: progressConnectingPercent}
	return nil
}

func (m *Manager) resetClientStatus() {
	m.mu.Lock()
	m.clientStatus = ClientStatus{Running: false, Phase: "idle", Progress: 0}
	m.mu.Unlock()
}

func (m *Manager) updateClientProgress(phase string, progress float64) {
	m.mu.Lock()
	m.clientStatus.Phase = phase
	m.clientStatus.Progress = progress
	m.mu.Unlock()
}

func setClientDefaults(config *ClientConfig) {
	if config.Port == 0 {
		config.Port = 5201
	}
	if config.Duration == 0 {
		config.Duration = 10
	}
	if config.Parallel == 0 {
		config.Parallel = 1
	}
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}
}

func normalizeDirection(config *ClientConfig) string {
	direction := strings.ToLower(config.Direction)
	if direction == "" {
		if config.Reverse {
			direction = directionDownload
		} else {
			direction = directionUpload
		}
	}
	if direction != directionDownload && direction != directionBidirectional {
		direction = directionUpload
	}
	config.Direction = direction
	return direction
}

func buildClientArgs(config *ClientConfig, direction string) []string {
	args := []string{
		"-c", config.Server,
		"-p", strconv.Itoa(config.Port),
		"-t", strconv.Itoa(config.Duration),
		"-P", strconv.Itoa(config.Parallel),
		"-J",
	}
	if config.Protocol == "udp" {
		args = append(args, "-u", "-b", "0")
	}
	switch direction {
	case directionDownload:
		config.Reverse = true
		args = append(args, "-R")
	case directionBidirectional:
		args = append(args, "--bidir")
		config.Reverse = false
	default:
		config.Reverse = false
	}
	return args
}

func parseClientResult(iperfOut *iperfJSON, config *ClientConfig, direction string) *Result {
	result := &Result{
		Protocol: config.Protocol, Server: config.Server, Port: config.Port,
		Timestamp: time.Now(), Direction: direction,
	}
	switch direction {
	case directionDownload:
		result.BitsPerSecond = iperfOut.End.SumReceived.BitsPerSecond
		result.Bandwidth = iperfOut.End.SumReceived.BitsPerSecond / bytesToMegabits
		result.Transfer = iperfOut.End.SumReceived.Bytes / bytesToMegabits
		result.Duration = iperfOut.End.SumReceived.Seconds
	case directionBidirectional:
		result.DownloadBitsPerSecond = iperfOut.End.SumReceived.BitsPerSecond
		result.DownloadBandwidth = iperfOut.End.SumReceived.BitsPerSecond / bytesToMegabits
		result.DownloadTransfer = iperfOut.End.SumReceived.Bytes / bytesToMegabits
		result.UploadBitsPerSecond = iperfOut.End.SumSent.BitsPerSecond
		result.UploadBandwidth = iperfOut.End.SumSent.BitsPerSecond / bytesToMegabits
		result.UploadTransfer = iperfOut.End.SumSent.Bytes / bytesToMegabits
		result.BitsPerSecond = result.DownloadBitsPerSecond
		result.Bandwidth = result.DownloadBandwidth
		result.Transfer = result.DownloadTransfer
		result.Duration = iperfOut.End.Sum.Seconds
		result.Retransmits = iperfOut.End.SumSent.Retransmits
	default:
		result.BitsPerSecond = iperfOut.End.SumSent.BitsPerSecond
		result.Bandwidth = iperfOut.End.SumSent.BitsPerSecond / bytesToMegabits
		result.Transfer = iperfOut.End.SumSent.Bytes / bytesToMegabits
		result.Duration = iperfOut.End.SumSent.Seconds
		result.Retransmits = iperfOut.End.SumSent.Retransmits
	}
	if config.Protocol == "udp" {
		result.Jitter = iperfOut.End.Sum.JitterMs
		result.LostPackets = iperfOut.End.Sum.LostPackets
		result.LostPercent = iperfOut.End.Sum.LostPercent
	}
	return result
}
