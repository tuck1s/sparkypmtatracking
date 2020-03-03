package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	spmta "github.com/tuck1s/sparkypmtatracking"
)

func main() {
	const spHostEnvVar = "SPARKPOST_HOST_INGEST"
	const spAPIKeyEnvVar = "SPARKPOST_API_KEY_INGEST"
	logfile := flag.String("logfile", "", "File written with message logs")
	flag.Usage = func() {
		const helpText = "Takes the opens and clicks from the Redis queue and feeds them to the SparkPost Ingest API\n" +
			"Requires environment variable %s and optionally %s\n" +
			"Usage of %s:\n"
		fmt.Fprintf(flag.CommandLine.Output(), helpText, spAPIKeyEnvVar, spHostEnvVar, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	spmta.MyLogger(*logfile)
	if *logfile != "" {
		fmt.Println("Starting feeder service, logging to", *logfile)
	}
	log.Println("Starting feeder service")

	// Get SparkPost ingest info from env vars
	host := spmta.HostCleanup(spmta.GetenvDefault(spHostEnvVar, "api.sparkpost.com"))
	apiKey := spmta.GetenvDefault(spAPIKeyEnvVar, "")
	if apiKey == "" {
		spmta.ConsoleAndLogFatal(fmt.Sprintf("%s not set - stopping", spAPIKeyEnvVar))
	}

	spmta.FeedForever(spmta.MyRedis(), host, apiKey, spmta.SparkPostIngestBatchMaxAge)
}
