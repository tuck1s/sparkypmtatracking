package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	. "github.com/sparkyPmtaTracking/src/common"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-redis/redis"
)

func TrackingServer(w http.ResponseWriter, req *http.Request) {
	// Expects URL paths of the form /tracking/open/xyzzy and /tracking/click/xyzzy
	// where xyzzy = base64 urlsafe encoded, Zlib compressed, []byte
	// These are written to the Redis queue

	s := strings.Split(req.URL.Path[1:], "/")
	switch s[1] {
	case "open":
		break
	case "click":
		break
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	zdata, err := base64.URLEncoding.DecodeString(s[2])
	if err != nil {
		log.Println("Invalid base64 url part found:", s[2])
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	d, err := zlib.NewReader(bytes.NewReader(zdata))
	if err != nil {
		log.Println("Invalid zlib data found:", zdata)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data, err := ioutil.ReadAll(d) // data is a []byte representation of JSON data, ideal for queuing
	log.Println(string(data))

	// Prepare to load a record into Redis. Assume server is on the standard port
	client := redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err = client.RPush(RedisQueue, data).Result()
	if err != nil {
		log.Println("Redis error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write([]byte("OK\n"))
	if err != nil {
		log.Println("http.ResponseWriter error", err)
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
