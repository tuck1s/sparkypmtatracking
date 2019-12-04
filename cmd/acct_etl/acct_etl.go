package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	spmta "github.com/tuck1s/sparkyPMTATracking"

	"github.com/go-redis/redis"
	"github.com/smartystreets/scanners/csv"
)

// Event information in redis will have this time-to-live
const ttl = time.Duration(time.Hour * 24 * 10)

// Scan input accounting records - required fields for enrichment must include type, header_x-sp-message-id.
// Delivery records are of type "d".
// These fields should match your /etc/pmta/config 
const typeField = "type"
const msgIDField = "header_x-sp-message-id"
const deliveryType = "d"

var requiredAcctFields = []string{
	typeField, msgIDField,
}
var optionalAcctFields = []string{
	"rcpt", "vmtaPool", "header_x-sp-subaccount-id",
}

// Acccounting header record sent at PowerMTA start, check for required and optional fields.
// Write these into persistent storage, so that we can decode "d" records in future, separate process invocations.
func storeHeaders(r []string, client *redis.Client) error {
	log.Println("PMTA accounting headers from pipe ", r)
	hdrs := make(map[string]int)
	for _, f := range requiredAcctFields {
		fpos, found := spmta.PositionIn(r, f)
		if found {
			hdrs[f] = fpos
		} else {
			spmta.ConsoleAndLogFatal("Required field", f, "is not present in PMTA accounting headers")
		}
	}
	// Pick up positions of optional fields, for event enrichment
	for _, f := range optionalAcctFields {
		fpos, found := spmta.PositionIn(r, f)
		if found {
			hdrs[f] = fpos
		}
	}
	hdrsJSON, err := json.Marshal(hdrs)
	if err != nil {
		return err
	}
	_, err = client.Set(spmta.RedisAcctHeaders, hdrsJSON, 0).Result()
	if err != nil {
		return err
	}
	log.Println("Loaded", spmta.RedisAcctHeaders, "->", string(hdrsJSON), "into Redis")
	return nil
}

// Store a single accounting event r into redis, based on previously seen header format
func storeEvent(r []string, client *redis.Client) error {
	hdrsJ, err := client.Get(spmta.RedisAcctHeaders).Result()
	if err == redis.Nil {
		spmta.ConsoleAndLogFatal("Error: redis key", spmta.RedisAcctHeaders, "not found")
	}
	hdrs := make(map[string]int)
	err = json.Unmarshal([]byte(hdrsJ), &hdrs)
	if err != nil {
		return err
	}
	// read fields from r into a message_id-specific redis key that will enable the feeder to enrich engagement events
	msgIDindex, ok := hdrs[msgIDField]
	if !ok {
		spmta.ConsoleAndLogFatal("Error: redis key", spmta.RedisAcctHeaders, "does not contain mapping for header_x-sp-message-id")
	}
	msgIDKey := spmta.TrackingPrefix + r[msgIDindex]
	enrichment := make(map[string]string)
	for k, i := range hdrs {
		if k != msgIDField && k != typeField {
			enrichment[k] = r[i]
		}
	}
	// Set key message_id in Redis
	enrichmentJSON, err := json.Marshal(enrichment)
	if err != nil {
		return err
	}
	_, err = client.Set(msgIDKey, enrichmentJSON, ttl).Result()
	if err != nil {
		return err
	}
	log.Println("Loaded", msgIDKey, "->", string(enrichmentJSON), "into Redis")
	return nil
}

func main() {
	logfile := flag.String("logfile", "", "File written with message logs")
	infile := flag.String("infile", "", "Input file (omit to read from stdin)")

	flag.Parse()
	spmta.MyLogger(*logfile)

	var f *os.File
	var err error
	if *infile == "" {
		f = os.Stdin
	} else {
		f, err = os.Open(*infile)
		if err != nil {
			spmta.ConsoleAndLogFatal(err)
		}
	}
	inbuf := bufio.NewReader(f)
	client := spmta.MyRedis()
	input := csv.NewScanner(inbuf)
	for input.Scan() {
		r := input.Record()
		switch r[0] {
		case deliveryType:
			err := storeEvent(r, client)
			if err != nil {
				spmta.ConsoleAndLogFatal(err)
			}
		case typeField:
			err := storeHeaders(r, client)
			if err != nil {
				spmta.ConsoleAndLogFatal(err)
			}
		default:
			spmta.ConsoleAndLogFatal("Accounting record not of type d:", r)
		}
	}
}
