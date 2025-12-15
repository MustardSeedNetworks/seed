package api

import (
	"testing"
	"time"
)

// TestHubClientCount tests the ClientCount method
func TestHubClientCount(t *testing.T) {
	hub := NewHub()
	
	if count := hub.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}

// TestHubBroadcast tests the Broadcast method
func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	
	msg := Message{
		Type: "test",
		Payload: map[string]string{"key": "value"},
	}
	
	// Should not panic with no clients
	hub.Broadcast(msg)
}

// TestHubBroadcastCardUpdate tests the BroadcastCardUpdate method
func TestHubBroadcastCardUpdate(t *testing.T) {
	hub := NewHub()
	
	// Should not panic with no clients
	hub.BroadcastCardUpdate("testCard", map[string]string{"status": "ok"})
}

// TestHubShutdown tests graceful shutdown
func TestHubShutdown(t *testing.T) {
	hub := NewHub()
	
	// Start the hub
	go hub.Run()
	
	// Give it time to start
	time.Sleep(10 * time.Millisecond)
	
	// Shutdown should not panic
	hub.Shutdown()
	
	// Give it time to shutdown
	time.Sleep(10 * time.Millisecond)
}

// TestNewHub tests hub creation
func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	// Verify hub has correct initial state
	if count := hub.ClientCount(); count != 0 {
		t.Errorf("Expected new hub to have 0 clients, got %d", count)
	}
}
