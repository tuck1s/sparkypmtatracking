package sparkypmtatracking_test

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

const mockAPIKey = "xyzzy"

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
	if s := checkHeaders(r.Header, expectedHeaders); s != nil {
		return s
	}

	// Pass the body through the Gzip reader
	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\nComment: %s\nModTime: %s\n\n", zr.Name, zr.Comment, zr.ModTime.UTC())

	if _, err := io.Copy(os.Stdout, zr); err != nil {
		log.Fatal(err)
	}

	if err := zr.Close(); err != nil {
		log.Fatal(err)
	}

	return nil
}

// Mock SparkPost endpoint
func ingestServer(w http.ResponseWriter, r *http.Request) {
	s := checkResponse(r)
	fmt.Println(s)
	//TODO: handle error return values
}

// Run the mock SparkPost endpoint
func startMockIngest(t *testing.T, addrPort string) {
	http.HandleFunc("/", ingestServer) // Accept subtree matches
	server := &http.Server{
		Addr: addrPort,
	}
	err := server.ListenAndServe()
	if err != nil {
		t.Errorf("Error starting mock SparkPost ingest endpoint %v", err)
	}
}

const testAction = "c" // click
const testURL = "http://example.com/this/is/a/test"
const testUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko)"
const testIPAddress = "12.34.56.78"

const testTime = 5 * time.Second

// Test the feeder function by creating a mockup endpoint that will receive data that we push to it
func TestFeedForever(t *testing.T) {
	rand.Seed(42)
	p := rand.Intn(1000) + 8000
	mockIngestAddrPort := ":" + strconv.Itoa(p)

	// Start the mock SparkPost endpoint server concurrently
	go startMockIngest(t, mockIngestAddrPort)
	client := spmta.MyRedis()

	// Start the feeder process concurrently. We don't have to wait the usual time
	go spmta.FeedForever(client, "http://"+mockIngestAddrPort, mockAPIKey, testTime)

	// Build a test event
	msgID := spmta.UniqMessageID()
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

	if _, err = client.RPush(spmta.RedisQueue, eBytes).Result(); err != nil {
		t.Errorf("Error %v", err)
	}

	// Wait for processes to run
	time.Sleep(2 * testTime)

	// Check if results happened. Our goroutines will keep running :shrug:
	fmt.Println("Done")
}
