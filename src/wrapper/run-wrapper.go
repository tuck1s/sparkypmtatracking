package wrapper

import (
	"os"

	c "github.com/tuck1s/sparkyPmtaTracking/src/common"
)

func main() {
	// Use logging, as this program will be executed without an attached console
	c.MyLogger("wrapper.log")

	htmlFile, err := os.Open("big.html")
	if err != nil {
		os.Exit(1)
	}
	// Only need to set these parts up once
	myTracker := NewTracker("http://pmta.signalsdemo.trymsys.net/")
	myTracker.MessageInfo(uniqMessageID(), "bob.lumreeker@gmail.com")
	err = myTracker.TrackHTML(htmlFile, os.Stdout)
	c.Check(err)
}
