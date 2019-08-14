package main

import (
	. "github.com/sparkyPmtaTracking/src/common"
	"log"
	"net/http"
	"os"
)

func TrackingServer(w http.ResponseWriter, req *http.Request) {
	// Expects URL paths of the form /tracking/open/xyzzy and /tracking/click/xyzzy
	// where xyzzy = base64 urlsafe encoded, Zlib compressed, []byte
	// These are written to the Redis queue
	log.Println(req.URL.Path)
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte("OK\n"))
	if err != nil {
		log.Println("http.ResponseWriter error", err)
	}
}

func main() {
	// Use logging, as this program will be executed without an attached console
	logfile, err := os.OpenFile("test.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	Check(err)
	log.SetOutput(logfile)

	http.HandleFunc("/tracking/", TrackingServer) // Accept subtree matches
	server := &http.Server{
		Addr: ":8888",
	}
	err = server.ListenAndServe()
	Check(err)
}
