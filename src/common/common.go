package common

import (
	"fmt"
	"log"
)

func Check(e error) {
	if e != nil {
		Console_and_log_fatal(e)
	}
}

func Console_and_log_fatal(s ...interface{}) {
	fmt.Println(s...)
	log.Fatalln(s...)
}

const RedisQueue = "trk_queue"

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

// Tracking event for SparkPost Ingest API. Note the double wrapping. There are some fields we're not populating:
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
			GeoIP         struct{} `json:"geo_ip"`
			InitialPixel  bool     `json:"initial_pixel"`
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
