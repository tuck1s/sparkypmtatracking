package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	spmta "github.com/tuck1s/sparkyPMTATracking"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

// Make a SparkPost formatted unique event_id, which needs to be a decimal string 0 .. (2^63-1)
func uniqEventID() string {
	u := uuid.New()
	num := binary.LittleEndian.Uint64(u[:8]) & 0x7fffffffffffffff
	return strconv.FormatUint(num, 10)
}

// Make a SparkPost formatted unique message_id, which needs to be a hex string
func uniqMessageID() string {
	u := uuid.New()
	h := hex.EncodeToString(u[:12])
	return strings.ToUpper(h)
}

// For efficiency under load, collect n events into a batch
const ingestBatchSize = 1000
const ingestMaxWait = 10 * time.Second

func makeSparkPostEvent(eStr string, client *redis.Client) (spmta.SparkPostEvent, error) {
	var tev spmta.TrackEvent
	var spEvent spmta.SparkPostEvent
	if err := json.Unmarshal([]byte(eStr), &tev); err != nil {
		return spEvent, err
	}
	// Shortcut pointer to the attribute-carrying leaf object; fill in received attributes
	eptr := &spEvent.EventWrapper.EventGrouping
	eptr.Type = spmta.ActionToType(tev.WD.Action)
	eptr.TargetLinkURL = tev.WD.TargetLinkURL
	eptr.MessageID = tev.WD.MessageID
	eptr.TimeStamp = tev.TimeStamp
	eptr.UserAgent = tev.UserAgent
	eptr.IPAddress = tev.IPAddress

	// Enrich with PowerMTA accounting-pipe values, if we have these, from persistent storage
	tKey := spmta.TrackingPrefix + tev.WD.MessageID
	if enrichmentJSON, err := client.Get(tKey).Result(); err == redis.Nil {
		log.Println("Warning: redis key", tKey, "not found, url=", tev.WD.TargetLinkURL)
	} else {
		enrichment := make(map[string]string)
		err = json.Unmarshal([]byte(enrichmentJSON), &enrichment)
		if err != nil {
			log.Println(err.Error())
		}
		eptr.MsgFrom = enrichment["orig"]
		eptr.RcptTo = enrichment["rcpt"]
		eptr.CampaignID = enrichment["jobId"]
		eptr.SendingIP = enrichment["dlvSourceIp"]
		eptr.IPPool = enrichment["vmtaPool"]
		eptr.RoutingDomain = strings.Split(eptr.RcptTo, "@")[1]
	}

	// Fill in these fields with default / unique / derived values
	eptr.DelvMethod = "esmtp"
	eptr.EventID = uniqEventID()
	eptr.InitialPixel = true // These settings reflect capability of wrapper function
	eptr.ClickTracking = true
	eptr.OpenTracking = true

	// Skip these fields for now; you may have information to populate them from your own sources
	//eptr.GeoIP, eptr.FriendlyFrom, eptr.RcptTags, eptr.RcptMeta
	//eptr.Subject			.. SparkPost doesn't fill this in on Open & Click events
	return spEvent, nil
}

// Prepare a batch of TrackingEvents for ingest to SparkPost.
func sparkPostIngest(batch []string, client *redis.Client, host string, apiKey string) {
	var ingestData bytes.Buffer
	ingestData.Grow(1024 * len(batch)) // Preallocate approx size of the string for efficiency
	for _, eStr := range batch {
		e, err := makeSparkPostEvent(eStr, client)
		if err != nil {
			log.Println(err.Error())
			continue // with next event, if we can
		}
		eJSON, err := json.Marshal(e)
		if err != nil {
			log.Println(err.Error())
			continue // with next event, if we can
		}
		ingestData.Write(eJSON)
		ingestData.WriteString("\n")
	}

	// Now have ingestData in buffer - flow through gzip writer
	var zbuf bytes.Buffer
	ir := bufio.NewReader(&ingestData)
	zw := gzip.NewWriter(&zbuf)
	_, err := io.Copy(zw, ir)
	spmta.Check(err)
	err = zw.Close() // ensure all data written (seems to be necessary)
	spmta.Check(err)
	gzipSize := zbuf.Len()

	// Prepare the https POST request
	zr := bufio.NewReader(&zbuf)
	var netClient = &http.Client{
		Timeout: time.Second * 300,
	}
	url := host + "/api/v1/ingest/events"
	req, err := http.NewRequest("POST", url, zr)
	spmta.Check(err)
	req.Header = map[string][]string{
		"Authorization":    {apiKey},
		"Content-Type":     {"application/x-ndjson"},
		"Content-Encoding": {"gzip"},
	}
	res, err := netClient.Do(req)
	spmta.Check(err)

	var resObj spmta.IngestResult
	respRd := json.NewDecoder(res.Body)
	err = respRd.Decode(&resObj)
	spmta.Check(err)
	log.Println("Uploaded", len(batch), "events", gzipSize, "bytes (gzip), SparkPost Ingest response:", res.Status, "results.id=", resObj.Results.ID)
	err = res.Body.Close()
	spmta.Check(err)
}

func main() {
	logfile := flag.String("logfile", "", "File written with message logs (also to stdout)")
	flag.Parse()
	spmta.MyLogger(*logfile)

	// Get SparkPost ingest info from env vars
	host := spmta.HostCleanup(spmta.GetenvDefault("SPARKPOST_HOST_INGEST", "api.sparkpost.com"))
	apiKey := spmta.GetenvDefault("SPARKPOST_API_KEY_INGEST", "")
	if apiKey == "" {
		spmta.ConsoleAndLogFatal("SPARKPOST_API_KEY_INGEST not set - stopping")
	}
	client := spmta.MyRedis()

	// Process forever data arriving via Redis queue
	trackingData := make([]string, 0, ingestBatchSize) // Pre-allocate for efficiency
	for {
		d, err := client.LPop(spmta.RedisQueue).Result()
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
