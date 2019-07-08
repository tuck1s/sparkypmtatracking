package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	. "github.com/sparkyPmtaTracking/src/common"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

// Make a SparkPost formatted unique event_id, which needs to be a decimal string 0 .. (2^63-1)
func uniq_event_id() string {
	u := uuid.New()
	num := binary.LittleEndian.Uint64(u[:8]) & 0x7fffffffffffffff
	return strconv.FormatUint(num, 10)
}

// Map our own tracking ID to the SparkPost Message ID via Redis
func trackingIDToMessageID(eStr string, client *redis.Client) string {
	var t TrackEvent
	err := json.Unmarshal([]byte(eStr), &t)
	Check(err)
	//t.TrackingID = "a5a1050437e345389bc2c7d6d79743f5"			//TODO: debug
	msgID, err := client.Get("trk_" + t.TrackingID).Result()
	Check(err)
	return msgID
}

// For efficiency under high load conditions, collect n events into a batch
const ingestBatchSize = 10000
const ingestMaxWait = 10 * time.Second

// Because the JSON declarations coincide, we can unmarshal a TrackingEvent into part of a SparkPostEvent
func sparkPostIngest(batch []string, client *redis.Client) {
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
		eptr.EventID = uniq_event_id()
		eptr.InitialPixel = false
		eptr.MessageID = trackingIDToMessageID(eStr, client)
		eptr.RoutingDomain = strings.Split(eptr.RcptTo, "@")[1]
		eptr.ClickTracking = true
		eptr.OpenTracking = true
		s, err := json.Marshal(e)
		Check(err)
		fmt.Println(string(s))
	}
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
	trackingData := make([]string, 0, ingestBatchSize) // Pre-allocate for efficiency
	for {
		d, err := client.LPop(RedisQueue).Result()
		if err == redis.Nil {
			// special value means queue is empty. Ingest any data we have collected, then wait a while
			if len(trackingData) > 0 {
				sparkPostIngest(trackingData, client)
				trackingData = trackingData[:0] // empty the data, but keep capacity allocated
			}
			fmt.Println("Sleeping ..")
			time.Sleep(ingestMaxWait)
		} else {
			if err != nil {
				log.Println(err)
			} else {
				// stash data away for later. If we have a full batch, SparkPost sparkPostIngest it
				trackingData = append(trackingData, d)
				if len(trackingData) >= ingestBatchSize {
					sparkPostIngest(trackingData, client)
					trackingData = trackingData[:0] // empty the data, but keep capacity allocated
				}
			}
		}
	}
}
