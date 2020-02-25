package sparkypmtatracking_test

import (
	"strings"
	"testing"

	spmta "github.com/tuck1s/sparkyPMTATracking"
)

const exampleCSV = `type,rcpt,header_x-sp-message-id,header_x-sp-subaccount-id
d,test+00102830@not-orange.fr.bouncy-sink.trymsys.net,0000123456789abcdef0,0
d,test+00113980@not-orange.fr.bouncy-sink.trymsys.net,0000123456789abcdef1,1
d,test+00183623@not-orange.fr.bouncy-sink.trymsys.net,0000123456789abcdef2,2
`

const validMinimalHeader = "type,header_x-sp-message-id\n"
const validHeaderWithRcpt = "type,rcpt,header_x-sp-message-id\n"

func loadCSV(csv string) error {
	f := strings.NewReader(csv)
	return spmta.AccountETL(f)
}

func loadCSVandCheckError(t *testing.T, csv string) {
	if err := loadCSV(csv); err != nil {
		t.Error(err)
	}
}

func TestAccountETL(t *testing.T) {
	loadCSVandCheckError(t, exampleCSV)
	loadCSVandCheckError(t, "\n")                            // empty file, should read OK
	loadCSVandCheckError(t, validMinimalHeader+"d,f00dbeef") // small input
}

// checks err contains a substring of an "expected" error
func checkExpectedError(t *testing.T, err error, s string) {
	if err == nil {
		t.Errorf("was expecting an error with %s, got %v", s, err)
	} else if !strings.Contains(err.Error(), s) {
		t.Error(err)
	}
}

func TestAccountETLFaultyInputs(t *testing.T) {
	// missing required header field
	err := loadCSV("type,rcpt\n" + "d,wilma@flintstone.org")
	checkExpectedError(t, err, "header_x-sp-message-id is not present")

	// incorrect record type
	err = loadCSV(validHeaderWithRcpt + "x,to@example.com,f00dbeef")
	checkExpectedError(t, err, "record not of expected type")

	// missing data
	err = loadCSV("type\n" + "d")
	checkExpectedError(t, err, "Insufficient data fields")
}

func TestAccountETLFaultyStoredHeader(t *testing.T) {
	// Load in a valid, minimal header, then delete it, which should cause the ETL of a data line to fail
	loadCSVandCheckError(t, validMinimalHeader)
	client := spmta.MyRedis()
	client.Del(spmta.RedisAcctHeaders)
	err := loadCSV("d,to@example.com,f00dbeef")
	checkExpectedError(t, err, "key acct_headers not found")

	// Load in a valid, minimal header ... then overwrite it, which will cause the ETL of a data line to fail
	loadCSVandCheckError(t, validMinimalHeader)
	client.Set(spmta.RedisAcctHeaders, `{"bananas":1,"type":0}`, 0)
	err = loadCSV("d,to@example.com,f00dbeef")
	checkExpectedError(t, err, "missing field header_x-sp-message-id")

	// Corrupt the header so it's not JSON
	client.Set(spmta.RedisAcctHeaders, `{gooseberry}`, 0)
	err = loadCSV("d,to@example.com,f00dbeef")
	checkExpectedError(t, err, "invalid character")
}

func TestAccountETLFaultyRedis(t *testing.T) {
	hdr := strings.Split(strings.TrimSuffix(validHeaderWithRcpt, "\n"), ",")
	client := spmta.MyRedis()
	err := client.Close() // Damage the Redis client connection
	err = spmta.StoreHeaders(hdr, client)
	checkExpectedError(t, err, "closed")
}
