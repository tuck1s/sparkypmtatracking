package sparkypmtatracking_test

import (
	"os"
	"strings"
	"testing"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

const exampleCSV = "example.csv"

func TestAccountETL(t *testing.T) {
	f, err := os.Open(exampleCSV)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	err = spmta.AccountETL(f)
	if err != nil {
		t.Errorf("Error %v", err)
	}

	// empty file, reads OK
	f2 := strings.NewReader("\n")
	err = spmta.AccountETL(f2)
	if err != nil {
		t.Errorf("Error %v", err)
	}

	// minimalist input
	f4 := strings.NewReader("type,header_x-sp-message-id\n" +
		"d,f00dbeef")
	err = spmta.AccountETL(f4)
	if err != nil {
		t.Errorf("Error %v", err)
	}
}

// missing mandatory headers
func TestAccountETLFaultyInputs(t *testing.T) {
	// missing required header field
	f := strings.NewReader("type,rcpt\n" +
		"d,test+00102830@not-orange.fr.bouncy-sink.trymsys.net")
	err := spmta.AccountETL(f)
	if err.Error() != "Required field header_x-sp-message-id is not present in PMTA accounting headers" {
		t.Errorf("Error %v", err)
	}

	// incorrect record type
	f3 := strings.NewReader("type,rcpt,header_x-sp-message-id\n" +
		"x,to@example.com,f00dbeef")
	err = spmta.AccountETL(f3)
	if err.Error() != "Accounting record not of expected type: [x to@example.com f00dbeef]" {
		t.Errorf("Error %v", err)
	}

	// missing data
	f5 := strings.NewReader("type\n" +
		"d")
	err = spmta.AccountETL(f5)
	if err.Error() != "Insufficient data fields [type]" {
		t.Errorf("Error %v", err)
	}
}

func TestAccountETLFaultyStoredHeader(t *testing.T) {
	// Load just the header ...
	f6 := strings.NewReader("type,header_x-sp-message-id\n")
	err := spmta.AccountETL(f6)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	// Then delete it
	client := spmta.MyRedis()
	client.Del(spmta.RedisAcctHeaders)

	// which will cause the ETL of a data line to fail
	f7 := strings.NewReader("d,to@example.com,f00dbeef")
	err = spmta.AccountETL(f7)
	if err.Error() != "Redis key acct_headers not found" {
		t.Errorf("Error %v", err)
	}

	// Load in a valid header ...
	f8 := strings.NewReader("type,header_x-sp-message-id\n")
	err = spmta.AccountETL(f8)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	// Then overwrite it
	client.Set(spmta.RedisAcctHeaders, `{"bananas":1,"type":0}`, 0)

	// which will cause the ETL of a data line to fail
	f9 := strings.NewReader("d,to@example.com,f00dbeef")
	err = spmta.AccountETL(f9)
	if err.Error() != "Redis key acct_headers is missing field header_x-sp-message-id" {
		t.Errorf("Error %v", err)
	}

}
