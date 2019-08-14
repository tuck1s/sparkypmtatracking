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
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

// Make a fake GeoIP
func fakeGeoIP() GeoIP {
	return GeoIP{
		Country:    "US",
		Region:     "MD",
		City:       "Columbia",
		Latitude:   39.1749,
		Longitude:  -76.8375,
		Zip:        21046,
		PostalCode: "21046",
	}
}

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

// For efficiency under load, collect n events into a batch
const ingestBatchSize = 1000
const ingestMaxWait = 10 * time.Second

func makeSparkPostEvent(eStr string, client *redis.Client) SparkPostEvent {
	var tev TrackEvent
	err := json.Unmarshal([]byte(eStr), &tev)
	Check(err)
	var spEvent SparkPostEvent
	// Shortcut pointer to the attribute-carrying leaf object; fill in received attributes
	eptr := &spEvent.EventWrapper.EventGrouping
	eptr.Type = tev.Type
	eptr.TargetLinkUrl = tev.TargetLinkUrl
	eptr.MessageID = tev.MessageID
	eptr.TimeStamp = tev.TimeStamp
	eptr.UserAgent = tev.UserAgent
	eptr.IPAddress = tev.IPAddress

	// Enrich with PowerMTA accounting-pipe values, if we have these, from persistent storage
	tKey := TrackingPrefix + tev.MessageID
	enrichmentJSON, err := client.Get(tKey).Result()
	if err == redis.Nil {
		Console_and_log_fatal("Error: redis key", tKey, "not found")
	}
	enrichment := make(map[string]string)
	err = json.Unmarshal([]byte(enrichmentJSON), &enrichment)
	Check(err)
	eptr.MsgFrom = enrichment["orig"]
	eptr.RcptTo = enrichment["rcpt"]
	eptr.CampaignID = enrichment["jobId"]
	eptr.SendingIP = enrichment["dlvSourceIp"]
	eptr.IPPool = enrichment["vmtaPool"]

	// Fill in these fields with default / unique / derived values
	eptr.DelvMethod = "esmtp"
	eptr.EventID = uniqEventId()
	eptr.InitialPixel = false
	eptr.ClickTracking = true
	eptr.OpenTracking = true
	eptr.RoutingDomain = strings.Split(eptr.RcptTo, "@")[1]

	// Skip these fields for now; you may have information to populate them from your own sources
	//eptr.GeoIP
	//eptr.FriendlyFrom
	//eptr.RcptTags
	//eptr.RcptMeta
	//eptr.Subject			.. SparkPost doesn't fill this in on Open & Click events
	return spEvent
}

// Prepare a batch of TrackingEvents for ingest to SparkPost.
func sparkPostIngest(batch []string, client *redis.Client, host string, apiKey string) {
	var ingestData bytes.Buffer
	ingestData.Grow(1024 * len(batch)) // Preallocate approx size of the string for efficiency
	for _, eStr := range batch {
		e := makeSparkPostEvent(eStr, client)
		eJSON, err := json.Marshal(e)
		Check(err)
		ingestData.Write(eJSON)
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
	gzipSize := zbuf.Len()

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
	log.Println("Uploaded", len(batch), "events", gzipSize, "bytes (gzip), SparkPost Ingest response:", res.Status, "results.id=", resObj.Results.Id)
	err = res.Body.Close()
	Check(err)
}

func main() {
	// Use logging, as this program will be executed without an attached console
	MyLogger("feeder.log")
	client := MyRedis()
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
