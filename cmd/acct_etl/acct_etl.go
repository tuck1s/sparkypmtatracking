package main

import (
	"flag"
	"fmt"
	"os"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

func main() {
	logfile := flag.String("logfile", "", "File written with message logs")
	infile := flag.String("infile", "", "Input file (omit to read from stdin)")
	flag.Usage = func() {
		const helpText = "Extracts, transforms and loads accounting data fed by PowerMTA pipe into Redis" +
			"Usage of %s:\n"
		fmt.Fprintf(flag.CommandLine.Output(), helpText, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	spmta.MyLogger(*logfile)
	fmt.Printf("Starting acct_etl, logging to %s\n", *logfile)

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
	err = spmta.AccountETL(f)
	if err != nil {
		spmta.ConsoleAndLogFatal(err)
	}
}
