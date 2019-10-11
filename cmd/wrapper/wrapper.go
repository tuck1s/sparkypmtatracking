package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"os"
	"time"

	"github.com/tuck1s/go-smtpproxy"
	c "github.com/tuck1s/sparkyPMTATracking"
)

func main() {
	//c.MyLogger("wrapper.log")

	inHostPort := flag.String("in_hostport", "localhost:587", "Port number to serve incoming SMTP requests")
	outHostPort := flag.String("out_hostport", "smtp.sparkpostmail.com:587", "host:port for onward routing of SMTP requests")
	verboseOpt := flag.Bool("verbose", false, "print out lots of messages")
	certfile := flag.String("certfile", "", "Certificate file for this server")
	privkeyfile := flag.String("privkeyfile", "", "Private key file for this server")
	serverDebug := flag.String("server_debug", "", "File to write downstream server SMTP conversation for debugging")
	upstreamDebug := flag.String("upstream_debug", "", "File to write upstream proxy SMTP conversation for debugging")
	flag.Parse()

	log.Println("Incoming host:port set to", *inHostPort)
	log.Println("Outgoing host:port set to", *outHostPort)

	var upstreamDebugFile os.File
	if *upstreamDebug != "" {
		// Overwrite each time
		upstreamDbgFile, err := os.OpenFile(*upstreamDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer upstreamDbgFile.Close()
	}

	// Set up parameters that the backend will use
	be := c.NewBackend(*outHostPort, *verboseOpt, &upstreamDebugFile)

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
	log.Println("Backend logging:", *verboseOpt)

	if *serverDebug != "" {
		// Need local ref to the file, to allow Close() and Name() methods which io.Writer doesn't have
		// Overwrite each time
		dbgFile, err := os.OpenFile(*serverDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer dbgFile.Close()
		s.Debug = dbgFile
		log.Println("Proxy logging SMTP commands, responses and downstream DATA to", dbgFile.Name())
	}

	if true { // TODO
		log.Println("Proxy writing upstream DATA to", upstreamDebugFile.Name())
	}

	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
