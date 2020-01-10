package sparkypmtatracking

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"gopkg.in/natefinch/lumberjack.v2" // timed rotating log handler
)

// ConsoleAndLogFatal writes error to both log and stdout
func ConsoleAndLogFatal(s ...interface{}) {
	fmt.Println(s...)
	log.Fatalln(s...)
}

// MyLogger sets up a custom logger, if filename is given, emitting to stdout as well
// If filename is blank string, then output is stdout only
func MyLogger(filename string) {
	if filename != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename: filename,
			MaxAge:   7,    //days
			Compress: true, // disabled by default
		})
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

// PositionIn returns the position of a value within an array of strings, if an element exactly matches val
func PositionIn(arr []string, val string) (int, bool) {
	for i, v := range arr {
		if v == val {
			return i, true
		}
	}
	return 0, false
}

// Contains returns whether arr contains an element exactly matching val
func Contains(arr []string, val string) bool {
	_, found := PositionIn(arr, val)
	return found
}

// SafeStringToInt logs an error and returns zero if it can't convert
func SafeStringToInt(s string) int {
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Println("Warning: cannot convert", s, "to int")
		i = 0
	}
	return i
}

//-----------------------------------------------------------------------------

// RedisQueue connects the tracker and feeder tasks
const RedisQueue = "trk_queue"

// RedisAcctHeaders holds the PowerMTA accounting file headers
const RedisAcctHeaders = "acct_headers"

// TrackingPrefix is the prefix for keys holding augmentation data
const TrackingPrefix = "msgID_"

// MsgIDTTL defines the time-to-live for augmentation data
const MsgIDTTL = time.Duration(time.Hour * 24 * 10)

// MyRedis returns a client handle for Redis, for server the standard port
func MyRedis() (client *redis.Client) {
	return redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
