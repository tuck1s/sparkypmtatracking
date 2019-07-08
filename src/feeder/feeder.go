package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	. "github.com/sparkyPmtaTracking/src/common"

	"github.com/go-redis/redis"
)

// For efficiency under high load conditions, collect n events into a batch
const ingestBatchSize = 10000
const ingestMaxWait = 10 * time.Second

func makeSparkPostEvent() {
}

func ingest(batch []string) {
	for _, eStr := range batch {
		var e TrackEvent
		err := json.Unmarshal([]byte(eStr), &e)
		Check(err)
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
			// special value means queue is empty. SparkPost ingest any data we have collected, then wait a while
			if len(trackingData) > 0 {
				ingest(trackingData)
				trackingData = trackingData[:0] // empty the data, but keep capacity allocated
			}
			fmt.Println("Sleeping ..")
			time.Sleep(ingestMaxWait)
		} else {
			if err != nil {
				log.Println(err)
			} else {
				// stash data away for later. If we have a full batch, SparkPost ingest it
				trackingData = append(trackingData, d)
				if len(trackingData) >= ingestBatchSize {
					ingest(trackingData)
					trackingData = trackingData[:0] // empty the data, but keep capacity allocated
				}
			}
		}
	}
}
