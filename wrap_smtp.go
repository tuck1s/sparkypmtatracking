package sparkypmtatracking

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"

	"github.com/tuck1s/go-smtpproxy"
)

//-----------------------------------------------------------------------------
// Backend handlers

// The Backend implements SMTP server methods.
type Backend struct {
	outHostPort        string
	verbose            bool
	upstreamDataDebug  *os.File
	wrapper            *Wrapper
	insecureSkipVerify bool
}

// NewBackend init function
func NewBackend(outHostPort string, verbose bool, upstreamDataDebug *os.File, wrapper *Wrapper, insecureSkipVerify bool) *Backend {
	b := Backend{
		outHostPort:        outHostPort,
		verbose:            verbose,
		upstreamDataDebug:  upstreamDataDebug,
		wrapper:            wrapper,
		insecureSkipVerify: insecureSkipVerify,
	}
	return &b
}

func (bkd *Backend) logger(args ...interface{}) {
	if bkd.verbose {
		log.Println(args...)
	}
}

func (bkd *Backend) loggerAlways(args ...interface{}) {
	log.Println(args...)
}

// Init the backend. Here we establish the upstream connection
func (bkd *Backend) Init() (smtpproxy.Session, error) {
	var s Session
	bkd.logger("---Connecting upstream")
	c, err := smtpproxy.Dial(bkd.outHostPort)
	s.bkd = bkd    // just for logging
	s.upstream = c // keep record of the upstream Client connection
	if err != nil {
		bkd.loggerAlways(respTwiddle(&s), "Connection error", bkd.outHostPort, err.Error())
	}
	bkd.logger(respTwiddle(&s), "Connection success", bkd.outHostPort)
	return &s, nil
}

//-----------------------------------------------------------------------------
// Session handlers

// A Session is returned after successful login. Here hold information that needs to persist across message phases.
type Session struct {
	bkd      *Backend          // The backend that created this session. Allows session methods to e.g. log
	upstream *smtpproxy.Client // the upstream client this backend is driving
}

// cmdTwiddle returns different flow markers depending on whether connection is secure (like Swaks does)
func cmdTwiddle(s *Session) string {
	if _, isTLS := s.upstream.TLSConnectionState(); isTLS {
		return "~>"
	}
	return "->"
}

// respTwiddle returns different flow markers depending on whether connection is secure (like Swaks does)
func respTwiddle(s *Session) string {
	if _, isTLS := s.upstream.TLSConnectionState(); isTLS {
		return "\t<~"
	}
	return "\t<-"
}

// Greet the upstream host and report capabilities back.
func (s *Session) Greet(helotype string) ([]string, int, string, error) {
	var (
		err  error
		code int
		msg  string
	)
	s.bkd.logger(cmdTwiddle(s), helotype)
	host, _, _ := net.SplitHostPort(s.bkd.outHostPort)
	if host == "" {
		host = "smtpproxy.localhost" // add dummy value in
	}
	code, msg, err = s.upstream.Hello(host)
	if err != nil {
		s.bkd.loggerAlways(respTwiddle(s), helotype, "error", err.Error())
		if code == 0 {
			// some errors don't show up in (code,msg) e.g. TLS cert errors, so map as a specific SMTP code/msg response
			code = 599
			msg = err.Error()
		}
		return nil, code, msg, err
	}
	s.bkd.logger(respTwiddle(s), helotype, "success")
	caps := s.upstream.Capabilities()
	s.bkd.logger("\tUpstream capabilities:", caps)
	return caps, code, msg, err
}

// StartTLS command
func (s *Session) StartTLS() (int, string, error) {
	host, _, _ := net.SplitHostPort(s.bkd.outHostPort)
	// Try the upstream server, it will report error if unsupported
	tlsconfig := &tls.Config{
		InsecureSkipVerify: s.bkd.insecureSkipVerify,
		ServerName:         host,
	}
	s.bkd.logger(cmdTwiddle(s), "STARTTLS")
	code, msg, err := s.upstream.StartTLS(tlsconfig)
	if err != nil {
		s.bkd.loggerAlways(respTwiddle(s), code, msg)
	} else {
		s.bkd.logger(respTwiddle(s), code, msg)
	}
	return code, msg, err
}

//Auth command backend handler
func (s *Session) Auth(expectcode int, cmd, arg string) (int, string, error) {
	return s.Passthru(expectcode, cmd, arg)
}

//Mail command backend handler
func (s *Session) Mail(expectcode int, cmd, arg string) (int, string, error) {
	return s.Passthru(expectcode, cmd, arg)
}

//Rcpt command backend handler
func (s *Session) Rcpt(expectcode int, cmd, arg string) (int, string, error) {
	return s.Passthru(expectcode, cmd, arg)
}

//Reset command backend handler
func (s *Session) Reset(expectcode int, cmd, arg string) (int, string, error) {
	return s.Passthru(expectcode, cmd, arg)
}

//Quit command backend handler
func (s *Session) Quit(expectcode int, cmd, arg string) (int, string, error) {
	return s.Passthru(expectcode, cmd, arg)
}

//Unknown command backend handler
func (s *Session) Unknown(expectcode int, cmd, arg string) (int, string, error) {
	return s.Passthru(expectcode, cmd, arg)
}

// Passthru a command to the upstream server, logging
func (s *Session) Passthru(expectcode int, cmd, arg string) (int, string, error) {
	s.bkd.logger(cmdTwiddle(s), cmd, arg)
	joined := cmd
	if arg != "" {
		joined = cmd + " " + arg
	}
	code, msg, err := s.upstream.MyCmd(expectcode, joined)
	if err != nil {
		s.bkd.loggerAlways(respTwiddle(s), cmd, code, msg, "error", err.Error())
		if code == 0 {
			// some errors don't show up in (code,msg) e.g. TLS cert errors, so map as a specific SMTP code/msg response
			code = 599
			msg = err.Error()
		}
	} else {
		s.bkd.logger(respTwiddle(s), code, msg)
	}
	return code, msg, err
}

// DataCommand pass upstream, returning a place to write the data AND the usual responses
func (s *Session) DataCommand() (io.WriteCloser, int, string, error) {
	s.bkd.logger(cmdTwiddle(s), "DATA")
	w, code, msg, err := s.upstream.Data()
	if err != nil {
		s.bkd.loggerAlways(respTwiddle(s), "DATA error", err.Error())
	}
	return w, code, msg, err
}

// Data body (dot delimited) pass upstream, returning the usual responses
func (s *Session) Data(r io.Reader, w io.WriteCloser) (int, string, error) {
	var w2 io.Writer // If upstream debugging, tee off a copy into the debug file.
	if s.bkd.upstreamDataDebug != nil {
		w2 = io.MultiWriter(w, s.bkd.upstreamDataDebug)
	} else {
		w2 = w
	}
	bytesWritten, err := smtpproxy.MailCopy(w2, r, s.bkd.wrapper) // Pass in the engagement tracking info
	if err != nil {
		msg := "DATA MailCopy error"
		s.bkd.loggerAlways(respTwiddle(s), msg, err.Error())
		return 0, msg, err
	}
	err = w.Close()
	code := s.upstream.DataResponseCode
	msg := s.upstream.DataResponseMsg
	if err != nil {
		s.bkd.loggerAlways(respTwiddle(s), "DATA Close error", err, ", bytes written =", bytesWritten)
	} else {
		s.bkd.logger(respTwiddle(s), "DATA accepted, bytes written =", bytesWritten)
		s.bkd.logger(respTwiddle(s), code, msg)
	}
	if !s.bkd.verbose {
		// Short-form logging - one line per message - used when "verbose" not set
		log.Printf("Message DATA upstream,%d,%d,%s\n", bytesWritten, code, msg)
	}
	return code, msg, err
}
