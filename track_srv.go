package sparkypmtatracking

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TransparentGif contains the bytes that should be served back to the client for an open pixel
var TransparentGif = []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff" +
	"\xff\xff\xff\x21\xf9\x04\x01\x0a\x00\x01\x00\x2c\x00\x00\x00\x00" +
	"\x01\x00\x01\x00\x00\x02\x02\x4c\x01\x00\x3b\x00")

// TrackingServer expects URL paths of the form /xyzzy
// where xyzzy = base64 urlsafe encoded, Zlib compressed, []byte
// These are written to the Redis queue
func TrackingServer(w http.ResponseWriter, req *http.Request) {
	// Emulate what SparkPost engagement tracker endpoint does. Necessary only for testing with bouncy sink.
	w.Header().Set("Server", "msys-http")
	switch req.Method {
	case "GET":
		break
	default:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s := strings.Split(req.URL.Path, "/")
	if s[0] != "" || len(s) != 2 || len(s[1]) <= 0 {
		log.Println("Incoming URL error:", req.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var e TrackEvent
	e.UserAgent = req.UserAgent() // add user agent
	e.IPAddress, _, _ = net.SplitHostPort(req.RemoteAddr)
	e.TimeStamp = strconv.FormatInt(time.Now().Unix(), 10)

	eBytes, err := DecodePath(s[1])
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
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

	client := MyRedis()
	defer client.Close()
	if _, err = client.RPush(RedisQueue, eBytes).Result(); err != nil {
		log.Println("Redis error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch e.WD.Action {
	case "o":
		fallthrough
	case "i":
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "no-cache, max-age=0")
		if _, err = w.Write(TransparentGif); err != nil {
			log.Println("http.ResponseWriter error", err)
		}
	case "c":
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Location", e.WD.TargetLinkURL)
		w.WriteHeader(http.StatusFound)
	}
}
