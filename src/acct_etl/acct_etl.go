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
	const ttl = time.Duration(time.Hour * 24 * 10)

	// Scan input records - required fields for enrichment must include type, header_x-sp-message-id.
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

	input := csv.NewScanner(inbuf)
	for input.Scan() {
		r := input.Record()
		hdrs := make(map[string]int)
		switch r[0] {
		case deliveryType:
			hdrsJ, err := client.Get(RedisAcctHeaders).Result()
			if err == redis.Nil {
				Console_and_log_fatal("Error: redis key", RedisAcctHeaders, "not found")
			}
			err = json.Unmarshal([]byte(hdrsJ), &hdrs)
			Check(err)
			// TODO: read fields from r into a message_id-specific redis key that enables the feeder to enrich engagement events
			msgID_i, ok := hdrs[msgIDField]
			if ok {
				msgID := "msgID_" + r[msgID_i]
				// Set key message_id in Redis. We don't keep from/to at the moment
				v := "fred"
				_, err = client.Set(msgID, v, ttl).Result()
				Check(err)
				log.Println("Loaded", msgID, "->", v, "into Redis")
			} else {
				Console_and_log_fatal("Error: redis key", RedisAcctHeaders, "does not contain mapping for header_x-sp-message-id")
			}

		case typeField:
			// Acccounting header record sent at PowerMTA start, check for required and optional fields.
			// Write these into persistent storage, so that we can decode "d" records in future, separate process invocations.
			log.Println("PMTA accounting headers from pipe ", r)
			for _, f := range requiredAcctFields {
				fpos, found := PositionIn(r, f)
				if found {
					hdrs[f] = fpos
				} else {
					Console_and_log_fatal("Required field", f, "is not present in PMTA accounting headers")
				}
			}
			// Pick up positions of optional fields
			for _, f := range optionalAcctFields {
				fpos, found := PositionIn(r, f)
				if found {
					hdrs[f] = fpos
				}
			}
			hdrsJ, err := json.Marshal(hdrs)
			Check(err)
			_, err = client.Set(RedisAcctHeaders, hdrsJ, 0).Result()
			Check(err)

		default:
			Console_and_log_fatal("Accounting record not of type d:", r)
		}
	}
}
