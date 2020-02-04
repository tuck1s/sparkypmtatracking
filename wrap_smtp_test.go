package sparkypmtatracking_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	smtpproxy "github.com/tuck1s/go-smtpproxy"
	spmta "github.com/tuck1s/sparkyPMTATracking"
)

const (
	Init = iota
	Greeted
	AskedUsername
	AskedPassword
	GotPassword
)

// Test design is to make a "sandwich" with wrapper in the middle.
//      test client <--> wrapper <--> mock SMTP server (Backend, Session)
// The mock SMTP server returns realistic looking response codes etc
type Backend struct {
}

// A Session is returned after successful login. Here hold information that needs to persist across message phases.
type Session struct {
	MockState int
}

// mockSMTPServer should be invoked as a goroutine to allow tests to continue
func mockSMTPServer(addr string) {
	mockbe := Backend{}
	s := smtpproxy.NewServer(&mockbe)
	s.Addr = addr
	s.ReadTimeout = 60 * time.Second // changeme?
	s.WriteTimeout = 60 * time.Second

	// Begin serving requests
	fmt.Println("Upstream mock SMTP server listening on", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// Init the backend. This does not need to do much.
func (bkd *Backend) Init() (smtpproxy.Session, error) {
	var s Session
	s.MockState = Init
	return &s, nil
}

// Greet the upstream host and report capabilities back.
func (s *Session) Greet(helotype string) ([]string, int, string, error) {
	s.MockState = Greeted
	caps := []string{"8BITMIME", "STARTTLS", "ENHANCEDSTATUSCODES", "AUTH LOGIN PLAIN", "SMTPUTF8"}
	return caps, 220, "", nil
}

// StartTLS command
func (s *Session) StartTLS() (int, string, error) {
	return 220, "", nil
}

const mockMsg = "2.0.0 mock server accepts all"

//Auth command mock backend handler - naive, handles only AUTH LOGIN PLAIN
func (s *Session) Auth(expectcode int, cmd, arg string) (int, string, error) {
	var code int
	var msg string
	switch s.MockState {
	case Init:
	case Greeted:
		if arg == "LOGIN" {
			code = 334
			msg = base64.StdEncoding.EncodeToString([]byte("Username:"))
			s.MockState = AskedUsername
		} else if strings.HasPrefix(arg, "PLAIN") {
			code = 235
			msg = mockMsg
			s.MockState = GotPassword
		}
	case AskedUsername:
		code = 334
		msg = base64.StdEncoding.EncodeToString([]byte("Password:"))
		s.MockState = AskedPassword
	case AskedPassword:
		code = 235
		msg = mockMsg
		s.MockState = GotPassword
	}
	return code, msg, nil
}

//Mail command mock backend handler
func (s *Session) Mail(expectcode int, cmd, arg string) (int, string, error) {
	return 250, mockMsg, nil
}

//Rcpt command mock backend handler
func (s *Session) Rcpt(expectcode int, cmd, arg string) (int, string, error) {
	return 250, mockMsg, nil
}

//Reset command mock backend handler
func (s *Session) Reset(expectcode int, cmd, arg string) (int, string, error) {
	s.MockState = Init
	return 250, "2.0.0 mock reset", nil
}

//Quit command mock backend handler
func (s *Session) Quit(expectcode int, cmd, arg string) (int, string, error) {
	s.MockState = Init
	return 221, "2.3.0 mock says bye", nil
}

//Unknown command mock backend handler
func (s *Session) Unknown(expectcode int, cmd, arg string) (int, string, error) {
	return 500, "mock does not recognize this command", nil
}

type discardCloser struct {
	io.Writer
}

func (discardCloser) Close() error {
	return nil
}

// DataCommand pass upstream, returning a place to write the data AND the usual responses
// If you want to see the mail contents, replace Discard with os.Stdout
func (s *Session) DataCommand() (io.WriteCloser, int, string, error) {
	return discardCloser{Writer: ioutil.Discard}, 354, `3.0.0 mock says continue.  finished with "\r\n.\r\n"`, nil
}

// Data body (dot delimited) pass upstream, returning the usual responses
func (s *Session) Data(r io.Reader, w io.WriteCloser) (int, string, error) {
	io.Copy(w, r)
	return 250, "2.0.0 OK mock got your dot", nil
}

//-----------------------------------------------------------------------------
// Start proxy server

func startProxy(s *smtpproxy.Server) {
	fmt.Println("Proxy (unit under test) listening on", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// wrap_smtp tests
func TestWrapSMTP(t *testing.T) {
	inHostPort := ":5587"
	rand.Seed(time.Now().UTC().UnixNano())
	p := 6317 // rand.Intn(1000) + 6000 // use a random port unmber as workaround for resource hogging during debug
	outHostPort := ":" + strconv.Itoa(p)
	certfile := "fullchain.pem"  // make this load from an embedded string
	privkeyfile := "privkey.pem" //make this load from a string
	verboseOpt := true
	downstreamDebug := "debug_wrap_smtp_test.log"
	upstreamDataDebug := "debug_wrap_smtp_test_upstream.eml"
	wrapURL := "https://track.example.com"
	insecureSkipVerify := true
	// Logging of proxy to upstream server DATA (in RFC822 .eml format)
	var upstreamDebugFile *os.File
	if upstreamDataDebug != "" {
		// Logging of proxy to upstream server DATA (in RFC822 .eml format)
		upstreamDebugFile, err := os.OpenFile(upstreamDataDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			t.Error(err)
		}
		defer upstreamDebugFile.Close()
	}
	myWrapper, err := spmta.NewWrapper(wrapURL)
	if err != nil {
		log.Fatal(err)
	}

	// Set up parameters that the backend will use
	be := spmta.NewBackend(outHostPort, verboseOpt, upstreamDebugFile, myWrapper, insecureSkipVerify)
	s := smtpproxy.NewServer(be)
	s.Addr = inHostPort
	s.ReadTimeout = 60 * time.Second
	s.WriteTimeout = 60 * time.Second

	subject, err := os.Hostname() // This is the fallback in case we have no cert / privkey to give us a Subject
	if err != nil {
		t.Errorf("Can't read hostname")
	}

	// Gather TLS credentials from filesystem. Use these with the server and also set the EHLO server name
	if certfile == "" || privkeyfile == "" {
		log.Println("Warning: certfile or privkeyfile not specified - proxy will NOT offer STARTTLS to clients")
	} else {
		cer, err := tls.LoadX509KeyPair(certfile, privkeyfile)
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
		log.Println("Gathered certificate", certfile, "and key", privkeyfile)
	}
	s.Domain = subject

	// Logging of downstream (client to proxy server) commands and responses
	if downstreamDebug != "" {
		dbgFile, err := os.OpenFile(downstreamDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			t.Error(err)
		}
		defer dbgFile.Close()
		s.Debug = dbgFile
	}

	// start the upstream mock SMTP server
	go mockSMTPServer(outHostPort)

	// start the proxy
	go startProxy(s)

	// open a test client TODO - for now just give time for the goroutines to start
	for i := 0; i < 60; i++ {
		time.Sleep(time.Second)
		fmt.Println(".")
	}
}
