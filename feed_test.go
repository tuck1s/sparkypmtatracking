package sparkypmtatracking_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
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
const testUserAgent = "Some Lovely Browser User Agent String"
const testIPAddress = "12.34.56.78"
const testTime = 1 * time.Second // delay we have to wait for batches to show up in the logfile
const testSubaccountID = 3
const testMockBatchResponse = "mock test passed"

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

// Wait for processes to log something for a defined number of seconds, look for a number of occurrences of an expected string
func checkLog(t *testing.T, waitfor int, myLogp *bytes.Buffer, expected string, times int) {
	res := ""
	count := 0
	for res == "" {
		time.Sleep(1 * time.Second)
		count++
		if count > waitfor {
			t.Errorf("Waited %v seconds and no response - exiting", waitfor)
			break
		}
		res = retrieveLog(myLogp)
	}
	time.Sleep(testTime)
	res = retrieveLog(myLogp)
	t.Log(res)
	if strings.Count(res, expected) != times {
		t.Error(res)
	}
}

// checkHeaders walks through the headers and checks against expected values
func checkHeaders(h http.Header, chk map[string]string) error {
	for k, v := range chk {
		if len(h[k]) != 1 {
			s := fmt.Errorf("Header %s has length %d", k, len(h[k]))
			return s
		}
		// Case-insensitive comparison
		if strings.ToLower(h[k][0]) != strings.ToLower(v) {
			s := fmt.Errorf("Header %s has value %s, was expecting value %s", k, h[k][0], v)
			return s
		}
	}
	return nil
}

// checkResponse compares the mock SparkPost request with what was expected
func checkResponse(r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("Unexpected Method found: %s", r.Method)
	}
	if r.RequestURI != "/api/v1/ingest/events" {
		return fmt.Errorf("Unexpected RequestURI: %s", r.RequestURI)
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
		ln := s.Bytes()
		// Decode back into struct
		var event spmta.SparkPostEvent
		if err := json.Unmarshal(ln, &event); err != nil {
			return err
		}
		// Check contents - allow for some missing fields during tests
		e := event.EventWrapper.EventGrouping
		if e.IPAddress != testIPAddress {
			return fmt.Errorf("Event has a suspect IPAddress %v", e.IPAddress)
		}
		if len(e.TimeStamp) < 10 || e.UserAgent != testUserAgent {
			return fmt.Errorf("Event has a suspect UserAgent %v", e.UserAgent)
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
	eJSON := fmt.Sprintf(`{"results": {"id": "%s"} }`, testMockBatchResponse)
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
	// Start the feeder process concurrently. We don't have to wait the usual time
	go spmta.FeedForever(client, "http://"+mockIngestAddrPort, mockAPIKey, testTime)

	t.Log("One event")
	myLogp := captureLog()
	emptyRedisQueue(client)
	mockEvents(t, 1, client, true)
	checkLog(t, 10, myLogp, testMockBatchResponse, 1)

	t.Log("Many events")
	myLogp = captureLog()
	emptyRedisQueue(client)
	mockEvents(t, 12000, client, true)
	checkLog(t, 10, myLogp, testMockBatchResponse, 2) // two batches

	t.Log("One event with no message_id in redis")
	myLogp = captureLog()
	emptyRedisQueue(client)
	mockEvents(t, 1, client, false)
	checkLog(t, 10, myLogp, testMockBatchResponse, 1)
}

func wrongTypeErr(err error) bool {
	s := err.Error()
	return len(s) >= 9 && s[0:9] == "WRONGTYPE"
}

func emptyRedisQueue(client *redis.Client) {
	// Make sure redis queue is empty
	for {
		_, err := client.LPop(spmta.RedisQueue).Result()
		if err == nil {
			continue // actual value read .. keep going
		}
		// Check for empty, or queue error
		if err == redis.Nil || wrongTypeErr(err) {
			break
		}
	}
}

const ttl = time.Duration(testTime * 20) // expires fairly quickly after test run

func mockEvents(t *testing.T, nEvents int, client *redis.Client, augment bool) {
	for i := 0; i < nEvents; i++ {
		msgID := spmta.UniqMessageID()
		recip := RandomRecipient()
		e := testEvent(msgID)
		// Build the composite info ready to push into the Redis queue
		eBytes, err := json.Marshal(e)
		if err != nil {
			t.Errorf("Error %v", err)
		}
		if augment {
			// Create a fake acct_etl augmentation record in Redis
			augmentData := map[string]string{
				"header_x-sp-subaccount-id": strconv.Itoa(testSubaccountID),
				"rcpt":                      recip,
			}
			augmentJSON, err := json.Marshal(augmentData)
			if err != nil {
				t.Errorf("Error %v", err)
			}
			msgIDKey := spmta.TrackingPrefix + msgID
			_, err = client.Set(msgIDKey, augmentJSON, ttl).Result()
			if err != nil {
				t.Errorf("Error %v", err)
			}
		}
		if _, err = client.RPush(spmta.RedisQueue, eBytes).Result(); err != nil {
			t.Errorf("Error %v", err)
		}
	}
}

// Build a test event
func testEvent(msgID string) spmta.TrackEvent {
	var e spmta.TrackEvent
	e.WD = spmta.WrapperData{
		Action:        testAction,
		TargetLinkURL: testURL,
		MessageID:     msgID,
	}
	e.UserAgent = testUserAgent
	e.IPAddress = testIPAddress
	e.TimeStamp = strconv.FormatInt(time.Now().Unix(), 10)
	return e
}

// Detailed unit tests

func TestFeedEventsErrorCases(t *testing.T) {
	client := spmta.MyRedis()
	client.Close() // deliberately close the connection before using
	err := spmta.FeedEvents(client, "http://example.com", "", testTime)
	if err.Error() != "redis: client is closed" {
		t.Errorf("Error %v", err)
	}
}

func TestAgedContent(t *testing.T) {
	// no content, not aged
	tBuf := spmta.TimedBuffer{
		Content:     []byte(""),
		TimeStarted: time.Now(),
		MaxAge:      10 * time.Second,
	}
	res := tBuf.AgedContent()
	if res {
		t.Errorf("Unexpected value")
	}

	// content, not aged
	tBuf = spmta.TimedBuffer{
		Content:     []byte("Some content"),
		TimeStarted: time.Now(),
		MaxAge:      10 * time.Second,
	}
	res = tBuf.AgedContent()
	if res {
		t.Errorf("Unexpected value")
	}

	// no content, aged
	tBuf = spmta.TimedBuffer{
		Content: []byte(""),
		// have to add a negative duration, see https://golang.org/pkg/time/#example_Time_Sub
		TimeStarted: time.Now().Add(-30 * time.Second),
		MaxAge:      10 * time.Second,
	}
	res = tBuf.AgedContent()
	if res {
		t.Errorf("Unexpected value")
	}

	// content, aged
	tBuf = spmta.TimedBuffer{
		Content:     []byte("Some content"),
		TimeStarted: time.Now().Add(-2 * time.Second),
		MaxAge:      1 * time.Second,
	}
	res = tBuf.AgedContent()
	if !res {
		t.Errorf("Unexpected value")
	}
}

func TestSparkPostEventNDJSONFaultyInputs(t *testing.T) {
	client := spmta.MyRedis()
	// Invalid input string, i.e. not properly constructed JSON
	const eStrFaulty = `{"WD":{"act":"c`
	_, err := spmta.SparkPostEventNDJSON(eStrFaulty, client)
	if !strings.Contains(err.Error(), "end of JSON") {
		t.Error(err)
	}

	// Message ID that can't be found, so event will succeed, not be augmented, and warning logged
	const notFoundMsgID = "0000123456789abcdef1"
	const redisKeyNotFoundMsgID = "msgID_" + notFoundMsgID
	client.Del(redisKeyNotFoundMsgID)
	myLogp := captureLog()
	e := testEvent(notFoundMsgID)
	eBytes, err := json.Marshal(e)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	_, err = spmta.SparkPostEventNDJSON(string(eBytes), client)
	if err != nil {
		t.Error(err)
	}
	checkLog(t, 1, myLogp, "Warning: redis key "+redisKeyNotFoundMsgID+" not found", 1)

	// MessageID Redis record corrupt
	const augmentFaulty = `{"header_x-sp-subaccount-id"`
	const augmentFaultyMsgID = "0000123456789abcdef2"
	e = testEvent(augmentFaultyMsgID)
	eBytes, err = json.Marshal(e)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	client.Set("msgID_"+augmentFaultyMsgID, augmentFaulty, ttl)
	_, err = spmta.SparkPostEventNDJSON(string(eBytes), client)
	if !strings.Contains(err.Error(), "end of JSON") {
		t.Error(err)
	}
}

func TestSparkPostIngestFaultyInputs(t *testing.T) {
	client := spmta.MyRedis()
	var ingestData []byte // empty

	// provoke an error in the host address
	host := "http://api.sparkpost.com/not_an_api"
	apiKey := "junk"
	err := spmta.SparkPostIngest(ingestData, client, host, apiKey)
	if !strings.Contains(err.Error(), "Only https is allowed") {
		t.Error(err)
	}

	// provoke an error in JSON response
	host = "https://example.com/"
	err = spmta.SparkPostIngest(ingestData, client, host, apiKey)
	if !strings.Contains(err.Error(), "invalid character") {
		t.Error(err)
	}
}

func TestFeedEventsFaultyInputs(t *testing.T) {
	client := spmta.MyRedis()
	// Invalid input string, i.e. not properly constructed JSON, pushed into Redis queue
	eBytesFaulty := []byte(`{"WD":{"act":"c`)
	if _, err := client.RPush(spmta.RedisQueue, eBytesFaulty).Result(); err != nil {
		t.Errorf("Error %v", err)
	}
	host := "http://api.sparkpost.com/not_an_api"
	apiKey := "junk"
	err := spmta.FeedEvents(client, host, apiKey, testTime)
	if !strings.Contains(err.Error(), "end of JSON") {
		t.Error(err)
	}
}
