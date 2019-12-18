package sparkypmtatracking

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

// Make a SparkPost formatted unique event_id, which needs to be a decimal string 0 .. (2^63-1)
func uniqEventID() string {
	u := uuid.New()
	num := binary.LittleEndian.Uint64(u[:8]) & 0x7fffffffffffffff
	return strconv.FormatUint(num, 10)
}

func makeSparkPostEvent(eStr string, client *redis.Client) (SparkPostEvent, error) {
	var tev TrackEvent
	var spEvent SparkPostEvent
	if err := json.Unmarshal([]byte(eStr), &tev); err != nil {
		return spEvent, err
	}
	// Shortcut pointer to the attribute-carrying leaf object; fill in received attributes
	eptr := &spEvent.EventWrapper.EventGrouping
	eptr.Type = ActionToType(tev.WD.Action)
	eptr.TargetLinkURL = tev.WD.TargetLinkURL
	eptr.MessageID = tev.WD.MessageID
	eptr.TimeStamp = tev.TimeStamp
	eptr.UserAgent = tev.UserAgent
	eptr.IPAddress = tev.IPAddress

	// Enrich with PowerMTA accounting-pipe values, if we have these, from persistent storage
	tKey := TrackingPrefix + tev.WD.MessageID
	if enrichmentJSON, err := client.Get(tKey).Result(); err == redis.Nil {
		log.Println("Warning: redis key", tKey, "not found, url=", tev.WD.TargetLinkURL)
	} else {
		enrichment := make(map[string]string)
		err = json.Unmarshal([]byte(enrichmentJSON), &enrichment)
		if err != nil {
			return spEvent, err
		}
		eptr.RcptTo = enrichment["rcpt"]
		eptr.SubaccountID = SafeStringToInt(enrichment["header_x-sp-subaccount-id"])
	}

	// Fill in these fields with default / unique / derived values
	eptr.DelvMethod = "esmtp"
	eptr.EventID = uniqEventID()
	// Skip these fields for now; you may have information to populate them from your own sources
	// 	eptr.GeoIP
	return spEvent, nil
}

// Enrich and format a SparkPost event into NDJSON
func sparkPostEventNDJSON(eStr string, client *redis.Client) []byte {
	e, err := makeSparkPostEvent(eStr, client)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	eJSON, err := json.Marshal(e)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	eJSON = append(eJSON, byte('\n'))
	return eJSON
}

// Prepare a batch of TrackingEvents for ingest to SparkPost.
func sparkPostIngest(ingestData []byte, client *redis.Client, host string, apiKey string) error {
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
	var resObj IngestResult
	err = json.Unmarshal(responseBody, &resObj)
	if err != nil {
		return err
	}
	if resObj.Errors != nil && len(resObj.Errors) > 0 {
		log.Printf("Uploaded %d bytes raw, %d bytes gzipped. SparkPost response: %s, errors[0]= %s\n",
			len(ingestData), gzipSize, res.Status, resObj.Errors[0].Message)
	}
	if resObj.Results.ID != "" {
		log.Printf("Uploaded %d bytes raw, %d bytes gzipped. SparkPost Ingest response: %s, results.id=%s\n",
			len(ingestData), gzipSize, res.Status, resObj.Results.ID)
	}
	err = res.Body.Close()
	return err
}

type myTimedBuffer struct {
	content     []byte
	timeStarted time.Time
	maxAge      time.Duration
}

// matureContent returns true if the buffer has contents that are now older than the specified maturity time
func (t *myTimedBuffer) matureContent() bool {
	age := time.Since(t.timeStarted)
	return len(t.content) > 0 && age >= t.maxAge
}

// FeedForever sends data arriving via Redis queue to SparkPost ingest API.
// Send a batch periodically, or every X MB, whichever comes first.
func FeedForever(client *redis.Client, host string, apiKey string, maxAge time.Duration) {
	var tBuf myTimedBuffer
	tBuf.content = make([]byte, 0, SparkPostIngestMaxPayload) // Pre-allocate for efficiency
	tBuf.maxAge = maxAge
	for {
		d, err := client.LPop(RedisQueue).Result()
		if err == redis.Nil {
			// Queue is now empty - send this batch if it's old enough
			if tBuf.matureContent() {
				err = sparkPostIngest(tBuf.content, client, host, apiKey)
				if err != nil {
					log.Println(err)
				}
				tBuf.content = tBuf.content[:0] // empty the data, but keep capacity allocated
			}
			time.Sleep(1 * time.Second) // polling wait time
			continue
		}
		if err != nil {
			log.Println(err) // report a Redis error
			continue
		}
		thisEvent := sparkPostEventNDJSON(d, client)
		// If this event would make the content oversize, send what we already have
		if len(tBuf.content)+len(thisEvent) >= SparkPostIngestMaxPayload {
			err = sparkPostIngest(tBuf.content, client, host, apiKey)
			if err != nil {
				log.Println(err)
			}
			tBuf.content = tBuf.content[:0] // empty the data, but keep capacity allocated
		}
		if len(tBuf.content) == 0 {
			// mark time of this event being placed into an empty buffer
			tBuf.timeStarted = time.Now()
		}
		tBuf.content = append(tBuf.content, thisEvent...)
	}
}
