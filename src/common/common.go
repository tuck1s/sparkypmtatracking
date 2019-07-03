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
