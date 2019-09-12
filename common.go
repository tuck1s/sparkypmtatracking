package sparkypmtatracking

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-redis/redis"
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
// Redis
const RedisQueue = "trk_queue"          // Name of the queue between tracker and feeder tasks
const RedisAcctHeaders = "acct_headers" // Key that holds the PowerMTA accounting file headers
const TrackingPrefix = "msgID_"         // Keys beginning with this prefix hold enrichment data

// return a client handle for Redis. Assume server is on the standard port
func MyRedis() (client *redis.Client) {
	return redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
