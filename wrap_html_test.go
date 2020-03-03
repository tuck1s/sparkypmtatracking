package sparkypmtatracking_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	spmta "github.com/tuck1s/sparkypmtatracking"
)

//-----------------------------------------------------------------------------
// test email & html templates

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

const testTextTemplate1 = `Spicy jalapeno bacon ipsum dolor amet pariatur mollit fatback venison, cillum occaecat quis ut labore pork belly culpa ea bacon in spare ribs.`

// string params: to, from, bound, bound, testTextTemplate1, bound, testHTML, bound
const testEmailTemplate = `To: %s
From: %s
Subject: Fresh, tasty Avocados delivered straight to your door!
Content-Type: multipart/alternative; boundary="%s"
MIME-Version: 1.0

--%s
Content-Transfer-Encoding: 7bit
Content-Type: text/plain; charset="UTF-8"

%s

--%s
Content-Transfer-Encoding: 8bit
Content-Type: text/html; charset="UTF-8"

%s
--%s--
`

func testHTML(htmlTemplate, URL1, URL2 string) string {
	return fmt.Sprintf(htmlTemplate, "", URL1, URL2, "")
}

// RandomTestEmail creates HTML body contents, and places inside an email
func RandomTestEmail() string {
	URL1 := RandomURLWithPath()
	URL2 := RandomURLWithPath()
	testHTML := testHTML(testHTMLTemplate1, URL1, URL2)
	to := RandomRecipient()
	from := RandomRecipient()
	u := uuid.New() // randomise boundary marker
	bound := fmt.Sprintf("%0x", u[:8])
	return fmt.Sprintf(testEmailTemplate, to, from, bound, bound, testTextTemplate1, bound, testHTML, bound)
}

// string params: to, from, bound, bound, bound, base64body, bound
const base64HTMLEmailTemplate = `To: %s
From: %s
Subject: A base64 encoded html email
Content-Type: multipart/alternative; boundary="%s"
MIME-Version: 1.0

--%s
Content-Transfer-Encoding: 7bit
Content-Type: text/plain; charset="UTF-8"

Short plaintext

--%s
Content-Transfer-Encoding: base64
Content-Type: text/html; charset="utf-8"

%s
--%s--
`

// Base64HTMLTestEmail creates HTML body contents, and places inside an email
func Base64HTMLTestEmail() string {
	URL1 := RandomURLWithPath()
	URL2 := RandomURLWithPath()
	testHTML := testHTML(testHTMLTemplate1, URL1, URL2)
	base64Body := base64.StdEncoding.EncodeToString([]byte(testHTML))
	to := RandomRecipient()
	from := RandomRecipient()
	u := uuid.New() // randomise boundary marker
	bound := fmt.Sprintf("%0x", u[:8])
	return fmt.Sprintf(base64HTMLEmailTemplate, to, from, bound, bound, bound, base64Body, bound)
}

const nestedEmailRFC822Template = `To: %s
From: %s
Subject: An email carrying a nested email
Content-Type: multipart/mixed; boundary="%s"
MIME-Version: 1.0

This is the original MIME multi-part message
--%s
Content-Transfer-Encoding: 7bit
Content-Type: text/plain; charset="UTF-8"

%s
--%s
Content-Type: image/gif; name="map_of_Argentina.gif"
Content-Transfer-Encoding: base64
Content-Disposition: in1ine; fi1ename="map_of_Argentina.gif"

R0lGODlhJQA1AKIAAP/////78P/omn19fQAAAAAAAAAAAAAAACwAAAAAJQA1AAAD7Qi63P5w
wEmjBCLrnQnhYCgM1wh+pkgqqeC9XrutmBm7hAK3tP31gFcAiFKVQrGFR6kscnonTe7FAAad
GugmRu3CmiBt57fsVq3Y0VFKnpYdxPC6M7Ze4crnnHum4oN6LFJ1bn5NXTN7OF5fQkN5WYow
BEN2dkGQGWJtSzqGTICJgnQuTJN/WJsojad9qXMuhIWdjXKjY4tenjo6tjVssk2gaWq3uGNX
U6ZGxseyk8SasGw3J9GRzdTQky1iHNvcPNNI4TLeKdfMvy0vMqLrItvuxfDW8ubjueDtJufz
7itICBxISKDBgwgTKjyYAAA7
--%s
Content-Transfer-Encoding: 8bit
Content-Type: message/rfc822
Content-Disposition: inline

%s
--%s
`

// NestedEmailRFC822 creates a multipart message, which contains an attachment in base64, and another message!
func NestedEmailRFC822() string {
	innerMsg := RandomTestEmail()
	to := RandomRecipient()
	from := RandomRecipient()
	bound := "boundybound" // fmt.Sprintf("%0x", uuid.New()) // randomise boundary marker
	testText := "this is some plain text"
	return fmt.Sprintf(nestedEmailRFC822Template, to, from, bound, bound, testText, bound, bound, innerMsg, bound)
}

const plainEmailTemplate = `To: %s
From: %s
Subject: A plaintext email
MIME-Version: 1.0

short plaintext
`

func PlainEmail() string {
	to := RandomRecipient()
	from := RandomRecipient()
	return fmt.Sprintf(plainEmailTemplate, to, from)
}

//-----------------------------------------------------------------------------

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

func expectedHTMLoutput(htmlTemplate, URL1, URL2 string, w *spmta.Wrapper) string {
	return fmt.Sprintf(htmlTemplate, w.InitialOpenPixel(), w.WrapURL(URL1), w.WrapURL(URL2), w.OpenPixel())
}

func testHTMLWrapping(htmlTemplate string, trkDomain string, URL1 string, URL2 string, msgID string, recip string, t *testing.T) {
	// work through all combinations of tracking flags
	var boolz = []bool{false, true}
	for _, trackOpen := range boolz {
		for _, trackInitialOpen := range boolz {
			for _, trackLink := range boolz {
				w, err := spmta.NewWrapper(trkDomain, trackOpen, trackInitialOpen, trackLink)
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
		}
	}
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

//-----------------------------------------------------------------------------
// Tests

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

func TestWrapperMethodsfaultyInputs(t *testing.T) {
	// With uninitialised tracker, pixels should return empty string
	w := spmta.Wrapper{}
	s := w.InitialOpenPixel()
	if s != "" {
		t.Errorf("Expecting empty result from InitialOpenPixel, got %s", s)
	}

	s = w.OpenPixel()
	if s != "" {
		t.Errorf("Expecting empty result from InitialOpenPixel, got %s", s)
	}

	// With uninitialised tracker, wrapURL should return value identical to input
	u := "https://xyzzy.org/foo/bar/?pet=pig"
	s = w.WrapURL(u)
	if s != u {
		t.Errorf("Expecting empty result from InitialOpenPixel, got %s", s)
	}
}

func TestNewWrapper(t *testing.T) {
	// faulty inputs: malformed URLs are rejected
	_, err := spmta.NewWrapper("httttps://not a url", true, true, true)
	if err == nil {
		t.Errorf("Faulty input test should have failed")
	}

	_, err = spmta.NewWrapper("https://example.com/?pet=dog", true, true, true)
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
	myTracker, err := spmta.NewWrapper(trkDomain, true, true, true)
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
			t.Error(err)
		}
		enc, err := spmta.EncodePath(expected)
		if err != nil {
			t.Error(err)
		}
		got, err := spmta.DecodePath(enc)
		if err != nil {
			t.Error(err)
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
	url, err := spmta.EncodeLink(trkDomain, eType, msgID, recip, link, true, true, true)
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

	// faulty inputs to EecodeLink
	eList := [][]string{
		{"Invalid encodeAction", trkDomain, "pigs"},                              // invalid action
		{"empty url", "", "click"},                                               // blank tracking domain
		{"invalid URI", "notaurl", "click"},                                      // invalid tracking domain
		{"Can't have query parameters", "https://example.com?pets=dog", "click"}, // got query parameter
	}
	for _, e := range eList {
		url, err := spmta.EncodeLink(e[1], e[2], msgID, recip, link, true, true, true)
		if !strings.Contains(err.Error(), e[0]) {
			t.Error(err)
		}
		_ = url
	}

	// faulty inputs to DecodeLink
	dList := [][]string{
		// chopped short base64 data
		{"illegal base64 data", "https://sjfbcxoeff.swxpldnj/eJwUzFEOgjAMBuC7_M"},
		// invalid zlib data (penultimate character changed)
		{"invalid checksum", "https://sjfbcxoeff.swxpldnj/eJwUzEsOgjAQBuC7_GsCxdBEu_ImZGb6QqlBOojBeHfjBb4PSBQOggY6busMh6y6VNd1dNv3rFlUNmrz-_6MnmLHR2KapKBBqWmcPByMMeZsrT-xDcOF-9gPjAarLH-br_XxmpOGQm048P0FAAD__7gCJdP="},
		// empty
		{"Invalid link path", ""},
		// not a valid URL (contains nonprintable ASCII)
		{"invalid control character in URL", "\n"},
		// too many / separators
		{"Invalid link path", "https://cats.dogs/too/many/separators"},
	}
	for _, d := range dList {
		_, _, _, err := spmta.DecodeLink(d[1])
		if !strings.Contains(err.Error(), d[0]) {
			t.Error(err)
		}
	}
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
	w, err := spmta.NewWrapper(trkDomain, true, true, true)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	a = w.Active()
	if !a {
		t.Errorf("Active() return value %v, expected true", a)
	}
}
