package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

func main() {
	inHostPort := flag.String("in_hostport", ":8888", "host:port to serve incoming HTTP requests")
	logfile := flag.String("logfile", "", "File written with message logs")
	flag.Usage = func() {
		const helpText = "Web service that decodes client email opens and clicks\n" +
			"Runs in plain mode, it should proxied (e.g. by nginx) to provide https and protection.\n" +
			"Usage of %s:\n"
		fmt.Fprintf(flag.CommandLine.Output(), helpText, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	spmta.MyLogger(*logfile)
	fmt.Printf("Starting http server on %s, logging to %s\n", *inHostPort, *logfile)
	log.Printf("Starting http server on %s\n", *inHostPort)
	// http server
	http.HandleFunc("/", spmta.TrackingServer) // Accept subtree matches
	server := &http.Server{
		Addr: *inHostPort,
	}
	err := server.ListenAndServe()
	if err != nil {
		spmta.ConsoleAndLogFatal(err)
	}
}
