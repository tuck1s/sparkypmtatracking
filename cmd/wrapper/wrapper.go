package main

import (
	"flag"

	c "github.com/tuck1s/sparkyPMTATracking"
)

func main() {
	c.MyLogger("wrapper.log")
	flag.Parse()
}
