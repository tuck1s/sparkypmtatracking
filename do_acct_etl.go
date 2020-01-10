package sparkypmtatracking

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis"
	"github.com/smartystreets/scanners/csv"
)

// Scan input accounting records - required fields for augmentation are: type, header_x-sp-message-id.
// Delivery records are of type "d".
// These fields should match your /etc/pmta/config
const typeField = "type"
const msgIDField = "header_x-sp-message-id"
const deliveryType = "d"

var requiredAcctFields = []string{
	typeField, msgIDField,
}
var optionalAcctFields = []string{
	"rcpt", "header_x-sp-subaccount-id",
}

// Acccounting header record sent at PowerMTA start, check for required and optional fields.
// Write these into persistent storage, so that we can decode "d" records in future, separate process invocations.
func storeHeaders(r []string, client *redis.Client) error {
	log.Printf("PowerMTA accounting headers: %v\n", r)
	hdrs := make(map[string]int)
	for _, f := range requiredAcctFields {
		fpos, found := PositionIn(r, f)
		if found {
			hdrs[f] = fpos
		} else {
			return fmt.Errorf("Required field %s is not present in PMTA accounting headers", f)
		}
	}
	// Pick up positions of optional fields, for event augmentation
	for _, f := range optionalAcctFields {
		fpos, found := PositionIn(r, f)
		if found {
			hdrs[f] = fpos
		}
	}
	hdrsJSON, err := json.Marshal(hdrs)
	if err != nil {
		return err
	}
	_, err = client.Set(RedisAcctHeaders, hdrsJSON, 0).Result()
	if err != nil {
		return err
	}
	log.Println("Loaded", RedisAcctHeaders, "->", string(hdrsJSON), "into Redis")
	return nil
}

// Store a single accounting event r into redis, based on previously seen header format
func storeEvent(r []string, client *redis.Client) error {
	hdrsJ, err := client.Get(RedisAcctHeaders).Result()
	if err == redis.Nil {
		return fmt.Errorf("Error: redis key %v not found", RedisAcctHeaders)
	}
	hdrs := make(map[string]int)
	err = json.Unmarshal([]byte(hdrsJ), &hdrs)
	if err != nil {
		return err
	}
	// read fields into a message_id-specific redis key
	msgIDindex, ok := hdrs[msgIDField]
	if !ok {
		return fmt.Errorf("Error: redis key %v does not contain mapping for header_x-sp-message-id", RedisAcctHeaders)
	}
	msgIDKey := TrackingPrefix + r[msgIDindex]
	augment := make(map[string]string)
	for k, i := range hdrs {
		if k != msgIDField && k != typeField {
			augment[k] = r[i]
		}
	}
	// Set key message_id in Redis
	augmentJSON, err := json.Marshal(augment)
	if err != nil {
		return err
	}
	_, err = client.Set(msgIDKey, augmentJSON, MsgIDTTL).Result()
	if err != nil {
		return err
	}
	log.Printf("Loaded %s -> %s into Redis\n", msgIDKey, string(augmentJSON))
	return nil
}

// AccountETL extracts, transforms accounting data from PowerMTA into Redis records
func AccountETL(f *os.File) error {
	client := MyRedis()
	input := csv.NewScanner(bufio.NewReader(f))
	for input.Scan() {
		r := input.Record()
		if len(r) < len(requiredAcctFields) {
			return fmt.Errorf("Insufficient data fields %v", r)
		}
		switch r[0] {
		case deliveryType:
			err := storeEvent(r, client)
			if err != nil {
				return err
			}
		case typeField:
			err := storeHeaders(r, client)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Accounting record not of expected type: %v", r)
		}
	}
	return nil
}
