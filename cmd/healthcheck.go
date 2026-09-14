package cmd

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/service"
)

// healthCheckTimeout is per dial. A container health check runs on a schedule
// and must not hang until the orchestrator's own timeout.
const healthCheckTimeout = 3 * time.Second

// healthCheck reports whether the panel is accepting connections on the port it
// is actually configured with.
//
// It reads the port from the database rather than taking one on the command
// line, because the operator can change it from the panel at any time. A health
// check with the port baked into the Dockerfile goes red the moment they do,
// and an orchestrator then kills a container that was working perfectly.
//
// It dials rather than speaking HTTP: the panel serves either HTTP or HTTPS
// depending on whether a certificate is configured, and a check that assumed
// one of them would fail on the other.
func healthCheck() {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("healthcheck: unable to open the database:", err)
		os.Exit(1)
	}

	settingService := service.SettingService{}
	port, err := settingService.GetPort()
	if err != nil {
		fmt.Println("healthcheck: unable to read the panel port:", err)
		os.Exit(1)
	}

	// The listen address may be a specific interface, but the check runs
	// inside the container, so loopback is what it can reach.
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, healthCheckTimeout)
	if err != nil {
		// IPv6-only loopback is possible when the panel binds "::".
		addr6 := net.JoinHostPort("::1", strconv.Itoa(port))
		conn, err = net.DialTimeout("tcp", addr6, healthCheckTimeout)
		if err != nil {
			fmt.Printf("healthcheck: nothing listening on port %d: %v\n", port, err)
			os.Exit(1)
		}
	}
	conn.Close()
	fmt.Printf("healthcheck: panel is listening on port %d\n", port)
}
