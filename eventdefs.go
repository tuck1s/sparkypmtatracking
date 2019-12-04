package sparkypmtatracking

// TrackEvent is the augmented info passed from "tracker" via the Redis event queue to "feeder"
type TrackEvent struct {
	WD        WrapperData
	TimeStamp string `json:"ts"`
	UserAgent string `json:"ua"`
	IPAddress string `json:"ip"`
}

// SparkPostEvent structure for SparkPost Ingest API. Note the nesting. There are some fields we're not populating here,
// as they will automatically be "enriched" by SparkPost, providing there is a matching injection event.
//    ab_test_id, ab_test_version, amp_enabled, binding, binding_group, campaign_id, click_tracking,
//    friendly_from, initial_pixel, injection_time, ip_pool, ip_pool_raw, msg_from, msg_size, open_tracking,
//    rcpt_meta, rcpt_tags, rcpt_type, recv_method, routing_domain, sending_ip, subject, template_id, template_version, transactional,
//    transmission_id
//
// We are also not populating: num_retries, queue_time, raw_rcpt_to, target_link_name, binding_group, binding
// A future implementation could usefully populate target_link_name if desired
type SparkPostEvent struct {
	EventWrapper struct {
		EventGrouping struct {
			Type string `json:"type"`
			// CampaignID    string   `json:"campaign_id"`
			// ClickTracking bool     `json:"click_tracking"`
			DelvMethod string `json:"delv_method"`
			EventID    string `json:"event_id"`
			// FriendlyFrom  string   `json:"friendly_from"`
			IPAddress string `json:"ip_address"`
			GeoIP     GeoIP  `json:"geo_ip"`
			// InitialPixel  bool     `json:"initial_pixel"`
			// IPPool        string   `json:"ip_pool"`
			MessageID string `json:"message_id"`
			// RcptTags      []string `json:"rcpt_tags"`
			// RoutingDomain string   `json:"routing_domain"`
			RcptTo string `json:"rcpt_to"`
			// OpenTracking  bool     `json:"open_tracking"`
			// MsgFrom       string   `json:"msg_from"`
			// RcptMeta      struct{} `json:"rcpt_meta"`
			// SendingIP     string   `json:"sending_ip"`
			// Subject       string   `json:"subject"`
			TimeStamp     string `json:"timestamp"`
			TargetLinkURL string `json:"target_link_url"`
			UserAgent     string `json:"user_agent"`
			SubaccountID  int    `json:"subaccount_id"`
		} `json:"track_event"`
	} `json:"msys"`
}

// IngestResult object coming back from SparkPost
type IngestResult struct {
	Results struct {
		ID string `json:"id"`
	} `json:"results"`
}

// GeoIP data expected by SparkPost
type GeoIP struct {
	Country    string
	Region     string
	City       string
	Latitude   float64
	Longitude  float64
	Zip        int
	PostalCode string
}
