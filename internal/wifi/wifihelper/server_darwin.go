//go:build darwin

package wifihelper

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// requestTimeout bounds a single exchange. A scan takes a few seconds; well
// beyond that the helper is wedged and the daemon should say so rather than
// block a request handler.
const requestTimeout = 20 * time.Second

var (
	// ErrNoHelper means no verified helper is currently connected, so macOS
	// cannot be scanned. It is distinct from a scan that found nothing.
	ErrNoHelper = errors.New("wifihelper: no helper connected")

	// ErrNotAssociated means the Wi-Fi interface has not joined a network.
	ErrNotAssociated = errors.New("wifihelper: not associated")
)

// Server is the daemon side. It listens on a unix socket, admits one helper at
// a time after verifying its code signature, and issues requests over that
// connection.
type Server struct {
	requirement string
	log         *slog.Logger

	listener *net.UnixListener

	mu   sync.Mutex // serializes exchanges; the protocol is one reply per request
	conn *net.UnixConn
	rd   *bufio.Reader
}

// NewServer starts listening at path. Only a peer satisfying requirement — a
// code-signing requirement string — is admitted.
func NewServer(path, requirement string, log *slog.Logger) (*Server, error) {
	l, err := listenExclusive(path)
	if err != nil {
		return nil, err
	}

	s := &Server{requirement: requirement, log: log, listener: l}
	go s.accept()
	return s, nil
}

// accept admits helpers one at a time, replacing any previous connection: the
// helper restarts whenever the user logs out and back in.
func (s *Server) accept() {
	for {
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			return // listener closed
		}

		if verifyErr := VerifyPeer(conn, s.requirement); verifyErr != nil {
			s.log.WarnContext(context.Background(), "rejected wifi helper connection",
				slog.String("event", "wifi.helper.rejected"),
				slog.String("error", verifyErr.Error()))
			_ = conn.Close()
			continue
		}

		s.mu.Lock()
		if s.conn != nil {
			_ = s.conn.Close()
		}
		s.conn = conn
		s.rd = bufio.NewReader(conn)
		s.mu.Unlock()

		s.log.InfoContext(context.Background(), "wifi helper connected",
			slog.String("event", "wifi.helper.connected"))
	}
}

// Close stops listening and drops any connected helper.
func (s *Server) Close() error {
	err := s.listener.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	return err
}

// exchange sends one request and reads its reply.
func (s *Server) exchange(op Op) (Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return Response{}, ErrNoHelper
	}

	if err := s.conn.SetDeadline(time.Now().Add(requestTimeout)); err != nil {
		return Response{}, fmt.Errorf("wifihelper: set deadline: %w", err)
	}

	payload, err := json.Marshal(Request{Op: op})
	if err != nil {
		return Response{}, fmt.Errorf("wifihelper: encode request: %w", err)
	}
	if _, err = s.conn.Write(append(payload, '\n')); err != nil {
		s.dropLocked()
		return Response{}, fmt.Errorf("wifihelper: send request: %w", err)
	}

	line, err := s.rd.ReadBytes('\n')
	if err != nil {
		s.dropLocked()
		return Response{}, fmt.Errorf("wifihelper: read reply: %w", err)
	}

	return DecodeResponse(line)
}

// dropLocked discards a connection that failed mid-exchange so the next request
// reports ErrNoHelper instead of reusing a broken stream. Caller holds s.mu.
func (s *Server) dropLocked() {
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
		s.rd = nil
	}
}

// Scan asks the helper for a Wi-Fi scan.
func (s *Server) Scan() ([]Network, error) {
	resp, err := s.exchange(OpScan)
	if err != nil {
		return nil, err
	}
	if err = resp.Err(); err != nil {
		return nil, err
	}
	return resp.Networks, nil
}

// Current asks the helper for the current association, returning
// [ErrNotAssociated] when the interface has not joined a network.
func (s *Server) Current() (Network, error) {
	resp, err := s.exchange(OpCurrent)
	if err != nil {
		return Network{}, err
	}
	if err = resp.Err(); err != nil {
		return Network{}, err
	}
	if len(resp.Networks) == 0 {
		return Network{}, ErrNotAssociated
	}
	return resp.Networks[0], nil
}

// Saved asks the helper for remembered network names.
func (s *Server) Saved() ([]string, error) {
	resp, err := s.exchange(OpSaved)
	if err != nil {
		return nil, err
	}
	if err = resp.Err(); err != nil {
		return nil, err
	}
	return resp.Names, nil
}
