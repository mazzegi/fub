package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/mazzegi/fub"
)

func main() {
	bind := flag.String("bind", "127.0.0.1:9201", "bind address")
	flag.Parse()

	fub.Infof("run fub server ...")
	s, err := fub.NewServer(fub.Bind(*bind))
	if err != nil {
		fub.Errorf("new-server: %v", err)
		os.Exit(1)
	}
	go s.Run()

	sigC := make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	<-sigC
	s.Close()
	fub.Infof("run fub server ... done")
}
