package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/smartystreets/scanners/csv"
)

func check(e error) {
	if e != nil {
		console_and_log_fatal(e)
	}
}

func console_and_log_fatal(s ...interface{}) {
	fmt.Println(s...)
	log.Fatalln(s...)
}

func main() {
	flag.Parse()
	var f *os.File

	// Use logging, as this program will be executed without an attached console
	logfile, err := os.OpenFile("acct_etl.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	check(err)
	log.SetOutput(logfile)

	// Check if input file specified on command-line args, or using stdin
	switch flag.NArg() {
	case 0:
		f = os.Stdin
		break
	case 1:
		f, err = os.Open(flag.Arg(0))
		check(err)
		break
	default:
		console_and_log_fatal("Command line args: input must be from stdin or file")
	}
	inbuf := bufio.NewReader(f)

	// Prepare to load records into Redis. Assume server is on the standard port
	client := redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err = client.Ping().Result()
	check(err)

	const ttl = time.Duration(time.Hour * 24 * 10)
	const expectedLen = 5
	// Scan input records - expected format recType, fromAddr, RcptTo, message_id, tracking_id
	input := csv.NewScanner(inbuf)
	for input.Scan() {
		r := input.Record()
		if len(r) != expectedLen {
			console_and_log_fatal("Accounting record not in expected format - should have", expectedLen, "elements")
		}
		recType, fromAddr, RcptTo, message_id, tracking_id := r[0], r[1], r[2], r[3], r[4]
		if recType != "d" {
			console_and_log_fatal("Accounting record not of type d")
		}
		_, err := client.Set("trk_"+tracking_id, message_id, ttl).Result()
		check(err)
		log.Println("Loaded", tracking_id, "=", message_id, "into Redis, from=", fromAddr, "RcptTo=", RcptTo)
	}
}
