package sparkypmtatracking_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis"
	spmta "github.com/tuck1s/sparkyPMTATracking"
)

const mockAPIKey = "xyzzy"
const testAction = "c" // click
const testURL = "http://example.com/this/is/a/test"
const testUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko)"
const testIPAddress = "12.34.56.78"
const testTime = 5 * time.Second
const testSubaccountID = 3

// Capture the usual log output into a memory buffer, for later verification
func captureLog() *bytes.Buffer {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0) // disable date/time prefix as it makes hard to compare
	return &buf
}

// Retrieve log output as a string
func retrieveLog(bptr *bytes.Buffer) string {
	return (*bptr).String()
}

// checkHeaders walks through the headers and checks against expected values
func checkHeaders(h http.Header, chk map[string]string) error {
	for k, v := range chk {
		if len(h[k]) != 1 {
			s := errors.New(fmt.Sprintf("Header %s has length %d", k, len(h[k])))
			return s
		}
		// Case-insensitive comparison
		if strings.ToLower(h[k][0]) != strings.ToLower(v) {
			s := errors.New(fmt.Sprintf("Header %s has value %s, was expecting value %s", k, h[k][0], v))
			return s
		}
	}
	return nil
}

// checkResponse compares the mock SparkPost request with what was expected
func checkResponse(r *http.Request) error {
	if r.Method != "POST" {
		return errors.New(fmt.Sprintf("Unexpected Method found: %s", r.Method))
	}
	if r.RequestURI != "/api/v1/ingest/events" {
		return errors.New(fmt.Sprintf("Unexpected RequestURI: %s", r.RequestURI))
	}
	expectedHeaders := map[string]string{
		"Authorization":    mockAPIKey,
		"Content-Encoding": "gzip",
		"Content-Type":     "application/x-ndjson",
		"Accept-Encoding":  "gzip",
	}
	if err := checkHeaders(r.Header, expectedHeaders); err != nil {
		return err
	}
	// Pass the body through the Gzip reader
	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		return err
	}

	// Scan for newline-delimited content
	s := bufio.NewScanner(zr)
	for s.Scan() {
		// Decode back into struct
		var event spmta.SparkPostEvent
		if err := json.Unmarshal(s.Bytes(), &event); err != nil {
			return err
		}
		// Check contents
		e := event.EventWrapper.EventGrouping
		if e.Type != "click" || e.DelvMethod != "esmtp" || len(e.EventID) < 10 {
			return errors.New(fmt.Sprintf("Event has a suspect value somewhere in a) %v %v %v", e.Type, e.DelvMethod, e.EventID))
		}
		if e.IPAddress != testIPAddress || e.MessageID != testMessageID || e.RcptTo != testRecipient {
			return errors.New(fmt.Sprintf("Event has a suspect value somewhere in b) %v %v %v", e.IPAddress, e.MessageID, e.RcptTo))
		}
		if len(e.TimeStamp) < 10 || e.TargetLinkURL != testURL || e.UserAgent != testUserAgent {
			return errors.New(fmt.Sprintf("Event has a suspect value somewhere in c) %v %v %v", e.TimeStamp, e.TargetLinkURL, e.UserAgent))
		}
		if e.SubaccountID != testSubaccountID {
			return errors.New(fmt.Sprintf("Event has a suspect value somewhere in d) %v", e.SubaccountID))
		}
	}
	return nil
}

// Mock SparkPost endpoint
func ingestServer(w http.ResponseWriter, r *http.Request) {
	s := checkResponse(r)
	if s != nil {
		w.WriteHeader(http.StatusBadRequest)
		eJSON := fmt.Sprintf(`{"errors": [ {"message": "%s"} ]}`, s.Error())
		w.Write([]byte(eJSON))
		return
	}
	// Success
	eJSON := `{"results": {"id": "mock test passed"} }`
	w.Write([]byte(eJSON))
}

// Run the mock SparkPost endpoint
func startMockIngest(t *testing.T, addrPort string) {
	http.HandleFunc("/", ingestServer)
	server := &http.Server{
		Addr: addrPort,
	}
	err := server.ListenAndServe()
	if err != nil {
		t.Errorf("Error starting mock SparkPost ingest endpoint %v", err)
	}
}

// Test the feeder function by creating a mockup endpoint that will receive data that we push to it
func TestFeedForever(t *testing.T) {
	p := rand.Intn(1000) + 8000
	mockIngestAddrPort := ":" + strconv.Itoa(p)

	// Start the mock SparkPost endpoint server concurrently
	go startMockIngest(t, mockIngestAddrPort)
	client := spmta.MyRedis()

	// Make sure redis queue is empty
	for {
		_, err := client.LPop(spmta.RedisQueue).Result()
		if err == redis.Nil {
			break
		}
	}

	// Start the feeder process concurrently. We don't have to wait the usual time
	go spmta.FeedForever(client, "http://"+mockIngestAddrPort, mockAPIKey, testTime)

	fmt.Println("One event")
	myLogp := captureLog()
	mockEvents(t, 1, client)
	checkLog(t, 20, myLogp, "SparkPost Ingest response: 200 OK, results.id=mock test passed")

	fmt.Println("Many events")
	myLogp = captureLog()
	mockEvents(t, 12000, client)
	checkLog(t, 20, myLogp, "SparkPost Ingest response: 200 OK, results.id=mock test passed")
}

func checkLog(t *testing.T, waitfor int, myLogp *bytes.Buffer, expected string) {
	// Wait for processes to log something
	res := ""
	count := 0
	for res == "" {
		time.Sleep(1 * time.Second)
		count++
		if count > waitfor {
			t.Error(fmt.Sprintf("Waited %v seconds and no response - exiting", waitfor))
			break
		}
		res = retrieveLog(myLogp)
	}
	time.Sleep(testTime * 2)
	res = retrieveLog(myLogp)
	fmt.Println(res)
	if !strings.Contains(res, expected) {
		t.Error(res)
	}
}

func mockEvents(t *testing.T, nEvents int, client *redis.Client) {
	// Build a test event
	msgID := testMessageID
	var e spmta.TrackEvent
	e.WD = spmta.WrapperData{
		Action:        testAction,
		TargetLinkURL: testURL,
		MessageID:     msgID,
	}
	e.UserAgent = testUserAgent
	e.IPAddress = testIPAddress
	e.TimeStamp = strconv.FormatInt(time.Now().Unix(), 10)

	// Build the composite info ready to push into the Redis queue
	eBytes, err := json.Marshal(e)
	if err != nil {
		t.Errorf("Error %v", err)
	}

	// Create a fake acct_etl enrichment record in Redis
	enrichment := map[string]string{
		"header_x-sp-subaccount-id": strconv.Itoa(testSubaccountID),
		"rcpt":                      testRecipient,
	}
	enrichmentJSON, err := json.Marshal(enrichment)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	msgIDKey := spmta.TrackingPrefix + msgID
	ttl := time.Duration(testTime * 20) // expires fairly quickly after test run
	_, err = client.Set(msgIDKey, enrichmentJSON, ttl).Result()
	if err != nil {
		t.Errorf("Error %v", err)
	}

	for i := 0; i < nEvents; i++ {
		if _, err = client.RPush(spmta.RedisQueue, eBytes).Result(); err != nil {
			t.Errorf("Error %v", err)
		}
	}
}
