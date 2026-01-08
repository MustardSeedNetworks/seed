package validation_test

import (
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/validation"
)

func TestValidateHTTPEndpointNameTooLong(t *testing.T) {
	endpoint := validation.HTTPEndpointRequest{
		Name:           strings.Repeat("a", 101),
		URL:            "https://example.com",
		ExpectedStatus: 200,
	}
	errors := validation.ValidateHTTPEndpoint(&endpoint)
	if len(errors) != 1 {
		t.Errorf("ValidateHTTPEndpoint() returned %d errors, want 1", len(errors))
	}
	if errors[0].Field != "name" {
		t.Errorf("ValidateHTTPEndpoint() error field = %q, want %q", errors[0].Field, "name")
	}
}

func TestValidatePingTargetNameTooLong(t *testing.T) {
	target := validation.PingTargetRequest{
		Name: strings.Repeat("a", 101),
		Host: "8.8.8.8",
	}
	errors := validation.ValidatePingTarget(&target)
	if len(errors) != 1 {
		t.Errorf("ValidatePingTarget() returned %d errors, want 1", len(errors))
	}
	if errors[0].Field != "name" {
		t.Errorf("ValidatePingTarget() error field = %q, want %q", errors[0].Field, "name")
	}
}

func TestValidateTCPPortNameTooLong(t *testing.T) {
	target := validation.TCPPortRequest{
		Name: strings.Repeat("a", 101),
		Host: "example.com",
		Port: 80,
	}
	errors := validation.ValidateTCPPort(&target)
	if len(errors) != 1 {
		t.Errorf("ValidateTCPPort() returned %d errors, want 1", len(errors))
	}
	if errors[0].Field != "name" {
		t.Errorf("ValidateTCPPort() error field = %q, want %q", errors[0].Field, "name")
	}
}

func TestValidateTCPPortEmptyName(t *testing.T) {
	target := validation.TCPPortRequest{
		Name: "",
		Host: "example.com",
		Port: 80,
	}
	errors := validation.ValidateTCPPort(&target)
	if len(errors) != 1 {
		t.Errorf("ValidateTCPPort() returned %d errors, want 1", len(errors))
	}
	if errors[0].Field != "name" {
		t.Errorf("ValidateTCPPort() error field = %q, want %q", errors[0].Field, "name")
	}
}

func TestValidateTCPPortEmptyHost(t *testing.T) {
	target := validation.TCPPortRequest{
		Name: "Test",
		Host: "",
		Port: 80,
	}
	errors := validation.ValidateTCPPort(&target)
	if len(errors) != 1 {
		t.Errorf("ValidateTCPPort() returned %d errors, want 1", len(errors))
	}
	if errors[0].Field != "host" {
		t.Errorf("ValidateTCPPort() error field = %q, want %q", errors[0].Field, "host")
	}
}

func TestValidateThresholdBothNegative(t *testing.T) {
	errors := validation.ValidateThreshold("latency", -100*time.Millisecond, -50*time.Millisecond)
	if len(errors) != 2 {
		t.Errorf("ValidateThreshold() returned %d errors, want 2", len(errors))
	}
}

func TestSafeHTTPClientCheckRedirect(t *testing.T) {
	client := validation.SafeHTTPClient(10 * time.Second)

	// The CheckRedirect function should be set
	if client.CheckRedirect == nil {
		t.Error("SafeHTTPClient should have CheckRedirect set")
	}

	// The timeout should be set
	if client.Timeout != 10*time.Second {
		t.Errorf("SafeHTTPClient timeout = %v, want %v", client.Timeout, 10*time.Second)
	}
}
