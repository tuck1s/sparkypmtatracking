package sparkypmtatracking_test

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/tuck1s/go-smtpproxy"
	spmta "github.com/tuck1s/sparkyPMTATracking"
)

// string params: initial_pixel, testTargetURL, end_pixel
const testHTMLTemplate1 = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>test mail</title>
</head>
<body>%s
  Click <a href="%s">SparkPost</a>
  <p>This is a very long line of text indeed containing !"#$%%&'()*+,-./0123456789:; escaped
    ?@ABCDEFGHIJKLMNOPQRSTUVWXYZ\[ ]^_abcdefghijklmnopqrstuvwxyz ~</p>
  <p>Here's some exotic characters to work the quoted-printable stuff ¡¢£¤¥¦§¨©ª« ¡¢£¤¥¦§¨©ª«
  </p>
  Click <a href="%s">Another tracked link</a>
%s</body>
</html>
`

// ioHarness takes input as a string, expected output as a string,
// calls the "io.Copy-like" function f (the function under test), and compares the returned output with expected
func ioHarness(in, expected string, f func(io.Writer, io.Reader) (n int, e error), t *testing.T) {
	r := strings.NewReader(in)
	var outbuf bytes.Buffer
	n, err := f(&outbuf, r) // Note order (dest, src) a la io.Copy
	if err != nil {
		t.Errorf("Error returned from myTracker.TrackHTML: %v", err)
	}
	got := outbuf.String()
	if got != expected {
		t.Errorf("Got and expected values differ:\n---Got\n%s\n\n---Expected\n%s\n", got, expected)
	}
	if n != len(expected) {
		t.Errorf("Count of copied bytes differs: got %d, expected %d\n", n, len(expected))
	}
}

func testHTML(htmlTemplate, URL1, URL2 string) string {
	return fmt.Sprintf(htmlTemplate, "", URL1, URL2, "")
}

func expectedHTMLoutput(htmlTemplate, URL1, URL2 string, w *spmta.Wrapper) string {
	return fmt.Sprintf(htmlTemplate, w.InitialOpenPixel(), w.WrapURL(URL1), w.WrapURL(URL2), w.OpenPixel())
}

func testHTMLWrapping(htmlTemplate string, trkDomain string, URL1 string, URL2 string, msgID string, recip string, t *testing.T) {
	w, err := spmta.NewWrapper(trkDomain)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	if w.URL.String() != trkDomain {
		t.Errorf("Tracking domain set wrongly to %s", w.URL.String())
	}
	w.SetMessageInfo(msgID, recip)
	testHTML := testHTML(htmlTemplate, URL1, URL2)
	expectedHTMLoutput := expectedHTMLoutput(htmlTemplate, URL1, URL2, w)
	ioHarness(testHTML, expectedHTMLoutput, w.TrackHTML, t) // run the test
}

func RandomWord() string {
	const dict = "abcdefghijklmnopqrstuvwxyz"
	l := rand.Intn(20) + 1
	s := ""
	for ; l > 0; l-- {
		p := rand.Intn(len(dict))
		s = s + string(dict[p])
	}
	return s
}

// A completely random URL (not belonging to any actual TLD), pathlen should be >= 0
func RandomURL(pathlen int) string {
	var method string
	if rand.Intn(2) > 0 {
		method = "https"
	} else {
		method = "http"
	}
	path := ""
	for ; pathlen > 0; pathlen-- {
		path = path + "/" + RandomWord()
	}
	return method + "://" + RandomWord() + "." + RandomWord() + path
}

func RandomBaseURL() string {
	return RandomURL(0)
}

func RandomURLWithPath() string {
	return RandomURL(rand.Intn(4))
}

func RandomRecipient() string {
	return RandomWord() + "@" + RandomWord() + "." + RandomWord()
}

func TestTrackHTML(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	// Run several times with randomised contents
	for i := 0; i <= 100; i++ {
		msgID := spmta.UniqMessageID()
		trkDomain := RandomBaseURL()
		testTargetURL := RandomURLWithPath()
		testTargetURL2 := RandomURLWithPath()
		testHTMLWrapping(testHTMLTemplate1, trkDomain, testTargetURL, testTargetURL2, msgID, RandomRecipient(), t)
	}
}

func TestTrackHTMLfaultyInputs(t *testing.T) {
	// With uninitialised tracker, pixels should return empty string
	myTracker := spmta.Wrapper{}
	s := myTracker.InitialOpenPixel()
	if s != "" {
		t.Errorf("Expecting empty result from InitialOpenPixel, got %s", s)
	}

	s = myTracker.OpenPixel()
	if s != "" {
		t.Errorf("Expecting empty result from InitialOpenPixel, got %s", s)
	}

	// With uninitialised tracker, wrapURL should return value identical to input
	u := "https://xyzzy.org/foo/bar/?pet=pig"
	s = myTracker.WrapURL(u)
	if s != u {
		t.Errorf("Expecting empty result from InitialOpenPixel, got %s", s)
	}
}

func TestNewTracker(t *testing.T) {
	// faulty inputs: malformed URLs are rejected
	_, err := spmta.NewWrapper("httttps://not a url")
	if err == nil {
		t.Errorf("Faulty input test should have failed")
	}

	_, err = spmta.NewWrapper("https://example.com/?pet=dog")
	if err == nil {
		t.Errorf("Faulty input test should have failed")
	}
}

func TestUniqMessageID(t *testing.T) {
	x := spmta.UniqMessageID()
	if len(x) != 20 {
		t.Errorf("Unexpected result from UniqMessageID")
	}

	y := spmta.UniqMessageID()
	if x == y {
		t.Errorf("UniqMessageID returned two consecutive identical values %s and %s. Pigs are now flying", x, y)
	}
}

func TestActionToType(t *testing.T) {
	aList := [][]string{
		{"c", "click"},
		{"o", "open"},
		{"i", "initial_open"},
		{"", ""},
		{" ", ""},
		{"cats_dogs", ""},
	}
	// Check responses
	for _, a := range aList {
		if spmta.ActionToType(a[0]) != a[1] {
			t.Errorf("Unexpected value returned from ActionToType")
		}
	}
}

func TestTrackHTMLFaultyInputs(t *testing.T) {
	msgID := spmta.UniqMessageID()
	trkDomain := RandomBaseURL()
	myTracker, err := spmta.NewWrapper(trkDomain)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	myTracker.SetMessageInfo(msgID, RandomRecipient())

	// Make faulty HTML
	faultyHTML := "<htm  thats all folks"
	ioHarness(faultyHTML, "", myTracker.TrackHTML, t)
}

func TestEncodeDecodePath(t *testing.T) {
	const MAX = 1000
	rand.Seed(time.Now().UTC().UnixNano())
	big := make([]byte, MAX)
	for i := 0; i <= MAX; i++ {
		expected := big[0:i]
		_, err := rand.Read(expected) // Generate random byte string of varying length
		if err != nil {
			t.Errorf(err.Error())
		}
		enc, err := spmta.EncodePath(expected)
		if err != nil {
			t.Errorf(err.Error())
		}
		got, err := spmta.DecodePath(enc)
		if err != nil {
			t.Errorf(err.Error())
		}
		if bytes.Compare(expected, got) != 0 {
			t.Errorf("EncodePath / DecodePath mismatch\nGot and expected values differ:\n---Got\n%s\n\n---Expected\n%s\n", got, expected)

		}
	}
}

func testED(eType, action, link string, t *testing.T) {
	trkDomain := RandomBaseURL()
	msgID := spmta.UniqMessageID()
	recip := RandomRecipient()
	url, err := spmta.EncodeLink(trkDomain, eType, msgID, recip, link)
	if err != nil {
		t.Error(err)
	}
	eBytes, wd, _, err := spmta.DecodeLink(url)
	if err != nil {
		t.Error(err)
	}
	got := string(eBytes)
	expected := fmt.Sprintf(`{"act":"%s","t_url":"%s","msg_id":"%s","rcpt":"%s"}`,
		action,
		link,
		msgID,
		recip)
	if got != expected {
		t.Errorf("EncodeLink/DecodeLink JSON Got and expected values differ:\n---Got\n%s\n\n---Expected\n%s\n", got, expected)
	}
	if wd.Action != action || wd.TargetLinkURL != link || wd.MessageID != msgID || wd.RcptTo != recip {
		t.Errorf("EncodeLink/DecodeLink Decoded unexpected value:\n---Got\n%s\n", wd)
	}
}

func TestEncodeDecodeLink(t *testing.T) {
	testED("click", "c", RandomURLWithPath(), t)
	testED("open", "o", "", t)
	testED("initial_open", "i", "", t)
}

func TestEncodeDecodeLinkFaultyInputs(t *testing.T) {
	msgID := spmta.UniqMessageID()
	recip := RandomRecipient()
	link := RandomURLWithPath()
	trkDomain := RandomBaseURL()

	// Faulty inputs - invalid action
	url, err := spmta.EncodeLink(trkDomain, "pigs", msgID, recip, link)
	if err.Error() != "Invalid encodeAction" {
		t.Error(err)
	}

	// Faulty inputs - blank tracking domain
	url, err = spmta.EncodeLink("", "click", msgID, recip, link)
	if err.Error() != "parse : empty url" {
		t.Error(err)
	}
	// invalid tracking domain
	url, err = spmta.EncodeLink("notaurl", "click", msgID, recip, link)
	if err.Error() != "parse notaurl: invalid URI for request" {
		t.Error(err)
	}
	fmt.Println(url, err)
	// got query parameter
	url, err = spmta.EncodeLink("https://example.com?pets=dog", "click", msgID, recip, link)
	if err.Error() != "Can't have query parameters in the tracking URL" {
		t.Error(err)
	}
	fmt.Println(url, err)
}

// Test functions that are usually called back by the smtpproxy
func TestWrapperActive(t *testing.T) {
	// start with nil value, should return false
	var w *spmta.Wrapper
	a := w.Active()
	if a {
		t.Errorf("Active() return value %v, expected false", a)
	}

	trkDomain := RandomBaseURL()
	w, err := spmta.NewWrapper(trkDomain)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	a = w.Active()
	if !a {
		t.Errorf("Active() return value %v, expected true", a)
	}
}

const testEmailTemplate = `To: Bob <bob.lumreeker@gmail.com>
Content-Type: multipart/alternative; boundary="D7F------------D7FD5A0B8AB9C65CCDBFA872"
MIME-Version: 1.0
Subject: Fresh, tasty Avocados delivered straight to your door!
From: "Steve At SparkPost" <steve@example.com>

--D7F------------D7FD5A0B8AB9C65CCDBFA872
Content-Transfer-Encoding: 7bit
Content-Type: text/plain; charset="UTF-8"

Spicy jalapeno bacon ipsum dolor amet pariatur mollit fatback venison, cillum occaecat quis ut labore pork belly culpa ea bacon in spare ribs.

--D7F------------D7FD5A0B8AB9C65CCDBFA872
Content-Transfer-Encoding: 8bit
Content-Type: text/html; charset="UTF-8"

%s
--D7F------------D7FD5A0B8AB9C65CCDBFA872`

func TestProcessMessageHeadersAndBody(t *testing.T) {
	// create HTML body contents, then place that inside an email
	URL1 := RandomURLWithPath()
	URL2 := RandomURLWithPath()
	testHTML := testHTML(testHTMLTemplate1, URL1, URL2)
	testEmail := fmt.Sprintf(testEmailTemplate, testHTML)
	message, err := mail.ReadMessage(strings.NewReader(testEmail))
	if err != nil {
		t.Error(err)
	}
	// Prepare to wrap
	trkDomain := RandomBaseURL()
	w, err := spmta.NewWrapper(trkDomain)
	if err != nil {
		t.Error(err)
	}
	err = w.ProcessMessageHeaders(message.Header)
	if err != nil {
		t.Error(err)
	}
	// Check that the message ID header was added
	msgID := message.Header.Get(spmta.SparkPostMessageIDHeader)
	if len(msgID) != 20 {
		t.Errorf("message ID header %s should be 20 chars long", spmta.SparkPostMessageIDHeader)
	}
	// Handle the message body, via the external smtpproxy lib, grabbing the output into a buffer
	var outbuf bytes.Buffer
	bw, err := smtpproxy.HandleMessageBody(&outbuf, message.Header, message.Body, w)
	if bw < len(testEmail) {
		t.Errorf("A surprisingly small email, bw=%d", bw)
	}
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
