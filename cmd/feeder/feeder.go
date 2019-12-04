package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strconv"
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

func safeStringToInt(s string) int {
	if s == "" {
		return 0 // Handle case where master account has blank/no header in data
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Println("Warning: cannot convert", s, "to int")
		i = 0
	}
	return i
}

// For efficiency under load, collect n events into a batch
const ingestBatchSize = 1000

func ingestMaxWait() time.Duration {
	if runtime.GOOS == "darwin" {
		return 1 * time.Second // developer setting
	}
	return 300 * time.Second // production setting
}

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
			return spEvent, err
		}
		// eptr.MsgFrom = enrichment["orig"]
		eptr.RcptTo = enrichment["rcpt"]
		// eptr.CampaignID = enrichment["jobId"]
		// eptr.SendingIP = enrichment["dlvSourceIp"]
		// eptr.IPPool = enrichment["vmtaPool"]
		// eptr.RoutingDomain = strings.Split(eptr.RcptTo, "@")[1]
		eptr.SubaccountID = safeStringToInt(enrichment["header_x-sp-subaccount-id"])
	}

	// Fill in these fields with default / unique / derived values
	eptr.DelvMethod = "esmtp"
	eptr.EventID = uniqEventID()
	// eptr.InitialPixel = true // These settings reflect capability of wrapper function
	// eptr.ClickTracking = true
	// eptr.OpenTracking = true

	// Skip these fields for now; you may have information to populate them from your own sources
	//eptr.GeoIP, eptr.FriendlyFrom, eptr.RcptTags, eptr.RcptMeta
	//eptr.Subject			.. SparkPost doesn't fill this in on Open & Click events
	return spEvent, nil
}

// Prepare a batch of TrackingEvents for ingest to SparkPost.
func sparkPostIngest(batch []string, client *redis.Client, host string, apiKey string) error {
	ingestData := make([]byte, 0, 5*1024*1024)
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
		ingestData = append(ingestData, eJSON...)
		ingestData = append(ingestData, byte('\n'))
	}
	// Now have ingestData ndJSON in byte slice form - gzip compress
	var zbuf bytes.Buffer
	zw := gzip.NewWriter(&zbuf)
	_, err := zw.Write(ingestData)
	if err != nil {
		return err
	}
	err = zw.Close() // ensure all data written (seems to be necessary)
	if err != nil {
		return err
	}
	gzipSize := zbuf.Len()

	// Prepare the https POST request. We have to supply a Reader for this, hence needing to realize the stream via zbuf
	zr := bufio.NewReader(&zbuf)
	var netClient = &http.Client{
		Timeout: time.Second * 300,
	}
	url := host + "/api/v1/ingest/events"
	req, err := http.NewRequest("POST", url, zr)
	if err != nil {
		return err
	}
	req.Header = map[string][]string{
		"Authorization":    {apiKey},
		"Content-Type":     {"application/x-ndjson"},
		"Content-Encoding": {"gzip"},
	}
	res, err := netClient.Do(req)
	if err != nil {
		return err
	}
	// Response body is a Reader; read it into []byte
	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var resObj spmta.IngestResult
	err = json.Unmarshal(responseBody, &resObj)
	if err != nil {
		return err
	}
	log.Printf("Uploaded %d events, %d bytes (gzip). SparkPost Ingest response: %s, results.id=%s\n", len(batch), gzipSize, res.Status, resObj.Results.ID)
	err = res.Body.Close()
	return err
}

func main() {
	logfile := flag.String("logfile", "", "File written with message logs")
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
				err = sparkPostIngest(trackingData, client, host, apiKey)
				if err != nil {
					log.Println(err.Error())
				}
				trackingData = trackingData[:0] // empty the data, but keep capacity allocated
			}
			time.Sleep(ingestMaxWait())
		} else {
			if err != nil {
				log.Println(err)
			} else {
				// stash data away for later. If we have a full batch, SparkPost sparkPostIngest it
				trackingData = append(trackingData, d)
				if len(trackingData) >= ingestBatchSize {
					err = sparkPostIngest(trackingData, client, host, apiKey)
					if err != nil {
						log.Println(err.Error())
					}
					trackingData = trackingData[:0] // empty the data, but keep capacity allocated
				}
			}
		}
	}
}
