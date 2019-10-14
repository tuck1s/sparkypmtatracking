package sparkypmtatracking_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	trk "github.com/tuck1s/sparkyPMTATracking"
)

const testHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>test mail</title>
</head>
<body>
  Click <a href="https://sparkpost.com/">SparkPost </a>
  <p>This is a very long line of text indeed containing !"#$%&'()*+,-./0123456789:; escaped
    ?@ABCDEFGHIJKLMNOPQRSTUVWXYZ\[ ]^_abcdefghijklmnopqrstuvwxyz ~</p>
  <p>Here's some exotic characters to work the quoted-printable stuff ¡¢£¤¥¦§¨©ª« ¡¢£¤¥¦§¨©ª«
  </p>
</body>
</html>
`
const testMessageID = "0000123456789abcdef0"
const testRecipient = "recipient@example.com"
const testTrackingDomain = "http://tracking.example.com"
const testTrackingPath = "wibble/wobble"

const expectedHTMLoutput = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>test mail</title>
</head>
<body><div style="color:transparent;visibility:hidden;opacity:0;font-size:0px;border:0;max-height:1px;width:1px;margin:0px;padding:0px;border-width:0px!important;display:none!important;line-height:0px!important;"><img border="0" width="1" height="1" src="http://tracking.example.com/eJwUy1sKhSAQBuC9_M9ymHPr4lM7ERunGNASMwiivUcL-E54rrBQGFS3lwgLGKRtdhpgQUT0_nx__6btej9ykIlgUDg_qghrVlnqIIdPOcqL14TrDgAA__8UYRn2"/></div>

  Click <a href="http://tracking.example.com/eJwUzE0KAjEMBtC7fOti47925U2GmolanDohiSCId5c5wHtfVA4UMBJieNuEgkeEesnZtdpTZ48Vzz0joft9aCMKiIjWm-1ufziezvXKo9wICca6XCbctMkrLvKpXSdZPH7_AAAA__8PHCI-">SparkPost </a>
  <p>This is a very long line of text indeed containing !"#$%&'()*+,-./0123456789:; escaped
    ?@ABCDEFGHIJKLMNOPQRSTUVWXYZ\[ ]^_abcdefghijklmnopqrstuvwxyz ~</p>
  <p>Here's some exotic characters to work the quoted-printable stuff ¡¢£¤¥¦§¨©ª« ¡¢£¤¥¦§¨©ª«
  </p>
<img border="0" width="1" height="1" alt="" src="http://tracking.example.com/eJwUy10KhCAQB_C7_J9lmd3t06duIjZOIWiKGQTR3aMD_C5YrtBIUKjmKAEaUIj7aryDBhHR9_dv2q4fRjuzk4WgUDi_qgj77GWrk5w25iAfThH3EwAA__8WLxn8">
</body>
</html>
`

const expectedHTMLoutputLongPath = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>test mail</title>
</head>
<body><div style="color:transparent;visibility:hidden;opacity:0;font-size:0px;border:0;max-height:1px;width:1px;margin:0px;padding:0px;border-width:0px!important;display:none!important;line-height:0px!important;"><img border="0" width="1" height="1" src="http://tracking.example.com/wibble/wobble/eJwUy1sKhSAQBuC9_M9ymHPr4lM7ERunGNASMwiivUcL-E54rrBQGFS3lwgLGKRtdhpgQUT0_nx__6btej9ykIlgUDg_qghrVlnqIIdPOcqL14TrDgAA__8UYRn2"/></div>

  Click <a href="http://tracking.example.com/wibble/wobble/eJwUzE0KAjEMBtC7fOti47925U2GmolanDohiSCId5c5wHtfVA4UMBJieNuEgkeEesnZtdpTZ48Vzz0joft9aCMKiIjWm-1ufziezvXKo9wICca6XCbctMkrLvKpXSdZPH7_AAAA__8PHCI-">SparkPost </a>
  <p>This is a very long line of text indeed containing !"#$%&'()*+,-./0123456789:; escaped
    ?@ABCDEFGHIJKLMNOPQRSTUVWXYZ\[ ]^_abcdefghijklmnopqrstuvwxyz ~</p>
  <p>Here's some exotic characters to work the quoted-printable stuff ¡¢£¤¥¦§¨©ª« ¡¢£¤¥¦§¨©ª«
  </p>
<img border="0" width="1" height="1" alt="" src="http://tracking.example.com/wibble/wobble/eJwUy10KhCAQB_C7_J9lmd3t06duIjZOIWiKGQTR3aMD_C5YrtBIUKjmKAEaUIj7aryDBhHR9_dv2q4fRjuzk4WgUDi_qgj77GWrk5w25iAfThH3EwAA__8WLxn8">
</body>
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
		t.Errorf("Count of copied bytes differs: got %d, expexcted %d\n", n, len(expected))
	}
}

func TestTrackHTML(t *testing.T) {
	myWrapper, err := trk.NewWrapper(testTrackingDomain)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	if myWrapper.URL.String() != testTrackingDomain {
		t.Errorf("Tracking domain set wrongly to %s", myWrapper.URL.String())
	}
	myWrapper.SetMessageInfo(testMessageID, testRecipient)
	ioHarness(testHTML, expectedHTMLoutput, myWrapper.TrackHTML, t)

	// with a trailing "/"
	expectedURL := testTrackingDomain + "/"
	myWrapper, err = trk.NewWrapper(expectedURL)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	if myWrapper.URL.String() != expectedURL {
		t.Errorf("Tracking domain set to %s", myWrapper.URL.String())
	}
	myWrapper.SetMessageInfo(testMessageID, testRecipient)
	ioHarness(testHTML, expectedHTMLoutput, myWrapper.TrackHTML, t)

	// with a longer path
	expectedURL = testTrackingDomain + "/" + testTrackingPath
	myWrapper, err = trk.NewWrapper(expectedURL)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	if myWrapper.URL.String() != expectedURL {
		t.Errorf("Tracking domain set to %s", myWrapper.URL.String())
	}
	myWrapper.SetMessageInfo(testMessageID, testRecipient)
	ioHarness(testHTML, expectedHTMLoutputLongPath, myWrapper.TrackHTML, t)
}

func TestTrackHTMLfaultyInputs(t *testing.T) {
	// With uninitialised tracker, pixels should return empty string
	myTracker := trk.Wrapper{}
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
	_, err := trk.NewWrapper("httttps://not a url")
	if err == nil {
		t.Errorf("Faulty input test should have failed")
	}

	_, err = trk.NewWrapper("https://example.com/?pet=dog")
	if err == nil {
		t.Errorf("Faulty input test should have failed")
	}
}

func TestUniqMessageID(t *testing.T) {
	x := trk.UniqMessageID()
	if len(x) != 21 {
		t.Errorf("Unexpected result from UniqMessageID")
	}

	y := trk.UniqMessageID()
	if x == y {
		t.Errorf("UniqMessageID returned two consecutive identical values %s and %s. Pigs are now flying", x, y)
	}
}

func TestTrackHTMLFaultyInputs(t *testing.T) {
	myTracker, err := trk.NewWrapper(testTrackingDomain)
	if err != nil {
		t.Errorf("Error returned from NewWrapper: %v", err)
	}
	myTracker.SetMessageInfo(testMessageID, testRecipient)

	// Make faulty HTML
	faultyHTML := "<htm  thats all folks"
	ioHarness(faultyHTML, "", myTracker.TrackHTML, t)
}
