package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	. "github.com/sparkyPmtaTracking/src/common"

	"github.com/go-redis/redis"
	"github.com/smartystreets/scanners/csv"
)

// Event information in redis will have this time-to-live
const ttl = time.Duration(time.Hour * 24 * 10)

// Scan input accounting records - required fields for enrichment must include type, header_x-sp-message-id.
// Delivery records are of type "d".
const typeField = "type"
const msgIDField = "header_x-sp-message-id"
const deliveryType = "d"

var requiredAcctFields = []string{
	typeField, msgIDField,
}
var optionalAcctFields = []string{
	"orig", "rcpt", "jobId", "dlvSourceIp", "vmta", "header_Subject",
}

// Acccounting header record sent at PowerMTA start, check for required and optional fields.
// Write these into persistent storage, so that we can decode "d" records in future, separate process invocations.
func storeHeaders(r []string, client *redis.Client) {
	log.Println("PMTA accounting headers from pipe ", r)
	hdrs := make(map[string]int)
	for _, f := range requiredAcctFields {
		fpos, found := PositionIn(r, f)
		if found {
			hdrs[f] = fpos
		} else {
			Console_and_log_fatal("Required field", f, "is not present in PMTA accounting headers")
		}
	}
	// Pick up positions of optional fields, for event enrichment
	for _, f := range optionalAcctFields {
		fpos, found := PositionIn(r, f)
		if found {
			hdrs[f] = fpos
		}
	}
	hdrsJSON, err := json.Marshal(hdrs)
	Check(err)
	_, err = client.Set(RedisAcctHeaders, hdrsJSON, 0).Result()
	Check(err)
	log.Println("Loaded", RedisAcctHeaders, "->", string(hdrsJSON), "into Redis")

}

// Store a single accounting event r into redis, based on previously seen header format
func storeEvent(r []string, client *redis.Client) {
	hdrsJ, err := client.Get(RedisAcctHeaders).Result()
	if err == redis.Nil {
		Console_and_log_fatal("Error: redis key", RedisAcctHeaders, "not found")
	}
	hdrs := make(map[string]int)
	err = json.Unmarshal([]byte(hdrsJ), &hdrs)
	Check(err)
	// read fields from r into a message_id-specific redis key that will enable the feeder to enrich engagement events
	if msgID_i, ok := hdrs[msgIDField]; !ok {
		Console_and_log_fatal("Error: redis key", RedisAcctHeaders, "does not contain mapping for header_x-sp-message-id")
	} else {
		msgIDKey := "msgID_" + r[msgID_i]
		enrichment := make(map[string]string)
		for k, i := range hdrs {
			if k != msgIDField && k != typeField {
				enrichment[k] = r[i]
			}
		}
		// Set key message_id in Redis
		enrichmentJSON, err := json.Marshal(enrichment)
		Check(err)
		_, err = client.Set(msgIDKey, enrichmentJSON, ttl).Result()
		Check(err)
		log.Println("Loaded", msgIDKey, "->", string(enrichmentJSON), "into Redis")
	}
}

func main() {
	flag.Parse()
	var f *os.File

	// Use logging, as this program will be executed without an attached console
	logfile, err := os.OpenFile("acct_etl.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	Check(err)
	log.SetOutput(logfile)

	// Check if input file specified on command-line args, or using stdin
	switch flag.NArg() {
	case 0:
		f = os.Stdin
	case 1:
		f, err = os.Open(flag.Arg(0))
		Check(err)
	default:
		Console_and_log_fatal("Command line args: input must be from stdin or file")
	}
	inbuf := bufio.NewReader(f)

	// Prepare to load records into Redis. Assume server is on the standard port
	client := redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	input := csv.NewScanner(inbuf)
	for input.Scan() {
		r := input.Record()
		switch r[0] {
		case deliveryType:
			storeEvent(r, client)
		case typeField:
			storeHeaders(r, client)
		default:
			Console_and_log_fatal("Accounting record not of type d:", r)
		}
	}
}
