package common

import (
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"os"
	"strings"
)

// -------------------------------------------------------------------------------------------------------------------
// Error handling and logging
func Check(e error) {
	if e != nil {
		Console_and_log_fatal(e)
	}
}

func Console_and_log_fatal(s ...interface{}) {
	fmt.Println(s...)
	log.Fatalln(s...)
}

// Set a custom logger
func MyLogger(filename string) {
	logfile, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	Check(err)
	log.SetOutput(logfile)
}

// -------------------------------------------------------------------------------------------------------------------
// Helper function for environment variables
func GetenvDefault(k string, d string) string {
	x := os.Getenv(k)
	if x == "" {
		x = d
	}
	return x
}

// -------------------------------------------------------------------------------------------------------------------
// Clean up SparkPost host address into canonical form (with schema, without /api/v1 path)
func HostCleanup(host string) string {
	if !strings.HasPrefix(host, "https://") {
		host = "https://" + host // Add schema
	}
	host = strings.TrimSuffix(host, "/")
	host = strings.TrimSuffix(host, "/api/v1")
	host = strings.TrimSuffix(host, "/")
	return host
}

// -------------------------------------------------------------------------------------------------------------------
// Find an element within an array slice
func PositionIn(arr []string, val string) (int, bool) {
	for i, v := range arr {
		if v == val {
			return i, true
		}
	}
	return 0, false
}

// -------------------------------------------------------------------------------------------------------------------
// Redis access
const RedisQueue = "trk_queue"          // Name of the queue between tracker and feeder tasks
const RedisAcctHeaders = "acct_headers" // Key that holds the PowerMTA accounting file headers

// return a client handle for Redis. Assume server is on the standard port
func MyRedis() (client *redis.Client) {
	return redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// Tracking event data passed in this project's tracking URLs (and in the Redis event queue)
type TrackEvent struct {
	Type          string   `json:"type"` // added from the URL literal path
	CampaignID    string   `json:"campaign_id"`
	RcptTo        string   `json:"rcpt_to"`
	MsgFrom       string   `json:"msg_from"`
	RcptMeta      struct{} `json:"rcpt_meta"`
	Subject       string   `json:"subject"`
	TimeStamp     string   `json:"timestamp"` // Added by tracker
	TargetLinkUrl string   `json:"target_link_url"`
	TrackingID    string   `json:"tracking_id"`
	UserAgent     string   `json:"user_agent"` // Added by tracker
}

// -------------------------------------------------------------------------------------------------------------------
// Tracking event for SparkPost Ingest API. Note the nesting. There are some fields we're not populating:
// ab_test_id, ab_test_version, injection_time, ip_address, ip_pool, msg_size, num_retries, queue_time,
// raw_rcpt_to, rcpt_type, sending_ip, subaccount_id, target_link_name, template_id, template_version, transactional,
// transmission_id, binding_group, binding
type SparkPostEvent struct {
	EventWrapper struct {
		EventGrouping struct {
			Type          string   `json:"type"`
			CampaignID    string   `json:"campaign_id"`
			ClickTracking bool     `json:"click_tracking"`
			DelvMethod    string   `json:"delv_method"`
			EventID       string   `json:"event_id"`
			FriendlyFrom  string   `json:"friendly_from"`
			GeoIP         GeoIP    `json:"geo_ip"`
			InitialPixel  bool     `json:"initial_pixel"`
			IPPool        string   `json:"ip_pool"`
			MessageID     string   `json:"message_id"`
			RcptTags      []string `json:"rcpt_tags"`
			RoutingDomain string   `json:"routing_domain"`
			RcptTo        string   `json:"rcpt_to"`
			OpenTracking  bool     `json:"open_tracking"`
			MsgFrom       string   `json:"msg_from"`
			RcptMeta      struct{} `json:"rcpt_meta"`
			Subject       string   `json:"subject"`
			TimeStamp     string   `json:"timestamp"`
			TargetLinkUrl string   `json:"target_link_url"`
			UserAgent     string   `json:"user_agent"`
		} `json:"track_event"`
	} `json:"msys"`
}

// Result object coming back from SparkPost
type IngestResult struct {
	Results struct {
		Id string `json:"id"`
	} `json:"results"`
}

type GeoIP struct {
	Country    string
	Region     string
	City       string
	Latitude   float64
	Longitude  float64
	Zip        int
	PostalCode string
}
