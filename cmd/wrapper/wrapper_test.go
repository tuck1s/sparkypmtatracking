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
// calls the "under test" io stream "copy-like" function f, and compares the returned output with expected
func ioHarness(in, expected string, f func(io.Reader, io.Writer) error, t *testing.T) {
	r := strings.NewReader(in)
	var outbuf bytes.Buffer
	err := f(r, &outbuf)
	if err != nil {
		t.Errorf("Error returned from myTracker.TrackHTML: %v", err)
	}
	got := outbuf.String()
	if got != expected {
		t.Errorf("Got and expected values differ:\n---Got\n%s\n\n---Expected\n%s\n", got, expected)
	}
}

func TestTrackHTML(t *testing.T) {

	myTracker, err := trk.NewTracker(testTrackingDomain)
	if err != nil {
		t.Errorf("Error returned from myTracker.NewTracker: %v", err)
	}
	if myTracker.URL.String() != testTrackingDomain {
		t.Errorf("Tracking domain set wrongly to %s", myTracker.URL.String())
	}
	myTracker.MessageInfo(testMessageID, testRecipient)
	ioHarness(testHTML, expectedHTMLoutput, myTracker.TrackHTML, t)

	// with a trailing "/"
	expectedURL := testTrackingDomain + "/"
	myTracker, err = trk.NewTracker(expectedURL)
	if err != nil {
		t.Errorf("Error returned from myTracker.NewTracker: %v", err)
	}
	if myTracker.URL.String() != expectedURL {
		t.Errorf("Tracking domain set to %s", myTracker.URL.String())
	}
	myTracker.MessageInfo(testMessageID, testRecipient)
	ioHarness(testHTML, expectedHTMLoutput, myTracker.TrackHTML, t)

	// with a longer path
	expectedURL = testTrackingDomain + "/" + testTrackingPath
	myTracker, err = trk.NewTracker(expectedURL)
	if err != nil {
		t.Errorf("Error returned from myTracker.NewTracker: %v", err)
	}
	if myTracker.URL.String() != expectedURL {
		t.Errorf("Tracking domain set to %s", myTracker.URL.String())
	}
	myTracker.MessageInfo(testMessageID, testRecipient)
	ioHarness(testHTML, expectedHTMLoutputLongPath, myTracker.TrackHTML, t)

	// faulty inputs: malformed URLs are rejected
	_, err = trk.NewTracker("httttps://not a url")
	if err == nil {
		t.Errorf("Faulty input test should have failed")
	}
}
