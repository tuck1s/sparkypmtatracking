# sparkyPmtaTracking
Open and click tracking modules for PMTA and SparkPost Signals:

|app|description|
|---|---|
|`acct_etl`|Extract, transform, load piped PMTA custom accounting stream into Redis|
|`tracker`|Web service that decodes client email opens and clicks|
|`wrapper`|SMTP proxy service that wraps html links in messages and adds `X-Tracking-Id` header|
|`feeder`|Takes open & click events, correlates them and feeds them to the SparkPost Signals [Ingest API](https://developers.sparkpost.com/api/events-ingest/)|

## Pre-requisites
- Redis - installation tips [here](#installing-redis-on-your-host)
- Git
- Golang

Check you have `redis` installed and working. This project assumes it is available 
on the default port `6379` on your host.

```
redis-cli --version
```
you should see `redis-cli 5.0.5` or similar
```
redis-cli PING
```
you should see `PONG`


Get this project with `git clone`, and dependencies with `go get`.

```
git clone https://github.com/tuck1s/sparkyPmtaTracking.git
cd sparkyPmtaTracking/
go get github.com/go-redis/redis
go get github.com/smartystreets/scanners/csv
```

Installation instructions follow, for each app.


## acct_etl
Extract, transform and load accounting data fed by [PMTA pipe](https://download.port25.com/files/UsersGuide.html#examples) 
into Redis.

PMTA config needs to have the following accounting pipe:
```
# Pipe into engagement tracking correlator. Expects a custom header in the injected mail also. 
<acct-file |/usr/local/bin/acct_etl>
    records d
    record-fields d orig,rcpt,header_x-sp-message-id,header_x-tracking-id
</acct-file>
```

Build, test this app and hook it into PMTA.
```
cd acct_etl/
go build

# test your build worked OK on example file. This should show some log entries.
./acct_etl example.csv
tail /opt/pmta/acct_etl.log

# copy executable to a place where PMTA can run it. Need to temporarily stop PMTA
sudo service pmta stop
sudo cp ./acct_etl /usr/local/bin/acct_etl
sudo chown pmta:pmta /usr/local/bin/acct_etl
sudo service pmta start
```

Check your PMTA log, there should not be startup errors.

The app logs into `/opt/pmta/acct_etl.log`, you should see
```
2019/07/02 17:26:15 PMTA accounting headers from pipe[type orig rcpt header_x-sp-message-id header_x-tracking-id]
2019/07/02 17:26:15 as expected by this application
```

Present some traffic to PMTA. The injected message should include a custom header `X-Tracking-Id`.
The above logfile should show entries such as
```
2019/07/02 17:30:04 Loaded 73277140a64645b0adee046cc7250e1f = F8AD9C941B5DBCA6B208 into Redis, from= test@pmta.signalsdemo.trymsys.net RcptTo= test+00073442@not-gmail.com.bouncy-sink.trymsys.net
2019/07/02 17:30:04 Loaded 2a889abbbea34370b9a85c902eb5b031 = 697C9C941B5D03CFA658 into Redis, from= test@pmta.signalsdemo.trymsys.net RcptTo= test+00179890@not-yahoo.co.uk.bouncy-sink.trymsys.net
```

Redis entries comprising key/value pairs of `(x-tracking-id, x-sp-meessage-id)` are set,
with configured time-to-live of 10 days (matching SparkPost's event retention).
You can list these keys with
```
redis-cli keys "trk_*"
```

## tracker
The tracker web service expects to receive URL requests of the form

```
http://pmta.signalsdemo.trymsys.net/tracking/open/eJxdT81uwyAMfpWI6xqSbolYeuoDrKc9AHKIoawBInCqRlXffVBt0jT5Ytnf750pcAtY46Wd2KFiH5CoOlm_ElafcLXeJLarWFQLSQoFQZjopW3bYS-G_ugD1SGCN8h15GNYvdrqZP2FU9xc2hL3SEXBJSN1DO5X4pgIr0h12f9jn24OCTL4_siHtI5fqKhw_4QiiAZJztlNrnEu3zPRcmgavOVWM3IVXGP9hDd-Jjc_ORHUJQv81N0LGF_LdAI6_T5B1-Nbr4dOTEKrHtnjG0smYWM=
```

These strings are base64-encoded (URL safe), Zlib-compressed, minified JSON. The code strips
these layers off to reveal the underlying minified JSON bytestring, i.e.

```
{"campaign_id": "Last Minute Savings", "rcpt_to": "test+00091795@not-orange.fr.bouncy-sink.trymsys.net", "msg_from": "test@stevet-test.trymsys.net", "rcpt_meta": {}, "subject": "Savings", "target_link_url": "http://example.com/index.html", "tracking_id": "17ab2b2b247a4f8da45e35f947d7fc5e"}
```

The bytestring data is pushed into a Redis queue for the feeder.

## feeder

TODO
## wrapper

TODO

---
### Installing Redis on your host

Redis does not currently seem to be available via a package manager on EC2 Linux.

Follow the QuickStart guide [here](https://redis.io/topics/quickstart), following the "Installing Redis" steps
and "Installing Redis more properly" steps. EC2 Linux does not have the
update-rc.d command, use `sudo chkconfig redis_6379 on` instead.

Here's the steps I followed:
```
# Building
wget http://download.redis.io/redis-stable.tar.gz
tar xvzf redis-stable.tar.gz
cd redis-stable
make
sudo make install

# install "properly" as a service
sudo vi /etc/redis/6379.conf
sudo cp utils/redis_init_script /etc/init.d/redis_6379
sudo vi /etc/init.d/redis_6379
sudo cp redis.conf /etc/redis/6379.conf
sudo mkdir /var/redis/6379
sudo vim /etc/redis/6379.conf

sudo chkconfig redis_6379 on
sudo /etc/init.d/redis_6379 start

# test
redis-cli ping
```

### Installing Git, Golang on your host
Your package manager should install these for you, e.g.
```
sudo yum install git go
``` 