package sparkypmtatracking_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-redis/redis"
	spmta "github.com/tuck1s/sparkypmtatracking"
)

// runHTTPTest wrapper convenience function
func runHTTPTest(t *testing.T, method string, reqURL string, expectCode int, expectBody []byte, client *redis.Client, realIPHeader string) {
	emptyRedisQueue(client)

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		t.Error(err)
	}
	if realIPHeader != "" {
		req.Header.Set(spmta.XRealIPHeader, realIPHeader) // example value
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
		_, err := client.LPop(spmta.RedisQueue).Result()
		if err != nil {
			t.Error(err)
		}
	}
}

// Make pseudo http requests in, check Redis queue contents comes out
func TestTrackingServer(t *testing.T) {
	var empty []byte
	client := spmta.MyRedis()

	// basic sniff test with a short path
	runHTTPTest(t, "GET", "/", http.StatusBadRequest, empty, client, "1.2.3.4")

	// Check responses
	type actionExpectedResponse struct {
		Act                  string
		resp                 int
		body                 []byte
		trackOpen            bool
		trackInitialOpen     bool
		trackLink            bool
		includeXRealIPHeader string
	}
	arList := []actionExpectedResponse{
		{"click", http.StatusFound, empty, true, true, true, ""},
		{"open", http.StatusOK, spmta.TransparentGif, true, true, true, ""},
		{"initial_open", http.StatusOK, spmta.TransparentGif, true, true, true, ""},

		{"click", http.StatusFound, empty, true, true, true, "1.2.3.4"},
		{"open", http.StatusOK, spmta.TransparentGif, true, true, true, "1.2.3.4"},
		{"initial_open", http.StatusOK, spmta.TransparentGif, true, true, true, "1.2.3.4"},
	}
	for _, ar := range arList {
		trkDomain := RandomBaseURL()
		msgID := spmta.UniqMessageID()
		recip := RandomRecipient()
		url, err := spmta.EncodeLink(trkDomain, ar.Act, msgID, recip, RandomURLWithPath(), ar.trackOpen, ar.trackInitialOpen, ar.trackLink)
		if err != nil {
			t.Error(err)
		}
		runHTTPTest(t, "GET", url, ar.resp, ar.body, client, ar.includeXRealIPHeader)
	}
	// Other (invalid) method verbs - see https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods
	for _, verb := range []string{"HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"} {
		runHTTPTest(t, verb, "/", http.StatusMethodNotAllowed, empty, client, "")
	}
}

func TestTrackingServerFaultyInputs(t *testing.T) {
	var empty []byte
	client := spmta.MyRedis()
	client.Del(spmta.RedisQueue)

	// Invalid path (will fail base64 decoding)
	runHTTPTest(t, "GET", "/~~~~~~", http.StatusBadRequest, empty, client, "")

	// Invalid path (will fail zlib decoding)
	runHTTPTest(t, "GET", "/not_a_valid_path", http.StatusBadRequest, empty, client, "")

	// Invalid path (will fail JSON decoding)
	truncPath := []byte(`{"act":"c","t_url":"xyzzy"`)
	tpEnc, err := spmta.EncodePath(truncPath)
	if err != nil {
		t.Error(err)
	}
	runHTTPTest(t, "GET", "/"+tpEnc, http.StatusBadRequest, empty, client, "")

	trkDomain := RandomBaseURL()
	msgID := spmta.UniqMessageID()
	recip := RandomRecipient()
	url, err := spmta.EncodeLink(trkDomain, "click", msgID, recip, RandomURLWithPath(), true, true, true)
	if err != nil {
		t.Error(err)
	}

	// Invalid X-Real-Ip header should be detected
	runHTTPTest(t, "GET", url, http.StatusBadRequest, empty, client, "banana")

	// force Redis RPush to fail
	client.Del(spmta.RedisQueue)
	client.Set(spmta.RedisQueue, "not a queue", 0)
	runHTTPTest(t, "GET", url, http.StatusInternalServerError, empty, client, "")

	// clean up after
	client.Del(spmta.RedisQueue)
}
