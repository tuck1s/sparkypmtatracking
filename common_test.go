package sparkypmtatracking_test

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

// Can't test ConsoleAndLogFatal() as it exits

func TestMyLogger(t *testing.T) {
	f := "common_test.log"
	spmta.MyLogger(f)
	s := errors.New("Something to log")
	log.Println(s)
}

func Test_GetenvDefault(t *testing.T) {
	vname := "some_weird_env_variable42"
	blank := "---"
	vval := "ta ta, a a value"
	v := spmta.GetenvDefault(vname, blank)
	if v != blank {
		t.Errorf("Unexpected value")
	}
	os.Setenv(vname, vval)
	v = spmta.GetenvDefault(vname, blank)
	if v != vval {
		t.Errorf("Unexpected value")
	}
	os.Unsetenv(vname)
}

func TestHostCleanup(t *testing.T) {
	u := "example.com"
	clean := "https://" + u
	if spmta.HostCleanup(u) != clean {
		t.Errorf("Unexpected value")
	}
	if spmta.HostCleanup(u+"/") != clean {
		t.Errorf("Unexpected value")
	}
	if spmta.HostCleanup(u+"/api/v1") != clean {
		t.Errorf("Unexpected value")
	}
	if spmta.HostCleanup(u+"/api/v1/") != clean {
		t.Errorf("Unexpected value")
	}
}

var haystack = []string{"the", "rain", "in", "spain", "falls", "mainly", "on", "the", "plain"}

const needle = "spain"
const miss = "snow"

func TestPositionIn(t *testing.T) {
	p, found := spmta.PositionIn(haystack, needle)
	if p != 3 || !found {
		t.Errorf("Unexpected value")
	}
	p, found = spmta.PositionIn([]string{}, needle)
	if p != 0 || found {
		t.Errorf("Unexpected value")
	}
	p, found = spmta.PositionIn([]string{}, miss)
	if p != 0 || found {
		t.Errorf("Unexpected value")
	}

}

func TestContains(t *testing.T) {
	found := spmta.Contains(haystack, needle)
	if !found {
		t.Errorf("Unexpected value")
	}
	found = spmta.Contains(haystack, miss)
	if found {
		t.Errorf("Unexpected value")
	}
}

func TestSafeStringToInt(t *testing.T) {
	if v := spmta.SafeStringToInt("0123456"); v != 123456 {
		t.Errorf(fmt.Sprintf("Unexpected value %d", v))
	}
	if v := spmta.SafeStringToInt(""); v != 0 {
		t.Errorf(fmt.Sprintf("Unexpected value %d", v))
	}
	if v := spmta.SafeStringToInt("     "); v != 0 {
		t.Errorf(fmt.Sprintf("Unexpected value %d", v))
	}
	if v := spmta.SafeStringToInt("-9876"); v != -9876 {
		t.Errorf(fmt.Sprintf("Unexpected value %d", v))
	}
	if v := spmta.SafeStringToInt("kittens"); v != 0 {
		t.Errorf(fmt.Sprintf("Unexpected value %d", v))
	}
}

func TestMyRedis(t *testing.T) {
	r := spmta.MyRedis()
	if r == nil {
		t.Errorf("Unexpected value")
	}
	r.Close()
}
