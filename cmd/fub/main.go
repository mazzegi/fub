package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mazzegi/fub"
)

func resolveTarget(target string) (string, error) {
	var host string
	ix := strings.Index(target, ":")
	if ix < 0 {
		host = target
		target += ":80"
	} else {
		host = target[:ix]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		as, err := net.LookupHost(host)
		if err != nil {
			return "", err
		}
		return as[0] + ":80", nil
	} else {
		return target, nil
	}
}

func main() {
	host := flag.String("host", "127.0.0.1:9201", "fubd-host")
	expose := flag.Int("expose", 4711, "port to expose by fub")
	target := flag.String("target", "google.de", "target to forward connections to")
	flag.Parse()

	addr, err := resolveTarget(*target)
	if err != nil {
		fub.Errorf("failed to resolve target %q: %v", *target, err)
		os.Exit(1)
	}
	fub.Infof("resolved %q -> %q", *target, addr)

	fub.Infof("run fub client to (%s) ...", *host)
	c := fub.NewClient(*host, *expose, addr)
	go c.Run()

	sigC := make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	<-sigC
	c.Close()
	fub.Infof("run fub server ... done")
}
