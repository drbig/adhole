// +build windows

package main

import (
	"log"
	"os"
	"os/signal"
)

func sigloop() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	<-sig
	log.Println("Signal received, stopping")

	return
}
