package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tuck1s/go-smtpproxy"
	spmta "github.com/tuck1s/sparkypmtatracking"
)

func main() {
	inHostPort := flag.String("in_hostport", "localhost:587", "Port number to serve incoming SMTP requests")
	outHostPort := flag.String("out_hostport", "smtp.sparkpostmail.com:587", "host:port for onward routing of SMTP requests")
	certfile := flag.String("certfile", "", "Certificate file for this server")
	privkeyfile := flag.String("privkeyfile", "", "Private key file for this server")
	logfile := flag.String("logfile", "", "File written with message logs (also to stdout)")
	verboseOpt := flag.Bool("verbose", false, "print out lots of messages")
	downstreamDebug := flag.String("downstream_debug", "", "File to write downstream server SMTP conversation for debugging")
	upstreamDataDebug := flag.String("upstream_data_debug", "", "File to write upstream DATA for debugging")
	trackingURL := flag.String("tracking_url", "http://localhost:8888", "URL of your tracking service endpoint")
	trackOpen := flag.Bool("track_open", true, "Insert an open tracking pixel at bottom of HTML mail")
	trackInitialOpen := flag.Bool("track_initial_open", false, "Insert an initial_open tracking pixel at top of HTML mail")
	trackLink := flag.Bool("track_click", false, "Wrap links in HTML mail, to track clicks")
	insecureSkipVerify := flag.Bool("insecure_skip_verify", false, "Skip check of peer cert on upstream side")
	flag.Usage = func() {
		const helpText = "SMTP proxy that accepts incoming messages from your downstream client, applies engagement-tracking\n" +
			"(wrapping links and adding open tracking pixels) and relays on to an upstream server.\n" +
			"Usage of %s:\n"
		fmt.Fprintf(flag.CommandLine.Output(), helpText, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	spmta.MyLogger(*logfile)
	fmt.Println("Starting smtp proxy service on port", *inHostPort, ", logging to", *logfile)
	log.Println("Starting smtp proxy service on port", *inHostPort)
	log.Println("Outgoing host:port set to", *outHostPort)
	log.Printf("Engagement tracking URL: %s, track_open %v, track_initial_open %v, track_click %v\n", *trackingURL, *trackOpen, *trackInitialOpen, *trackLink)
	myWrapper, err := spmta.NewWrapper(*trackingURL, *trackOpen, *trackInitialOpen, *trackLink)
	if err != nil && !strings.Contains(err.Error(), "empty url") {
		log.Fatal(err)
	}

	// Logging of upstream server DATA (in RFC822 .eml format) for debugging
	var upstreamDebugFile *os.File // need this not in inner scope
	if *upstreamDataDebug != "" {
		upstreamDebugFile, err = os.OpenFile(*upstreamDataDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Proxy writing upstream DATA to", upstreamDebugFile.Name())
		defer upstreamDebugFile.Close()
	}

	// Set up parameters that the backend will use
	be := spmta.NewBackend(*outHostPort, *verboseOpt, upstreamDebugFile, myWrapper, *insecureSkipVerify)
	s := smtpproxy.NewServer(be)
	s.Addr = *inHostPort
	s.ReadTimeout = 60 * time.Second
	s.WriteTimeout = 60 * time.Second

	// Gather TLS credentials for the proxy server
	if *certfile != "" && *privkeyfile != "" {
		cert, err := ioutil.ReadFile(*certfile)
		if err != nil {
			log.Fatal(err)
		}
		privkey, err := ioutil.ReadFile(*privkeyfile)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Gathered certificate", *certfile, "and key", *privkeyfile)
		err = s.ServeTLS(cert, privkey)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("certfile or privkeyfile not specified - proxy will NOT offer STARTTLS to clients")
		s.Domain, err = os.Hostname() // This is the fallback in case we have no cert / privkey to give us a Subject
		if err != nil {
			log.Fatal("Can't read hostname")
		}
	}

	log.Println("Proxy will advertise itself as", s.Domain)
	log.Println("Verbose SMTP conversation logging:", *verboseOpt)
	log.Println("insecure_skip_verify (Skip check of peer cert on upstream side):", *insecureSkipVerify)

	// Logging of downstream (client to proxy server) commands and responses
	if *downstreamDebug != "" {
		dbgFile, err := os.OpenFile(*downstreamDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer dbgFile.Close()
		s.Debug = dbgFile
		log.Println("Proxy logging SMTP commands, responses and downstream DATA to", dbgFile.Name())
	}

	// Begin serving requests
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
