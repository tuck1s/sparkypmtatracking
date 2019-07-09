package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	. "github.com/sparkyPmtaTracking/src/common"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

// Make a SparkPost formatted unique event_id, which needs to be a decimal string 0 .. (2^63-1)
func uniqEventId() string {
	u := uuid.New()
	num := binary.LittleEndian.Uint64(u[:8]) & 0x7fffffffffffffff
	return strconv.FormatUint(num, 10)
}

// Make a SparkPost formatted unique message_id, which needs to be a hex string
func uniqMessageId() string {
	u := uuid.New()
	h := hex.EncodeToString(u[:12])
	return strings.ToUpper(h)
}

// Map our own tracking ID to the SparkPost Message ID via Redis - if found
func trackingIDToMessageID(eStr string, client *redis.Client) string {
	var t TrackEvent
	err := json.Unmarshal([]byte(eStr), &t)
	Check(err)
	//t.TrackingID = "a5a1050437e345389bc2c7d6d79743f5"			//TODO: debug
	trkID := "trk_" + t.TrackingID
	msgID, err := client.Get(trkID).Result()
	if err == redis.Nil {
		log.Println("Tracking ID", trkID, "not found - using a generated unique message_id")
		msgID = uniqMessageId()
	}
	return msgID
}

// For efficiency under high load conditions, collect n events into a batch
const ingestBatchSize = 10000
const ingestMaxWait = 10 * time.Second

// Prepare a batch of TrackingEvents for ingest to SparkPost.
// Because the JSON declarations coincide, we can unmarshal a TrackingEvent into the leaf part of a SparkPostEvent
func sparkPostIngest(batch []string, client *redis.Client, host string, apiKey string) {
	var ingestData bytes.Buffer
	ingestData.Grow(1024 * len(batch)) // Preallocate approx size of the string for efficiency
	for _, eStr := range batch {
		var e SparkPostEvent
		eptr := &e.EventWrapper.EventGrouping
		err := json.Unmarshal([]byte(eStr), eptr)
		Check(err)
		// Fill in some fields with default values
		eptr.DelvMethod = "esmtp"
		eptr.GeoIP = GeoIP{
			Country:    "US",
			Region:     "MD",
			City:       "Columbia",
			Latitude:   39.1749,
			Longitude:  -76.8375,
			Zip:        21046,
			PostalCode: "21046",
		}
		eptr.EventID = uniqEventId()
		eptr.InitialPixel = false
		eptr.MessageID = trackingIDToMessageID(eStr, client)
		eptr.RoutingDomain = strings.Split(eptr.RcptTo, "@")[1]
		eptr.ClickTracking = true
		eptr.OpenTracking = true
		s, err := json.Marshal(e)
		Check(err)
		ingestData.Write(s)
		ingestData.WriteString("\n")
	}
	// Now have ingestData in buffer - flow through gzip writer
	var zbuf bytes.Buffer
	ir := bufio.NewReader(&ingestData)
	zw := gzip.NewWriter(&zbuf)
	_, err := io.Copy(zw, ir)
	Check(err)
	err = zw.Close() // ensure all data written (seems to be necessary)
	Check(err)

	// Prepare the https POST request
	zr := bufio.NewReader(&zbuf)
	var netClient = &http.Client{
		Timeout: time.Second * 300,
	}
	url := host + "/api/v1/ingest/events"
	req, err := http.NewRequest("POST", url, zr)
	Check(err)
	req.Header = map[string][]string{
		"Authorization":    {apiKey},
		"Content-Type":     {"application/x-ndjson"},
		"Content-Encoding": {"gzip"},
	}
	res, err := netClient.Do(req)
	Check(err)

	var resObj IngestResult
	respRd := json.NewDecoder(res.Body)
	err = respRd.Decode(&resObj)
	Check(err)
	log.Println("SparkPost Ingest response:", res.Status, "results.id=", resObj.Results.Id)
	err = res.Body.Close()
	Check(err)
}

func main() {
	// Use logging, as this program will be executed without an attached console
	logfile, err := os.OpenFile("feeder.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	Check(err)
	log.SetOutput(logfile)

	// Prepare to pop records from Redis. Assume server is on the standard port
	client := redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Get SparkPost ingest credentials from env vars
	host := HostCleanup(GetenvDefault("SPARKPOST_HOST_INGEST", "api.sparkpost.com"))
	apiKey := GetenvDefault("SPARKPOST_API_KEY_INGEST", "")
	if apiKey == "" {
		Console_and_log_fatal("SPARKPOST_API_KEY_INGEST not set - stopping")
	}

	// Process forever data arriving via Redis queue
	trackingData := make([]string, 0, ingestBatchSize) // Pre-allocate for efficiency
	for {
		d, err := client.LPop(RedisQueue).Result()
		if err == redis.Nil {
			// special value means queue is empty. Ingest any data we have collected, then wait a while
			if len(trackingData) > 0 {
				sparkPostIngest(trackingData, client, host, apiKey)
				trackingData = trackingData[:0] // empty the data, but keep capacity allocated
			}
			time.Sleep(ingestMaxWait)
		} else {
			if err != nil {
				log.Println(err)
			} else {
				// stash data away for later. If we have a full batch, SparkPost sparkPostIngest it
				trackingData = append(trackingData, d)
				if len(trackingData) >= ingestBatchSize {
					sparkPostIngest(trackingData, client, host, apiKey)
					trackingData = trackingData[:0] // empty the data, but keep capacity allocated
				}
			}
		}
	}
}
