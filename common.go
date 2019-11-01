package sparkypmtatracking

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-redis/redis"
)

// Check error status, log and try to continue
func Check(e error) {
	if e != nil {
		log.Println(e)
	}
}

// ConsoleAndLogFatal writes error to both log and stdout
func ConsoleAndLogFatal(s ...interface{}) {
	fmt.Println(s...)
	log.Fatalln(s...)
}

// MyLogger sets up a custom logger, if filename is given, emitting to stdout as well
// If filename is blank string, then output is stdout only
func MyLogger(filename string) {
	if filename != "" {
		logfile, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		Check(err)
		log.SetOutput(logfile)
	}
}

// GetenvDefault returns an environment variable, with default if unset
func GetenvDefault(k string, d string) string {
	x := os.Getenv(k)
	if x == "" {
		x = d
	}
	return x
}

//-----------------------------------------------------------------------------

// HostCleanup returns a SparkPost host address in canonical form (with schema, without /api/v1 path)
func HostCleanup(host string) string {
	if !strings.HasPrefix(host, "https://") {
		host = "https://" + host // Add schema
	}
	host = strings.TrimSuffix(host, "/")
	host = strings.TrimSuffix(host, "/api/v1")
	host = strings.TrimSuffix(host, "/")
	return host
}

//-----------------------------------------------------------------------------

// PositionIn returns the position of a value within an array of strings, and whether found or not
func PositionIn(arr []string, val string) (int, bool) {
	for i, v := range arr {
		if v == val {
			return i, true
		}
	}
	return 0, false
}

// Contains tells whether a contains x
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

//-----------------------------------------------------------------------------

// RedisQueue connects the tracker and feeder tasks
const RedisQueue = "trk_queue"

// RedisAcctHeaders holds the PowerMTA accounting file headers
const RedisAcctHeaders = "acct_headers"

// TrackingPrefix is the prefix for keys holding enrichment data
const TrackingPrefix = "msgID_"

// MyRedis returns a client handle for Redis, for server the standard port
func MyRedis() (client *redis.Client) {
	return redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
