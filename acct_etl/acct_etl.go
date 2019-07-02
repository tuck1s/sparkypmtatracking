package main

import (
	"bufio"
	"log"
	"os"
)

func main() {
	logfile, err := os.OpenFile("acct_etl.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", logfile, ":", err)
	}
	log.SetOutput(logfile)

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		ln := input.Text()
		log.Println(ln)
	}
}
