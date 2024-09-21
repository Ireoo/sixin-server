package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ireoo/sixin-server/base"
)

func StartServer(server *http.Server, baseInstance *base.Base) {
	log.Printf("Server running on %s\n", server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	exit := make(chan struct{})
	SignalC := make(chan os.Signal, 1)

	signal.Notify(SignalC, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				close(exit)
				return
			}
		}
	}()

	<-exit
	baseInstance.IO.Close(nil)
	server.Shutdown(context.Background())
	os.Exit(0)
}
