// See LICENSE.txt for licensing information.
// +build windows

package main

import (
	"log"
	"os"
	"os/signal"
)

// sigwait processes signals such as a CTRL-C hit.
func sigwait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	<-sig
	log.Println("Signal received, stopping")

	return
}
