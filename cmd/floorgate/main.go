// Package main implements floorgate, the stable authenticated front door for
// theboringfloor offices whose per-project control ports and tokens are
// intentionally ephemeral.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/version"
)

func main() {
	bind := flag.String("bind", "", "HTTP listen address")
	enableExec := flag.Bool("exec", false, "enable remote shell command execution")
	printToken := flag.Bool("print-token", false, "print the gateway bearer token and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return
	}

	token, err := loadGatewayToken(gatewayHome())
	if err != nil {
		fmt.Fprintln(os.Stderr, "floorgate:", err)
		os.Exit(1)
	}
	if *printToken {
		fmt.Println(token)
		return
	}

	listener, err := net.Listen("tcp", resolvedBind(*bind, config.Env("GATE_BIND")))
	if err != nil {
		fmt.Fprintln(os.Stderr, "floorgate:", err)
		os.Exit(1)
	}
	defer listener.Close()
	if warning := bindWarning(listener.Addr()); warning != "" {
		fmt.Fprint(os.Stderr, warning)
	}

	gateway := newGateway(token)
	gateway.exec = *enableExec || config.Env("FLOORGATE_EXEC") == "1"
	server := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           gateway,
		ReadHeaderTimeout: 5 * time.Second,
		// Image messages can contain up to 16 MiB of decoded attachments. Allow
		// slow mobile uploads enough time to reach the route's 32 MiB body cap.
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   125 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "floorgate:", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	signal.Stop(signals)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "floorgate:", err)
	}
}
