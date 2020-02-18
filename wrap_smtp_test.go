package sparkypmtatracking_test

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"io"
	"io/ioutil"
	"math/rand"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	smtpproxy "github.com/tuck1s/go-smtpproxy"
	spmta "github.com/tuck1s/sparkyPMTATracking"
)

// localhostCert is a PEM-encoded TLS cert.pem, made for domain test.example.com
//		openssl req -nodes -new -x509 -keyout key.pem -out cert.pem
var localhostCert = []byte(`
-----BEGIN CERTIFICATE-----
MIIDvDCCAqQCCQDG9Km7C037rDANBgkqhkiG9w0BAQsFADCBnzELMAkGA1UEBhMC
dWsxDzANBgNVBAgMBkxvbmRvbjEPMA0GA1UEBwwGTG9uZG9uMRIwEAYDVQQKDAlT
cGFya1Bvc3QxHjAcBgNVBAsMFU1lc3NhZ2luZyBFbmdpbmVlcmluZzEZMBcGA1UE
AwwQdGVzdC5leGFtcGxlLmNvbTEfMB0GCSqGSIb3DQEJARYQdGVzdEBleGFtcGxl
LmNvbTAeFw0yMDAyMDYyMTIyMDNaFw0yMDAzMDcyMTIyMDNaMIGfMQswCQYDVQQG
EwJ1azEPMA0GA1UECAwGTG9uZG9uMQ8wDQYDVQQHDAZMb25kb24xEjAQBgNVBAoM
CVNwYXJrUG9zdDEeMBwGA1UECwwVTWVzc2FnaW5nIEVuZ2luZWVyaW5nMRkwFwYD
VQQDDBB0ZXN0LmV4YW1wbGUuY29tMR8wHQYJKoZIhvcNAQkBFhB0ZXN0QGV4YW1w
bGUuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4yGJRYAI6xtQ
ZPRxIWU+ZjlKo66LFvfr2VrWd30m8dflB0CNMkaaEMGt29jwLvzkP/mfn5dYVw3E
dFJ2yBGR3wDy02ssmBVaOYkbYxgxeFa9jIgBLJONA3HIJRjn91/3lSCxDo6cE7l+
ufhf8pc78YBZvhbC50kBajQtYaENcca9asj5cCRHS44hL7sCzN4kGETkg1jYtocT
CMjJIgQ3dJool7M9MEAafWiFnIcO76O/jxewggLgOkfj7i9Y1iP6aWScEq6nNkW7
8xFNqFafnK7W85TzkpfRIN/ntpEwgPcUHG4b4AWpXWR6q+1do25WgaWvt/od45KN
aIo1kylOwQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQCODjmvtqracVuOjsRp7841
7glTqDQeXIUr3X7UvDTyvl70oeGeqnaEs3hO79T15gz0pKlKbYlB3B4v7fldmrLU
mu0uQ7W112NBXYt71wpwuVQWdWSRi9rcAyvuf2nHLZ9fVjczxbCAi+QUFVY+ERoO
CfngvPkPQvLB7VT/oKXKN+j8bXBJ+fYLA6fX4kzpuwx9hf+ay9x+JpPAB/dPEDjB
KsbnfZsIPeuERAlWoSX/c9ggXPXzh95oZz6RhicmtPy3z2ZYJL4BsgEtbazOc6aO
7c/t3Z1FScoSgCql4MXv9kLVL2LNGTWja89pnFnRaobagQ7XB0MEUotrM0ow18SM
-----END CERTIFICATE-----`)

// corresponding private key.pem
var localhostKey = []byte(`
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDjIYlFgAjrG1Bk
9HEhZT5mOUqjrosW9+vZWtZ3fSbx1+UHQI0yRpoQwa3b2PAu/OQ/+Z+fl1hXDcR0
UnbIEZHfAPLTayyYFVo5iRtjGDF4Vr2MiAEsk40DccglGOf3X/eVILEOjpwTuX65
+F/ylzvxgFm+FsLnSQFqNC1hoQ1xxr1qyPlwJEdLjiEvuwLM3iQYROSDWNi2hxMI
yMkiBDd0miiXsz0wQBp9aIWchw7vo7+PF7CCAuA6R+PuL1jWI/ppZJwSrqc2Rbvz
EU2oVp+crtbzlPOSl9Eg3+e2kTCA9xQcbhvgBaldZHqr7V2jblaBpa+3+h3jko1o
ijWTKU7BAgMBAAECggEAHbtvH8Tx5ezuajjBcnCxaWpIhgK8PGZ53jsQ5hVg+rmb
RobBtPofAuCHpMbSMiRysJk5twd1zfeEZwHAgNIj+UBDiT93V/U7mVqEVkV9fFZG
e9X16WLrS68iVxDalLxgSYo9Az3R2pcmqquDy9rWQvfdR4/tNZ+N6twnsKcHfoQZ
Z2lIZrmbR1ZqAEK7T7J5rm2WR+430cuTGEl/X39iIVimwo9QZIs6VikYRYyJoS8u
8VtNsPY7lhnoPctMyErzWeslZXThFmuA5xqtEgFai51dhiJd/+iLkKtbHkfiLeF9
ej+b40LnPT/rnYkBkyyvp2vVXnEUxPEAOzImzE8bQQKBgQD8TP5/Lg/lGK6CcSjD
XG3/w0sfFQtC+oN3I/iFv/tgTQQRF/el7uF79si31TicZPDJgKbnuOGkOdSEyl4u
Mg4yEwX4e+Grb13aENZb5p+fyN91P0jD+4lzLm6k4RaSN/EkDEe9LSn+wIUedO/A
iG4S79EPyYo8pWdNUBO4ZQx3uQKBgQDmdhFiPIdynNDWy1IxhVUnrUuDMyUKFNZB
Rd3KgABgfOBcdB9oeFEijsH86DI2kjHO+rVyCC9F1s8H5VC3eDKtuUaExqBixtu6
TB3BXX+ZapiH8dThXtIa8vteTD5MHLC7pDcESVGzJH3vhdcOhek7es8j78vXZRZq
q/teONQDSQKBgGBh2WckZZYTU7cpG3VmPe9S38PD+kVgBhDhgPM3YARt53vQOB7/
nswIfq0bm0DDnuibaSdkjW57WSBRXqEvJhUjB0jhqlgfdy7y97Cr7ZbQ2eykfFvC
H8QMnOAHzOOW01v+BPnT4xMa4L+91Eks1UAOtULerxxz4365dI8gqx6hAoGAT5iZ
um8jbN9idb01fysI1TJSMVc5xLibo2GpD6aT+r9Gkkf9DQz5INFjiKD9rsFheJY4
ktDm2t0tFhIKhcN65WtnQraDcHo0K6zcXguX5Xnegp1wpAIm2O3xCYmVvp3uIHDA
G7fjAtdos5BrTXXMryFkZ4oLwjIEwwTxRYKlHxkCgYEAi3lkuZl5soQT3d2tkhmc
F6WuDkR4nHxalD05oYtpjAPGpJqwJsyChFAyuUm7kn3qeX0l/Ll4GT6V4KsGQyin
g3Iip0KPOiY+ndAxffTAAiyjFHB7UVe5vfe8NAIU9eBDT8Ibbi2ay9IhQaRABWOc
KnpOfyDnCZbjNekskQaOqiE=
-----END PRIVATE KEY-----`)

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
func mockSMTPServer(t *testing.T, addr string) {
	mockbe := Backend{}
	s := smtpproxy.NewServer(&mockbe)
	s.Addr = addr
	s.ReadTimeout = 60 * time.Second // changeme?
	s.WriteTimeout = 60 * time.Second
	err := s.ServeTLS(localhostCert, localhostKey)
	if err != nil {
		t.Error(err)
	}

	// Begin serving requests
	t.Log("Upstream mock SMTP server listening on", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		t.Error(err)
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

type myWriteCloser struct {
	io.Writer
}

func (myWriteCloser) Close() error {
	return nil
}

// DataCommand pass upstream, returning a place to write the data AND the usual responses
// If you want to see the mail contents, replace Discard with os.Stdout
func (s *Session) DataCommand() (io.WriteCloser, int, string, error) {
	return myWriteCloser{Writer: ioutil.Discard}, 354, `3.0.0 mock says continue.  finished with "\r\n.\r\n"`, nil
}

// Data body (dot delimited) pass upstream, returning the usual responses
func (s *Session) Data(r io.Reader, w io.WriteCloser) (int, string, error) {
	io.Copy(w, r)
	return 250, "2.0.0 OK mock got your dot", nil
}

//-----------------------------------------------------------------------------
// Start proxy server

func startProxy(t *testing.T, s *smtpproxy.Server) {
	t.Log("Proxy (unit under test) listening on", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		t.Error(err)
	}
}

// tlsClientConfig is built from the passed in cert, privkey. InsecureSkipVerify allows self-signed certs to work
func tlsClientConfig(cert []byte, privkey []byte) (*tls.Config, error) {
	cer, err := tls.X509KeyPair(cert, privkey)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	config.InsecureSkipVerify = true
	return config, nil
}

// wrap_smtp tests
func TestWrapSMTP(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	inHostPort := ":5587"
	outHostPort := ":5588"
	verboseOpt := true
	downstreamDebug := "debug_wrap_smtp_test.log"
	upstreamDataDebug := "debug_wrap_smtp_test_upstream.eml"
	wrapURL := "https://track.example.com"
	insecureSkipVerify := true

	// Logging of upstream server DATA (in RFC822 .eml format) for debugging
	var upstreamDebugFile *os.File
	var err error
	if upstreamDataDebug != "" {
		upstreamDebugFile, err = os.OpenFile(upstreamDataDebug, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Proxy writing upstream DATA to", upstreamDebugFile.Name())
		defer upstreamDebugFile.Close()
	}

	myWrapper, err := spmta.NewWrapper(wrapURL)
	if err != nil && !strings.Contains(err.Error(), "empty url") {
		t.Error(err)
	}

	// Set up parameters that the backend will use, and initialise the proxy server parameters
	be := spmta.NewBackend(outHostPort, verboseOpt, upstreamDebugFile, myWrapper, insecureSkipVerify)
	s := smtpproxy.NewServer(be)
	s.Addr = inHostPort
	s.ReadTimeout = 60 * time.Second
	s.WriteTimeout = 60 * time.Second
	err = s.ServeTLS(localhostCert, localhostKey)
	if err != nil {
		t.Error(err)
	}
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
	go mockSMTPServer(t, outHostPort)

	// start the proxy
	go startProxy(t, s)

	// Allow server a little while to start, then send a test mail using standard net/smtp.Client
	c, err := smtp.Dial(inHostPort)
	for i := 0; err != nil && i < 10; i++ {
		time.Sleep(time.Millisecond * 100)
		c, err = smtp.Dial(inHostPort)
	}
	if err != nil {
		t.Error(err)
	}

	// EHLO
	err = c.Hello("testclient.local")
	if err != nil {
		t.Error(err)
	}

	// STARTTLS
	if tls, _ := c.Extension("STARTTLS"); tls {
		// client uses same certs as mock server and proxy, which seems fine for testing purposes
		cfg, err := tlsClientConfig(localhostCert, localhostKey)
		if err != nil {
			t.Error(err)
		}
		err = c.StartTLS(cfg)
		if err != nil {
			t.Error(err)
		}
	}

	// AUTH
	auth := smtp.PlainAuth("", "user@example.com", "password", "")
	err = c.Auth(auth)
	if err != nil {
		t.Error(err)
	}

	// Submit an email .. MAIL FROM, RCPT TO, DATA ... QUIT
	err = c.Mail(RandomRecipient())
	if err != nil {
		t.Error(err)
	}
	err = c.Rcpt(RandomRecipient())
	if err != nil {
		t.Error(err)
	}
	w, err := c.Data()
	if err != nil {
		t.Error(err)
	}
	testEmail := RandomTestEmail()
	r := strings.NewReader(testEmail)
	bytesWritten, err := io.Copy(w, r)
	if err != nil {
		t.Error(err)
	}
	if int(bytesWritten) != len(testEmail) {
		t.Fatalf("Unexpected DATA copy length %v", bytesWritten)
	}

	err = w.Close() // Close the data phase
	if err != nil {
		t.Error(err)
	}

	// Provoke unknown command
	id, err := c.Text.Cmd("WEIRD")
	if err != nil {
		t.Error(err)
	}
	c.Text.StartResponse(id)
	code, msg, err := c.Text.ReadResponse(501)
	t.Log("Response to WEIRD command:", code, msg)
	if code != 501 {
		t.Fatalf("Provoked unknown command - got error %v", err)
	}
	c.Text.EndResponse(id)

	// RESET is not part of the usual happy path for a message ,but we can test
	err = c.Reset()
	if err != nil {
		t.Error(err)
	}

	// QUIT
	err = c.Quit()
	if err != nil {
		t.Error(err)
	}
}

func makeFakeSession(t *testing.T, be *spmta.Backend, url string) smtpproxy.Session {
	c, err := textproto.Dial("tcp", url)
	if err != nil {
		t.Error(err)
	}
	return spmta.MakeSession(&smtpproxy.Client{Text: c}, be)
}

func TestWrapSMTPFaultyInputs(t *testing.T) {
	outHostPort := ":9988"
	verboseOpt := false // vary this from the usual
	wrapURL := "https://track.example.com"
	insecureSkipVerify := true
	var upstreamDebugFile *os.File // placeholder

	myWrapper, err := spmta.NewWrapper(wrapURL)
	if err != nil && !strings.Contains(err.Error(), "empty url") {
		t.Error(err)
	}
	// Set up parameters that the backend will use, and initialise the proxy server parameters
	be := spmta.NewBackend(outHostPort, verboseOpt, upstreamDebugFile, myWrapper, insecureSkipVerify)
	_, err = be.Init() // expect an error
	if err == nil {
		t.Errorf("This test should have returned a non-nil error code")
	}

	const dummyServer = "example.com:80"
	// Provoke error path in Greet (hitting an http server, not an smtp one)
	s := makeFakeSession(t, be, dummyServer)
	caps, code, msg, err := s.Greet("EHLO")
	if err == nil {
		t.Errorf("This test should have returned a non-nil error code")
	}

	// Provoke error path in STARTTLS. Need to get a fresh connection each time
	s = makeFakeSession(t, be, dummyServer)
	code, msg, err = s.StartTLS()
	if err == nil {
		t.Errorf("This test should have returned a non-nil error code")
	}

	// Exercise the session unknown command handler (passthru)
	s = makeFakeSession(t, be, dummyServer)
	code, msg, err = s.Unknown(0, "NONSENSE", "")
	if err == nil {
		t.Errorf("This test should have returned a non-nil error code")
	}

	// Exercise the error paths in DataCommand
	s = makeFakeSession(t, be, dummyServer)
	w, code, msg, err := s.DataCommand()
	if err == nil {
		t.Errorf("This test should have returned a non-nil error code")
	}

	// Exercise the error paths in Data (body)
	s = makeFakeSession(t, be, dummyServer)
	r := strings.NewReader("it is only the hairs on a gooseberry") // this should cause a mailcopy error, as it's not valid RFC822
	code, msg, err = s.Data(r, myWriteCloser{Writer: ioutil.Discard})
	if err == nil {
		t.Errorf("This test should have returned a non-nil error code")
	}

	s = makeFakeSession(t, be, dummyServer)
	input := RandomTestEmail()
	r = strings.NewReader(input) // valid email

	var buf bytes.Buffer
	code, msg, err = s.Data(r, myWriteCloser{Writer: &buf})
	if err != nil {
		t.Error(err)
	}
	// buf now contains the "wrapped" email
	outputMail, err := mail.ReadMessage(&buf)
	if err != nil {
		t.Error(err)
	}
	inputMail, err := mail.ReadMessage(strings.NewReader(input))
	if err != nil {
		t.Error(err)
	}
	// check the headers match
	for hdrType, _ := range inputMail.Header {
		in := inputMail.Header.Get(hdrType)
		out := outputMail.Header.Get(hdrType)
		if in != out {
			t.Errorf("Header %v mismatch", hdrType)
		}
	}
	// output mail should additionally have a message ID
	msgID := outputMail.Header.Get(spmta.SparkPostMessageIDHeader)
	if msgID == "" {
		t.Errorf("outputMail missing message ID header %v", spmta.SparkPostMessageIDHeader)
	}
	// Compare body lengths
	inBody, err := ioutil.ReadAll(inputMail.Body)
	if err != nil {
		t.Error(err)
	}
	outBody, err := ioutil.ReadAll(outputMail.Body)
	if err != nil {
		t.Error(err)
	}
	if len(inBody) > len(outBody) {
		t.Errorf("output mail body short \n%v\n", string(outBody))
	}

	// workaround these variables being "unused"
	_ = caps
	_ = code
	_ = msg
	_ = w
}

func TestProcessMessageHeadersAndParts(t *testing.T) {
	testEmail := RandomTestEmail()
	message, err := mail.ReadMessage(strings.NewReader(testEmail))
	if err != nil {
		t.Error(err)
	}
	// Prepare to wrap
	trkDomain := RandomBaseURL()
	wrap, err := spmta.NewWrapper(trkDomain)
	if err != nil {
		t.Error(err)
	}
	err = wrap.ProcessMessageHeaders(message.Header)
	if err != nil {
		t.Error(err)
	}
	// Check that the message ID header was added
	msgID := message.Header.Get(spmta.SparkPostMessageIDHeader)
	if len(msgID) != 20 {
		t.Errorf("message ID header %s should be 20 chars long", spmta.SparkPostMessageIDHeader)
	}
	// Handle the message body, grabbing the output into a buffer
	var outbuf bytes.Buffer
	err = wrap.HandleMessagePart(&outbuf, message.Body, message.Header.Get("Content-Type"), message.Header.Get("Content-Transfer-Encoding"))
	s := outbuf.String()
	if len(s) < len(testEmail) {
		t.Errorf("A surprisingly small email, len=%d", len(s))
	}
}

func TestProcessMessageHeadersFaultyInputs(t *testing.T) {
	var message mail.Message
	trkDomain := RandomBaseURL()
	w, err := spmta.NewWrapper(trkDomain)
	if err != nil {
		t.Error(err)
	}
	// empty message - missing TO address
	err = w.ProcessMessageHeaders(message.Header)
	if err.Error() != "mail: header not in message" {
		t.Error(err)
	}
	// Correct number of TO addresses
	message.Header = mail.Header{
		"From":    []string{"John Doe <jdoe@machine.example>"},
		"To":      []string{"Mary Smith <mary@example.net>"},
		"Subject": []string{"Saying Hello"},
	}
	err = w.ProcessMessageHeaders(message.Header)
	if err != nil {
		t.Error(err)
	}
	// Too many recipient addresses
	message.Header["Cc"] = []string{"Mary Smith 2<mary2@example.net>"}
	err = w.ProcessMessageHeaders(message.Header)
	if err == nil {
		t.Error(err)
	}
}

// This is the most interesting part of email wrapping, from a benchmarking / performance point of view
func BenchmarkMailCopy(b *testing.B) {
	wrapURL := "https://testing1234.example.com"
	myWrapper, err := spmta.NewWrapper(wrapURL)
	if err != nil {
		b.Fatal(err)
	}
	input := RandomTestEmail()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(input) // valid email
		var buf bytes.Buffer
		err := myWrapper.MailCopy(&buf, r)
		if err != nil {
			b.Fatal(err)
		}
		// buf now contains the "wrapped" email
	}
}
