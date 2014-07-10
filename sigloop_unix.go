// +build !windows

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func sigloop() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

forever:
	for {
		s := <-sig
		switch s {
		case syscall.SIGUSR1:
			log.Println("SIGUSR1 received, reloading rules")
			parseList(flag.Arg(2))
		default:
			log.Println("Signal received, stopping")
			break forever
		}
	}

	return
}
