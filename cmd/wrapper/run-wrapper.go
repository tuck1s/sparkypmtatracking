package main

import (
	"os"

	c "github.com/tuck1s/sparkyPMTATracking"
)

func main() {
	// Use logging, as this program will be executed without an attached console
	c.MyLogger("wrapper.log")

	htmlFile, err := os.Open("big.html")
	if err != nil {
		os.Exit(1)
	}
	// Only need to set these parts up once
	myTracker := c.NewTracker("http://pmta.signalsdemo.trymsys.net/")
	myTracker.MessageInfo(c.UniqMessageID(), "bob.lumreeker@gmail.com")
	err = myTracker.TrackHTML(htmlFile, os.Stdout)
	c.Check(err)
}
