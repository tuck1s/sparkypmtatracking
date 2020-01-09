package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

func usageNQuit() {
	flag.Usage()
	os.Exit(1)
}

func main() {
	encodeCmd := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeTrackingURL := encodeCmd.String("tracking_url", "http://localhost:8888", "URL of your tracking service endpoint")
	encodeMessageID := encodeCmd.String("message_id", "0000123456789abcdef0", "message_id")
	encodeRcptTo := encodeCmd.String("rcpt_to", "any@example.com", "rcpt_to")
	encodeAction := encodeCmd.String("action", "open", "[open|initial_open|click]")
	encodeTargetLinkURL := encodeCmd.String("target_link_url", "https://example.com", "URL of your target link")
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
		w, err := spmta.NewWrapper(*encodeTrackingURL)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		w.SetMessageInfo(*encodeMessageID, *encodeRcptTo)
		switch *encodeAction {
		case "open":
			fmt.Println(w.OpenPixel())
		case "initial_open":
			fmt.Println(w.InitialOpenPixel())
		case "click":
			fmt.Println(w.WrapURL(*encodeTargetLinkURL))
		default:
			usageNQuit()
		}

	case decodeCmd.Name():
		if err := decodeCmd.Parse(os.Args[2:]); err != nil {
			usageNQuit()
		}
		if len(os.Args) < 3 {
			usageNQuit()
		}
		url, err := url.Parse(os.Args[2]) // pick up url directly from cmd line args
		if err != nil {
			fmt.Println(err)
		}
		pathBytes := strings.Split(url.Path, "/") // should be only one / separator
		if len(pathBytes) != 2 || pathBytes[0] != "" {
			fmt.Println("Unexpected URL path with more than one /")
			os.Exit(1)
		}
		eBytes, err := spmta.DecodePath(pathBytes[1])
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(eBytes))
		var tev spmta.TrackEvent
		wdp := &tev.WD
		if err := json.Unmarshal(eBytes, wdp); err != nil {
			fmt.Println(err)
		}
		decodeTrackingURL := url.Scheme + "://" + url.Host
		fmt.Printf("encode -tracking_url %s -rcpt_to %s -action %s -target_link_url %s -message_id %s\n",
			decodeTrackingURL, wdp.RcptTo, spmta.ActionToType(wdp.Action), wdp.TargetLinkURL, wdp.MessageID)

	default:
		usageNQuit()
	}
}
