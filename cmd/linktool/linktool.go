package main

import (
	"flag"
	"fmt"
	"os"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

func usageNQuit() {
	flag.Usage()
	os.Exit(1)
}

func main() {
	encodeCmd := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeMessageID := encodeCmd.String("message_id", "0000123456789abcdef0", "message_id")
	encodeRcptTo := encodeCmd.String("rcpt_to", "any@example.com", "rcpt_to")
	encodeAction := encodeCmd.String("action", "open", "[open|initial_open|click]")
	encodeTargetLinkURL := encodeCmd.String("target_link_url", "https://example.com", "URL of your target link")
	encodeTrackingURL := encodeCmd.String("tracking_url", "http://localhost:8888", "URL of your tracking service endpoint")
	encodeCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "encode\n")
		encodeCmd.PrintDefaults()
	}

	decodeCmd := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "decode url\n")
	}

	flag.Usage = func() {
		const helpText = "%s [encode|decode] encode and decode link URLs\n"
		fmt.Fprintf(flag.CommandLine.Output(), helpText, os.Args[0])
		encodeCmd.Usage()
		decodeCmd.Usage()
	}

	if len(os.Args) < 2 {
		usageNQuit()
	}
	switch os.Args[1] {
	case encodeCmd.Name():
		if err := encodeCmd.Parse(os.Args[2:]); err != nil {
			usageNQuit()
		}
		link, err := spmta.EncodeLink(*encodeTrackingURL, *encodeAction, *encodeMessageID, *encodeRcptTo, *encodeTargetLinkURL, true, true, true)
		if err != nil {
			fmt.Println(err)
			usageNQuit()
		}
		fmt.Println(link)

	case decodeCmd.Name():
		if err := decodeCmd.Parse(os.Args[2:]); err != nil {
			usageNQuit()
		}
		if len(os.Args) < 3 {
			usageNQuit()
		}

		eBytes, wd, decodeTrackingURL, err := spmta.DecodeLink(os.Args[2])
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("JSON: %s\n", string(eBytes))
		fmt.Printf("Equivalent to encode -tracking_url %s -rcpt_to %s -action %s -target_link_url %s -message_id %s\n",
			decodeTrackingURL, wd.RcptTo, spmta.ActionToType(wd.Action), wd.TargetLinkURL, wd.MessageID)

	default:
		usageNQuit()
	}
}
