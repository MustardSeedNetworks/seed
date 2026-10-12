// Command fakeiperf3 stands in for the iperf3 binary in Manager tests.
//
// It is a real, separate process that really listens on the port it is given
// and then prints iperf3's listening banner, so StartServer's process spawn,
// readiness wait, PID capture, Wait monitoring and Kill on stop are all
// exercised for real. Like iperf3 writing into a pipe, it prints the banner
// only when given --forceflush. What it does not do
// is speak the iperf3 protocol — a genuine transfer belongs in the
// iperf3-required CI job, not in a test every developer runs.
//
// Behaviour is driven by argv and by FAKE_IPERF3_MODE so a single binary can
// play the failure cases too:
//
//	(unset) or "listen" — bind 127.0.0.1:<port> and block until killed
//	"exit-nonzero"      — exit 1 immediately, as a missing-permission iperf3 would
//	"never-listen"      — stay alive but never bind, so the readiness wait times out
//
// FAKE_IPERF3_PIDFILE, when set, receives this process's pid before it does
// anything else, so a test can assert the parent really killed it.
// FAKE_IPERF3_CONNFILE, when set, gets the remote address of every connection
// accepted, one per line, so a test can see who dialed the server.
package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

func main() {
	port := ""
	forceFlush := false
	for i, arg := range os.Args {
		switch {
		case arg == "-p" && i+1 < len(os.Args):
			port = os.Args[i+1]
		case arg == "--forceflush":
			forceFlush = true
		}
	}

	if pidFile := os.Getenv("FAKE_IPERF3_PIDFILE"); pidFile != "" {
		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "fakeiperf3: write pidfile: %v\n", err)
			os.Exit(3)
		}
	}

	switch os.Getenv("FAKE_IPERF3_MODE") {
	case "exit-nonzero":
		fmt.Fprintln(os.Stderr, "fakeiperf3: refusing to start")
		os.Exit(1)
	case "never-listen":
		// Alive, but nothing is ever bound: the readiness wait must give up
		// and StartServer must kill this process rather than leave it running.
		time.Sleep(10 * time.Minute)

		return
	}

	if port == "" {
		fmt.Fprintln(os.Stderr, "fakeiperf3: no -p <port> given")
		os.Exit(2)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fakeiperf3: listen: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = listener.Close() }()

	if forceFlush {
		fmt.Printf("-----------------------------------------------------------\n"+
			"Server listening on %s (test #1)\n", port)
	}

	// Accept and immediately close. Blocks until the parent kills the process.
	connFile := os.Getenv("FAKE_IPERF3_CONNFILE")
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		if connFile != "" {
			recordConn(connFile, conn.RemoteAddr().String())
		}
		_ = conn.Close()
	}
}

func recordConn(path, remote string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fakeiperf3: open connfile: %v\n", err)
		os.Exit(3)
	}
	defer func() { _ = f.Close() }()
	if _, err := fmt.Fprintln(f, remote); err != nil {
		fmt.Fprintf(os.Stderr, "fakeiperf3: write connfile: %v\n", err)
		os.Exit(3)
	}
}
