package sparkypmtatracking

// TrackEvent data passed in this project's tracking URLs (and in the Redis event queue)
type TrackEvent struct {
	Type          string `json:"type"` // added from the URL literal path
	TargetLinkURL string `json:"target_link_url"`
	MessageID     string `json:"x_sp_message_id"`
	TimeStamp     string `json:"timestamp"`
	UserAgent     string `json:"user_agent"`
	IPAddress     string `json:"ip_address"`
}

// SparkPostEvent structure for SparkPost Ingest API. Note the nesting. There are some fields we're not populating:
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
			IPAddress     string   `json:"ip_address"`
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
			SendingIP     string   `json:"sending_ip"`
			Subject       string   `json:"subject"`
			TimeStamp     string   `json:"timestamp"`
			TargetLinkURL string   `json:"target_link_url"`
			UserAgent     string   `json:"user_agent"`
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
