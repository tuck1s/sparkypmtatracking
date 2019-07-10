package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"github.com/go-redis/redis"
	. "github.com/sparkyPmtaTracking/src/common"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func TrackingServer(w http.ResponseWriter, req *http.Request) {
	// Expects URL paths of the form /tracking/open/xyzzy and /tracking/click/xyzzy
	// where xyzzy = base64 urlsafe encoded, Zlib compressed, []byte
	// These are written to the Redis queue

	s := strings.Split(req.URL.Path[1:], "/")
	if len(s) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var e TrackEvent
	e.Type = s[1] // add the event type in from the path
	e.UserAgent = req.UserAgent()
	t := time.Now().Unix()
	e.TimeStamp = strconv.FormatInt(t, 10)
	if e.Type != "open" && e.Type != "click" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	zdata, err := base64.URLEncoding.DecodeString(s[2])
	if err != nil {
		log.Println("Invalid base64 url part found:", s[2])
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	eReader, err := zlib.NewReader(bytes.NewReader(zdata))
	if err != nil {
		log.Println("Invalid zlib data found:", zdata)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	eBytes, err := ioutil.ReadAll(eReader) // []byte representation of JSON
	err = json.Unmarshal(eBytes, &e)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	eBytes, err = json.Marshal(e)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println(e)

	// Prepare to load a record into Redis. Assume server is on the standard port
	client := redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err = client.RPush(RedisQueue, eBytes).Result()
	if err != nil {
		log.Println("Redis error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Emulate response that SparkPost gives on GET opens, clicks and OPTIONS method. Change as required
	w.Header().Set("Server", "msys-http")
	w.Header().Set("X-Robots-Tag", "noindex")

	// Special value expected by Bouncy Sink. Not needed for production applications
	w.Header().Set("X-MSYS", "Signals SMTP Traffic Generator Tracking Endpoint")
	switch req.Method {
	case "GET":
		switch e.Type {
		case "open":
			w.Header().Set("Content-Type", "image/gif")
			w.Header().Set("Cache-Control", "no-cache, max-age=0")
			transparentGif := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff" +
				"\xff\xff\xff\x21\xf9\x04\x01\x0a\x00\x01\x00\x2c\x00\x00\x00\x00" +
				"\x01\x00\x01\x00\x00\x02\x02\x4c\x01\x00\x3b\x00")
			_, err = w.Write(transparentGif)
			if err != nil {
				log.Println("http.ResponseWriter error", err)
			}
		case "click":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Location", e.TargetLinkUrl)
			w.WriteHeader(http.StatusFound)
		}
	case "OPTIONS":
		// Emulate what SparkPost engagement tracker endpoint does. Not strictly necessary for PMTA
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	// Use logging, as this program will be executed without an attached console
	logfile, err := os.OpenFile("tracker.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	Check(err)
	log.SetOutput(logfile)

	http.HandleFunc("/tracking/", TrackingServer) // Accept subtree matches
	server := &http.Server{
		Addr: ":8888",
	}
	err = server.ListenAndServe()
	Check(err)
}
