// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/krisarmstrong/netscope/internal/auth"
	"github.com/krisarmstrong/netscope/internal/config"
	"github.com/krisarmstrong/netscope/internal/dhcp"
	"github.com/krisarmstrong/netscope/internal/discovery"
	"github.com/krisarmstrong/netscope/internal/dns"
	"github.com/krisarmstrong/netscope/internal/gateway"
	"github.com/krisarmstrong/netscope/internal/network"
	"github.com/krisarmstrong/netscope/internal/vlan"
	"github.com/krisarmstrong/netscope/internal/wifi"
	"github.com/krisarmstrong/netscope/internal/cable"
	"github.com/krisarmstrong/netscope/internal/speedtest"
)

// Server represents the HTTP/HTTPS server.
type Server struct {
	config           *config.Config
	httpServer       *http.Server
	authManager      *auth.Manager
	wsHub            *Hub
	mux              *http.ServeMux
	netManager       *network.Manager
	linkMonitor      *network.LinkMonitor
	discoveryManager *discovery.Manager
	dnsTester        *dns.Tester
	dhcpMonitor      *dhcp.Monitor
	gatewayTester    *gateway.Tester
	vlanManager      *vlan.Manager
	wifiManager      *wifi.Manager
	cableTester      *cable.Tester
	speedtestTester  *speedtest.Tester
}

// NewServer creates a new server instance.
func NewServer(cfg *config.Config, netMgr *network.Manager) *Server {
	s := &Server{
		config:     cfg,
		mux:        http.NewServeMux(),
		netManager: netMgr,
		authManager: auth.NewManager(
			cfg.Auth.JWTSecret,
			cfg.Auth.SessionTimeout,
			cfg.Auth.DefaultUsername,
			cfg.Auth.DefaultPasswordHash,
		),
		linkMonitor:      network.NewLinkMonitor(cfg.Interface.Default),
		discoveryManager: discovery.NewManager(cfg.Interface.Default),
		dnsTester:        dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds()),
		dhcpMonitor:      dhcp.NewMonitor(cfg.Interface.Default),
		gatewayTester:    gateway.NewTester(gateway.DefaultThresholds()),
		vlanManager:      vlan.NewManager(cfg.Interface.Default),
		wifiManager:      wifi.NewManager(cfg.Interface.Default),
		cableTester:      cable.NewTester(cfg.Interface.Default),
		speedtestTester:  speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID),
	}

	// Set up link state change callback
	s.linkMonitor.OnStateChange(s.onLinkStateChange)

	s.wsHub = NewHub()
	s.setupRoutes()

	return s
}

// onLinkStateChange handles link up/down events.
func (s *Server) onLinkStateChange(event network.LinkEvent) {
	log.Printf("Link state change: %s -> %s", event.Interface, event.State)

	if event.State == network.LinkStateUp {
		// Link came up - restart discovery to catch LLDP/CDP frames
		log.Println("Link up - restarting discovery capture")
		s.discoveryManager.Stop()
		if err := s.discoveryManager.Start(); err != nil {
			log.Printf("Warning: Failed to restart discovery: %v", err)
		}

		// Notify WebSocket clients
		s.wsHub.Broadcast(Message{
			Type: "linkState",
			Payload: map[string]interface{}{
				"interface": event.Interface,
				"state":     "up",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		})
	} else if event.State == network.LinkStateDown {
		// Link went down - notify clients
		log.Println("Link down - notifying clients")
		s.wsHub.Broadcast(Message{
			Type: "linkState",
			Payload: map[string]interface{}{
				"interface": event.Interface,
				"state":     "down",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		})
	}
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/settings", s.handleSettings)
	s.mux.HandleFunc("/api/interfaces", s.handleInterfaces)
	s.mux.HandleFunc("/api/interface", s.handleInterface)
	s.mux.HandleFunc("/api/export", s.handleExport)
	s.mux.HandleFunc("/api/link", s.handleLink)
	s.mux.HandleFunc("/api/ipconfig", s.handleIPConfig)
	s.mux.HandleFunc("/api/ipconfig/settings", s.handleIPSettings)
	s.mux.HandleFunc("/api/discovery", s.handleDiscovery)
	s.mux.HandleFunc("/api/dns", s.handleDNS)
	s.mux.HandleFunc("/api/gateway", s.handleGateway)
	s.mux.HandleFunc("/api/vlan", s.handleVLAN)
	s.mux.HandleFunc("/api/wifi", s.handleWiFi)
	s.mux.HandleFunc("/api/wifi/settings", s.handleWiFiSettings)
	s.mux.HandleFunc("/api/cable", s.handleCable)
	s.mux.HandleFunc("/api/speedtest", s.handleSpeedtest)
	s.mux.HandleFunc("/api/speedtest/status", s.handleSpeedtestStatus)
	s.mux.HandleFunc("/api/tests/settings", s.handleTestsSettings)
	s.mux.HandleFunc("/api/tests/run", s.handleCustomTests)

	// WebSocket
	s.mux.HandleFunc("/ws", s.handleWebSocket)

	// Static files (frontend)
	s.mux.Handle("/", http.FileServer(http.Dir("web/dist")))
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start starts the HTTP/HTTPS server.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)

	// Apply CORS middleware then auth middleware
	handler := corsMiddleware(s.authManager.Middleware(s.mux))

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start WebSocket hub
	go s.wsHub.Run()

	// Start link state monitor
	if err := s.linkMonitor.Start(); err != nil {
		log.Printf("Warning: Link monitor failed to start: %v", err)
	} else {
		log.Printf("Link monitor started for %s (state: %s)",
			s.config.Interface.Default, s.linkMonitor.GetState())
	}

	// Start discovery capture (requires root/CAP_NET_RAW)
	if err := s.discoveryManager.Start(); err != nil {
		log.Printf("Warning: Discovery capture failed to start (may require root): %v", err)
	} else {
		log.Println("Discovery capture started")
	}

	if s.config.Server.HTTPS {
		return s.startHTTPS()
	}
	return s.startHTTP()
}

// startHTTP starts the server in HTTP mode.
func (s *Server) startHTTP() error {
	log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// startHTTPS starts the server in HTTPS mode.
func (s *Server) startHTTPS() error {
	certFile := s.config.Server.CertFile
	keyFile := s.config.Server.KeyFile

	// Generate self-signed cert if not provided
	if certFile == "" || keyFile == "" {
		var err error
		certFile, keyFile, err = s.ensureSelfSignedCert()
		if err != nil {
			return fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}
	s.httpServer.TLSConfig = tlsConfig

	log.Printf("Starting HTTPS server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

// ensureSelfSignedCert generates a self-signed certificate if needed.
func (s *Server) ensureSelfSignedCert() (certFile, keyFile string, err error) {
	certsDir := "certs"
	certFile = filepath.Join(certsDir, "server.crt")
	keyFile = filepath.Join(certsDir, "server.key")

	// Check if certs already exist
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return certFile, keyFile, nil
		}
	}

	// Ensure certs directory exists
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		return "", "", err
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"NetScope"},
			CommonName:   "NetScope Self-Signed",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "netscope.local"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	// Write certificate
	certOut, err := os.Create(certFile)
	if err != nil {
		return "", "", err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return "", "", err
	}

	// Write private key
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", err
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		return "", "", err
	}

	log.Printf("Generated self-signed certificate: %s", certFile)
	return certFile, keyFile, nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.wsHub.Shutdown()
	s.linkMonitor.Stop()
	s.discoveryManager.Stop()
	return s.httpServer.Shutdown(ctx)
}

// Hub returns the WebSocket hub.
func (s *Server) Hub() *Hub {
	return s.wsHub
}
