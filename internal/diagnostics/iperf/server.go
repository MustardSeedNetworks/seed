package iperf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/validation"

	"github.com/MustardSeedNetworks/seed/internal/platform/procrun"
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

	// No deadline: the context lives as long as the server and exists so
	// StopServer can cancel it. A start deadline here used to kill every
	// server ten seconds after the operator started it (seed#3343).
	ctx, cancel := context.WithCancel(context.Background())
	m.serverCancel = cancel

	// --forceflush, because into a pipe iperf3 otherwise buffers its stdout
	// and the listening banner would not arrive until the buffer filled.
	listening := newListenWatcher()
	cmd := exec.CommandContext(ctx, binaryPath, "-s", "-p", strconv.Itoa(port), "--forceflush")
	cmd.Stdout = listening
	if startErr := procrun.Start(cmd); startErr != nil {
		cancel()
		return fmt.Errorf("failed to start iperf3 server: %w", startErr)
	}

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	// Readiness comes from iperf3 itself, never from dialing the port: iperf3
	// serves one test at a time and takes any connection as a client, so a
	// probe ties it up and the operator's first client can be refused while
	// it resets (seed#3342).
	select {
	case <-listening.seen:
	case waitErr := <-exited:
		cancel()
		if waitErr == nil {
			return errors.New("iperf3 server exited before listening")
		}
		return fmt.Errorf("iperf3 server exited before listening: %w", waitErr)
	case <-time.After(serverListenTimeout):
		_ = cmd.Process.Kill()
		<-exited
		cancel()
		return fmt.Errorf("iperf3 server failed to start listening on port %d within %v", port, serverListenTimeout)
	}

	m.serverCmd = cmd
	m.serverStatus = ServerStatus{
		Running: true,
		Port:    port,
		PID:     cmd.Process.Pid,
	}

	// Monitor the server process
	go func() {
		waitErr := <-exited

		m.mu.RLock()
		phase := m.monitorPhase
		m.mu.RUnlock()
		if phase != nil {
			phase("waited")
			defer phase("done")
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		// Only report on the server this goroutine was started for.
		//
		// StopServer kills the process, which makes Wait return here, and the
		// operator's next StartServer can publish Running: true before this
		// goroutine gets the lock. Without this check the stale monitor then
		// clears the status of the server that replaced it: StopServer answers
		// "server not running" while an iperf3 process is very much running and
		// holding the port, and the UI shows stopped for a server that is up.
		if m.serverCmd != cmd {
			return
		}

		m.serverStatus.Running = false
		if waitErr != nil && ctx.Err() == nil {
			m.serverStatus.Error = waitErr.Error()
		}
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

// listeningBanner is what iperf3 -s prints once its control socket is
// listening; it writes the line straight after listen().
const listeningBanner = "Server listening on"

// listenBannerSearchLimit bounds how much output is searched for the banner.
// iperf3 prints it within its first hundred bytes.
const listenBannerSearchLimit = 4096

// listenWatcher is the iperf3 server's stdout. It closes seen when the
// listening banner appears and discards all output.
type listenWatcher struct {
	head    []byte
	seen    chan struct{}
	stopped bool
}

func newListenWatcher() *listenWatcher {
	return &listenWatcher{seen: make(chan struct{})}
}

// Write is only called from the one goroutine [exec.Cmd] copies stdout on.
func (w *listenWatcher) Write(p []byte) (int, error) {
	if w.stopped {
		return len(p), nil
	}

	w.head = append(w.head, p...)
	switch {
	case bytes.Contains(w.head, []byte(listeningBanner)):
		close(w.seen)
		w.stopped, w.head = true, nil
	case len(w.head) > listenBannerSearchLimit:
		w.stopped, w.head = true, nil
	}

	return len(p), nil
}
