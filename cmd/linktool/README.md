# linktool
Command-line tool to encode and decode link URLs, useful during testing.

```
./linktool -h
./linktool [encode|decode] encode and decode link URLs

encode
  -action string
        [open|initial_open|click] (default "open")
  -message_id string
        message_id (default "0000123456789abcdef0")
  -rcpt_to string
        rcpt_to (default "any@example.com")
  -target_link_url string
        URL of your target link (default "https://example.com")
  -tracking_url string
        URL of your tracking service endpoint (default "http://localhost:8888")

decode url
```

Example: encode a URL
```
./linktool encode -tracking_url https://my-tracking-domain.com -rcpt_to fred@thetucks.com -action click -target_link_url https://thetucks.com -message_id 00000deadbeeff00d1337

https://my-tracking-domain.com/eJxUzLEOQiEMRuF3-WciGAaTTr4JwbaIUSKBMhnf_Ybxnv18P2Q2EBgOltb4gFDN-iTvraotfs8Lfxsc2nyml4AQdqJZHqqlhCDXGG9wGNw3VYbK_fT-jwAAAP__f2Mg1g==
```

Decode a URL
```
./linktool decode https://my-tracking-domain.com/eJxUzLEOQiEMRuF3-WciGAaTTr4JwbaIUSKBMhnf_Ybxnv18P2Q2EBgOltb4gFDN-iTvraotfs8Lfxsc2nyml4AQdqJZHqqlhCDXGG9wGNw3VYbK_fT-jwAAAP__f2Mg1g==

JSON: {"act":"c","t_url":"https://thetucks.com","msg_id":"00000deadbeeff00d1337","rcpt":"fred@thetucks.com"}
Equivalent to encode -tracking_url https://my-tracking-domain.com -rcpt_to fred@thetucks.com -action click -target_link_url https://thetucks.com -message_id 00000deadbeeff00d1337```
