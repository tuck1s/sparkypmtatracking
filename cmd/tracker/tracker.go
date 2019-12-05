package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

// Declare this in package scope, as it's unchanging
var transparentGif = []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff" +
	"\xff\xff\xff\x21\xf9\x04\x01\x0a\x00\x01\x00\x2c\x00\x00\x00\x00" +
	"\x01\x00\x01\x00\x00\x02\x02\x4c\x01\x00\x3b\x00")

// TrackingServer expects URL paths of the form /xyzzy
// where xyzzy = base64 urlsafe encoded, Zlib compressed, []byte
// These are written to the Redis queue
func trackingServer(w http.ResponseWriter, req *http.Request) {
	s := strings.Split(req.URL.Path, "/")
	if s[0] != "" || len(s) != 2 {
		log.Println("Incoming URL error:", req.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var e spmta.TrackEvent
	e.UserAgent = req.UserAgent() // add user agent
	e.IPAddress, _, _ = net.SplitHostPort(req.RemoteAddr)
	t := time.Now().Unix() // add timestamp
	e.TimeStamp = strconv.FormatInt(t, 10)

	eBytes, err := spmta.DecodePath(s[1])
	if err != nil {
		log.Println(err)
	}
	if err = json.Unmarshal(eBytes, &e.WD); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Build the composite info ready to push into the Redis queue
	eBytes, err = json.Marshal(e)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Log information received
	log.Printf("Timestamp %s, IPAddress %s, UserAgent %s, Action %s, URL %s, MsgID %s\n", e.TimeStamp, e.IPAddress, e.UserAgent, e.WD.Action, e.WD.TargetLinkURL, e.WD.MessageID)

	client := spmta.MyRedis()
	defer client.Close()
	if _, err = client.RPush(spmta.RedisQueue, eBytes).Result(); err != nil {
		log.Println("Redis error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Emulate response that SparkPost gives on GET opens, clicks and OPTIONS method. Change as required
	w.Header().Set("Server", "msys-http")
	switch req.Method {
	case "GET":
		switch e.WD.Action {
		case "o":
		case "i":
			w.Header().Set("Content-Type", "image/gif")
			w.Header().Set("Cache-Control", "no-cache, max-age=0")
			if _, err = w.Write(transparentGif); err != nil {
				log.Println("http.ResponseWriter error", err)
			}
		case "c":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Location", e.WD.TargetLinkURL)
			w.WriteHeader(http.StatusFound)
		}
	default:
		// Emulate what SparkPost engagement tracker endpoint does. Not strictly necessary for PMTA
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	inHostPort := flag.String("in_hostport", ":8888", "host:port to serve incoming HTTP requests")
	logfile := flag.String("logfile", "", "File written with message logs")
	//verboseOpt := flag.Bool("verbose", false, "print out lots of messages")
	flag.Parse()
	spmta.MyLogger(*logfile)
	fmt.Println("Starting http server on", *inHostPort, ", logging to", *logfile)
	log.Println("Starting http server on", *inHostPort)
	// http server runs in plain, as it will be proxied (e.g. by nginx) that can provide https
	http.HandleFunc("/", trackingServer) // Accept subtree matches
	server := &http.Server{
		Addr: *inHostPort,
	}
	err := server.ListenAndServe()
	if err != nil {
		spmta.ConsoleAndLogFatal(err)
	}
}
