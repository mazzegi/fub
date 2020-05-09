package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/mazzegi/fub"
)

func main() {
	host := flag.String("host", "127.0.0.1:9201", "fubd-host")
	flag.Parse()

	fub.Infof("run fub client to (%s) ...", *host)
	c := fub.NewClient(*host)
	go c.Run()

	sigC := make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	<-sigC
	c.Close()
	fub.Infof("run fub server ... done")
}
