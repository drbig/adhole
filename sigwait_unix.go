// See LICENSE.txt for licensing information.
// +build !windows

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// sigwait processes signals such as a CTRL-C hit.
func sigwait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	<-sig
	log.Println("Signal received, stopping")

	return
}
