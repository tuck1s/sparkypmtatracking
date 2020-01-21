package sparkypmtatracking_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-redis/redis"
	spmta "github.com/tuck1s/sparkyPMTATracking"
)

// runHTTPTest wrapper convenience function
func runHTTPTest(t *testing.T, method string, reqURL string, expectCode int, expectBody []byte, client *redis.Client) {
	emptyRedisQueue(client)

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		t.Error(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(spmta.TrackingServer)
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != expectCode {
		t.Errorf("handler returned wrong status code: got %v want %v", status, expectCode)
	}

	// Check the response body is what we expect.
	gotBody, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(gotBody, expectBody) != 0 {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectBody)
	}

	if len(expectBody) > 0 {
		// Check the Redis queue entry is what we expect.
		d, err := client.LPop(spmta.RedisQueue).Result()
		if err != nil {
			t.Error(err)
		}
		fmt.Println(d)
	}
}

// Make pseudo http requests in, check Redis queue contents comes out
func TestTrackingServer(t *testing.T) {
	var empty []byte
	client := spmta.MyRedis()

	// basic sniff test with a short path
	runHTTPTest(t, "GET", "/", http.StatusBadRequest, empty, client)

	// click
	url, err := spmta.EncodeLink(testTrackingDomain, "click", testMessageID, testRecipient, testTargetURL)
	if err != nil {
		t.Error(err)
	}
	runHTTPTest(t, "GET", url, http.StatusFound, empty, client)

	// open
	url, err = spmta.EncodeLink(testTrackingDomain, "open", testMessageID, testRecipient, testTargetURL)
	if err != nil {
		t.Error(err)
	}
	runHTTPTest(t, "GET", url, http.StatusOK, spmta.TransparentGif, client)

	// initial_open
	url, err = spmta.EncodeLink(testTrackingDomain, "initial_open", testMessageID, testRecipient, testTargetURL)
	if err != nil {
		t.Error(err)
	}
	runHTTPTest(t, "GET", url, http.StatusOK, spmta.TransparentGif, client)

	// Other (invalid) method verbs - see https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods
	runHTTPTest(t, "HEAD", "/", http.StatusMethodNotAllowed, empty, client)
	runHTTPTest(t, "POST", "/", http.StatusMethodNotAllowed, empty, client)
	runHTTPTest(t, "PUT", "/", http.StatusMethodNotAllowed, empty, client)
	runHTTPTest(t, "DELETE", "/", http.StatusMethodNotAllowed, empty, client)
	runHTTPTest(t, "CONNECT", "/", http.StatusMethodNotAllowed, empty, client)
	runHTTPTest(t, "OPTIONS", "/", http.StatusMethodNotAllowed, empty, client)
	runHTTPTest(t, "TRACE", "/", http.StatusMethodNotAllowed, empty, client)
	runHTTPTest(t, "PATCH", "/", http.StatusMethodNotAllowed, empty, client)
}
func TestTrackingServerFaultyInputs(t *testing.T) {
	var empty []byte
	client := spmta.MyRedis()
	client.Del(spmta.RedisQueue)

	// Invalid path (will fail base64 decoding)
	runHTTPTest(t, "GET", "/~~~~~~", http.StatusBadRequest, empty, client)

	// Invalid path (will fail zlib decoding)
	runHTTPTest(t, "GET", "/not_a_valid_path", http.StatusBadRequest, empty, client)

	// Invalid path (will fail JSON decoding)
	truncPath := []byte(`{"act":"c","t_url":"xyzzy"`)
	tpEnc, err := spmta.EncodePath(truncPath)
	if err != nil {
		t.Error(err)
	}
	runHTTPTest(t, "GET", "/"+tpEnc, http.StatusBadRequest, empty, client)

	// force Redis RPush to fail
	client.Del(spmta.RedisQueue)
	client.Set(spmta.RedisQueue, "not a queue", 0)
	// click
	url, err := spmta.EncodeLink(testTrackingDomain, "click", testMessageID, testRecipient, testTargetURL)
	if err != nil {
		t.Error(err)
	}
	runHTTPTest(t, "GET", url, http.StatusInternalServerError, empty, client)
	// clean up after
	client.Del(spmta.RedisQueue)
}
