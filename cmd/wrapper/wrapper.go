package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"os"
	"time"

	"github.com/tuck1s/go-smtpproxy"
	spmta "github.com/tuck1s/sparkyPMTATracking"
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
	wrapURL := flag.String("engagement_url", "", "Engagement tracking URL used in html email body for opens and clicks")
	insecureSkipVerify := flag.Bool("insecure_skip_verify", false, "Skip check of peer cert on upstream side")
	flag.Parse()
	spmta.MyLogger(*logfile)

	log.Println("Incoming host:port set to", *inHostPort)
	log.Println("Outgoing host:port set to", *outHostPort)

	// Logging of proxy to upstream server DATA (in RFC822 .eml format)
	var upstreamDebugFile *os.File
	var err error
	if *upstreamDataDebug != "" {
		upstreamDebugFile, err = os.OpenFile(*upstreamDataDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Proxy writing upstream DATA to", upstreamDebugFile.Name())
		defer upstreamDebugFile.Close()
	}

	var myWrapper *spmta.Wrapper // Will be nil if not using engagement tracking

	log.Println("Engagement tracking URL:", *wrapURL)
	myWrapper, err = spmta.NewWrapper(*wrapURL)
	if err != nil && err.Error() != "parse : empty url" {
		log.Fatal(err)
	}

	log.Println("insecure_skip_verify (Skip check of peer cert on upstream side):", *insecureSkipVerify)

	// Set up parameters that the backend will use
	be := spmta.NewBackend(*outHostPort, *verboseOpt, upstreamDebugFile, myWrapper, *insecureSkipVerify)
	s := smtpproxy.NewServer(be)
	s.Addr = *inHostPort
	s.ReadTimeout = 60 * time.Second
	s.WriteTimeout = 60 * time.Second

	subject, err := os.Hostname() // This is the fallback in case we have no cert / privkey to give us a Subject
	if err != nil {
		log.Fatal("Can't read hostname")
	}

	// Gather TLS credentials from filesystem. Use these with the server and also set the EHLO server name
	if *certfile == "" || *privkeyfile == "" {
		log.Println("Warning: certfile or privkeyfile not specified - proxy will NOT offer STARTTLS to clients")
	} else {
		cer, err := tls.LoadX509KeyPair(*certfile, *privkeyfile)
		if err != nil {
			log.Fatal(err)
		}
		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		s.TLSConfig = config

		leafCert, err := x509.ParseCertificate(cer.Certificate[0])
		if err != nil {
			log.Fatal(err)
		}
		subject = leafCert.Subject.CommonName
		log.Println("Gathered certificate", *certfile, "and key", *privkeyfile)
	}
	s.Domain = subject
	log.Println("Proxy will advertise itself as", s.Domain)
	log.Println("Verbose SMTP conversation logging:", *verboseOpt)

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
