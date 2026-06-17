package iperf

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// StartServer starts the iperf3 server.
func (m *Manager) StartServer(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.serverStatus.Running {
		return fmt.Errorf("server already running on port %d", m.serverStatus.Port)
	}

	// Validate port number
	if err := validation.ValidatePort(port); err != nil {
		return err
	}

	binaryPath, err := findIperf3Binary()
	if err != nil {
		return err
	}

	// Use timeout context for server startup monitoring
	ctx, cancel := context.WithTimeout(context.Background(), serverStartTimeout)
	m.serverCancel = cancel

	// Start iperf3 server: iperf3 -s -p <port>

	cmd := exec.CommandContext(ctx, binaryPath, "-s", "-p", strconv.Itoa(port))
	if startErr := cmd.Start(); startErr != nil {
		cancel()
		return fmt.Errorf("failed to start iperf3 server: %w", startErr)
	}

	// Wait for port to be ready
	if portErr := waitForPortReady(port, portCheckTimeout); portErr != nil {
		// Kill the process if port check fails
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		cancel()
		return fmt.Errorf("iperf3 server failed to start listening: %w", portErr)
	}

	m.serverCmd = cmd
	m.serverStatus = ServerStatus{
		Running: true,
		Port:    port,
		PID:     cmd.Process.Pid,
	}

	// Monitor the server process
	go func() {
		waitErr := cmd.Wait()
		m.mu.Lock()
		m.serverStatus.Running = false
		if waitErr != nil && ctx.Err() == nil {
			m.serverStatus.Error = waitErr.Error()
		}
		m.mu.Unlock()
	}()

	return nil
}

// StopServer stops the iperf3 server.
func (m *Manager) StopServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.serverStatus.Running {
		return errors.New("server not running")
	}

	if m.serverCancel != nil {
		m.serverCancel()
	}

	if m.serverCmd != nil && m.serverCmd.Process != nil {
		if err := m.serverCmd.Process.Kill(); err != nil {
			// Log the error, but don't fail, as we are trying to stop the server
			// and it might already be dead or unreachable.
			logging.GetLogger().
				Warn("Error killing iperf3 server process", "pid", m.serverCmd.Process.Pid, "error", err)
		}
	}

	m.serverStatus = ServerStatus{Running: false}
	m.serverCmd = nil
	m.serverCancel = nil

	return nil
}
