# acct_etl
Extracts, transforms and loads accounting data fed by [PowerMTA pipe](https://download.port25.com/files/UsersGuide.html#examples) into Redis.

```
 ./acct_etl -h
Extracts, transforms and loads accounting data fed by PowerMTA pipe into Redis
Usage of ./acct_etl:
  -infile string
        Input file (omit to read from stdin)
  -logfile string
        File written with message logs
```

Here is an example [PowerMTA config file](../../etc/pmta/config.example) showing "accounting pipe" setup. The pipe carries message attributes that "feeder" uses to augment the open and click event data.

|PowerMTA accounting file config|SparkPost term / project usage|
|--|--|
|type|d=delivery|
|rcpt|Recipient (`To:` address)|
|header_x-sp-message-id|Message ID (added by `wrapper`)|
|header_x-sp-subaccount-id|Optional subaccount ID. Place in injected message if you wish to use|

### acct_etl internals
You can test without PowerMTA using the included example file:
```
./acct_etl -infile example.csv
```

```log
Starting acct_etl, logging to
2020/01/10 19:02:32 PowerMTA accounting headers: [type rcpt header_x-sp-message-id header_x-sp-subaccount-id]
2020/01/10 19:02:32 Loaded acct_headers -> {"header_x-sp-message-id":2,"header_x-sp-subaccount-id":3,"rcpt":1,"type":0} into Redis
2020/01/10 19:02:32 Loaded msgID_0000123456789abcdef0 -> {"header_x-sp-subaccount-id":"0","rcpt":"test+00102830@not-orange.fr.bouncy-sink.trymsys.net"} into Redis
2020/01/10 19:02:32 Loaded msgID_0000123456789abcdef1 -> {"header_x-sp-subaccount-id":"1","rcpt":"test+00113980@not-orange.fr.bouncy-sink.trymsys.net"} into Redis
2020/01/10 19:02:32 Loaded msgID_0000123456789abcdef2 -> {"header_x-sp-subaccount-id":"2","rcpt":"test+00183623@not-orange.fr.bouncy-sink.trymsys.net"} into Redis
```

The `start.sh` file copies the `acct_etl` executable to a place where PowerMTA runs it, and sets owner. It temporarily stops and restarts PowerMTA.

Logfile default location for this process is `/opt/pmta/acct_etl.log`.

Redis key/value pairs hold data for each message ID, with a configured time-to-live (matching SparkPost's event retention).
You can list these keys with `redis-cli keys msgID*`.
