// Command fakeiperf3 stands in for the iperf3 binary in Manager tests.
//
// It is a real, separate process that really listens on the port it is given,
// so StartServer's process spawn, port-readiness wait, PID capture, Wait
// monitoring and Kill on stop are all exercised for real. What it does not do
// is speak the iperf3 protocol — a genuine transfer belongs in the
// iperf3-required CI job, not in a test every developer runs.
//
// Behaviour is driven by argv and by FAKE_IPERF3_MODE so a single binary can
// play the failure cases too:
//
//	(unset) or "listen" — bind 127.0.0.1:<port> and block until killed
//	"exit-nonzero"      — exit 1 immediately, as a missing-permission iperf3 would
//	"never-listen"      — stay alive but never bind, so the port wait times out
//
// FAKE_IPERF3_PIDFILE, when set, receives this process's pid before it does
// anything else, so a test can assert the parent really killed it.
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
	for i, arg := range os.Args {
		if arg == "-p" && i+1 < len(os.Args) {
			port = os.Args[i+1]
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
		// Alive, but nothing is ever bound: the port-readiness wait must give up
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

	// Accept and immediately close, so a readiness probe that dials sees a live
	// listener. Blocks until the parent kills the process.
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		_ = conn.Close()
	}
}
