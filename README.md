# sparkyPmtaTracking
Open and click tracking modules for PMTA and SparkPost Signals:

|app|description|
|---|---|
|`acct_etl`|Extract, transform, load piped PMTA custom accounting stream into Redis|
|`tracker`|Web service that decodes client email opens and clicks|
|`wrapper`|SMTP proxy service that adds engagement information to email
|`feeder`|Takes open & click events, correlates them and feeds them to the SparkPost Signals [Ingest API](https://developers.sparkpost.com/api/events-ingest/)|

## Pre-requisites
- Git & Golang - installation tips [here](#installing-git-golang-on-your-host)
- Redis - installation tips [here](#installing-redis-on-your-host)
- nginx - installation tips [here](#installing-and-configuring-nginx-proxy)

Get this project with `git clone`, and dependencies with `go get`.

```
# Let's assume your GOPATH is the usual, i.e.
export GOPATH=$HOME/go

cd ~/go/src/github.com/
git clone https://github.com/tuck1s/sparkyPmtaTracking.git
cd sparkyPmtaTracking/

# Get needed go packages
go get github.com/go-redis/redis
go get github.com/smartystreets/scanners/csv
go get github.com/google/uuid
```

Installation instructions follow, for each app.

## acct_etl
Extracts, transforms and loads accounting data fed by [PMTA pipe](https://download.port25.com/files/UsersGuide.html#examples) 
into Redis.

PMTA config needs to have the following accounting pipe. An example config file is [shown here](etc/pmta/config.example). The accounting pipe carries message attributes that are used to enrich the open and click event data:

|PMTA term|SparkPost term|
|--|--|
|x-sp-message-id|Message ID|
|orig|Sender (`From:` address)|
|rcpt|Recipient (`To:` address)|
|jobId|Campaign ID (name)|
|dlvSourceIp|Sending IP (e.g. `10.0.0.1`)|
|vmtaPool|IP Pool (name)|


Build, test this app and hook it into PMTA.
```
cd ~/go/src/github.com/sparkyPmtaTracking/src/acct_etl
go build
cd ../..

# test your build worked OK on example file. This should write log entries in your current dir.
src/acct_etl/acct_etl example.csv
cat acct_etl.log

# copy executable to a place where PMTA can run it, and set owner. Need to temporarily stop PMTA
sudo service pmta stop
sudo cp src/acct_etl/acct_etl /usr/local/bin/acct_etl
sudo chown pmta:pmta /usr/local/bin/acct_etl
sudo service pmta start
```
Check your PMTA log, there should not be startup errors.

Logfile `/opt/pmta/acct_etl.log` should show
```
2019/07/02 17:26:15 PMTA accounting headers from pipe[type orig rcpt header_x-sp-message-id header_x-tracking-id]
2019/07/02 17:26:15 as expected by this application
```

Present some traffic to PMTA, including the `x-sp-message-id` header.
The logfile should show entries such as
```
2019/07/02 17:30:04 Loaded 73277140a64645b0adee046cc7250e1f = F8AD9C941B5DBCA6B208 into Redis, from= test@pmta.signalsdemo.trymsys.net RcptTo= test+00073442@not-gmail.com.bouncy-sink.trymsys.net
2019/07/02 17:30:04 Loaded 2a889abbbea34370b9a85c902eb5b031 = 697C9C941B5D03CFA658 into Redis, from= test@pmta.signalsdemo.trymsys.net RcptTo= test+00179890@not-yahoo.co.uk.bouncy-sink.trymsys.net
```

Redis entries (key/value pairs) hold the enrichment data for each message ID, with configured time-to-live of 10 days (matching SparkPost's event retention).
You can list these keys with
```
redis-cli keys msgID*
```

## tracker

To install:

```
cd ~/go/src/github.com/sparkyPmtaTracking/src/tracker
go build
cd ../..
```
To test, run from the command line. This will listen on port 8888 for incoming requests.
```
src/tracker/tracker &
```

Test using the following example

```
curl -v localhost:8888/tracking/open/eJxdT81uwyAMfpWI6xqSbolYeuoDrKc9AHKIoawBInCqRlXffVBt0jT5Ytnf750pcAtY46Wd2KFiH5CoOlm_ElafcLXeJLarWFQLSQoFQZjopW3bYS-G_ugD1SGCN8h15GNYvdrqZP2FU9xc2hL3SEXBJSN1DO5X4pgIr0h12f9jn24OCTL4_siHtI5fqKhw_4QiiAZJztlNrnEu3zPRcmgavOVWM3IVXGP9hDd-Jjc_ORHUJQv81N0LGF_LdAI6_T5B1-Nbr4dOTEKrHtnjG0smYWM=
```

Local logfile `tracker.log` displays the action (open/click), target URL, datetime, user_agent, and remote (client) IP address:

```
2019/08/19 23:06:32 {open http://example.com/index.html  1566252392 curl/7.54.0 ::1}
``` 

### Internals
The tracker web service receives URL requests with the path carrying base64-encoded (URL safe), Zlib-compressed, minified JSON.
Each event is augmented with
- event type (open, click)
- user agent
- timestamp (time of opening / clicking)

and pushed into a Redis queue for the feeder task (using `RPUSH`).

It's usual to deploy a proxy such as `nginx` in front of this service; an example nginx config is given [here](etc/nginx/conf.d/server1.conf).

## feeder

The feeder task reads events from the Redis queue and feeds them to the
SparkPost Ingest API.

Check local logfile with `feeder.log`. The number of events, GZipped upload
size, Ingest API response and Batch ID are logged.

```
2019/07/25 14:52:05 Uploaded 1 events 483 bytes (gzip), SparkPost Ingest response: 200 OK results.id= 6984b656-034e-40d8-89ab-f4dd33a41d49
``` 

## wrapper

TODO

### Starting these applications on boot
Script `start.sh` is provided for this purpose. You can make it run on boot using
```
crontab cronfile
```
or `crontab -e` then paste in cronfile contents.


---

# Pre-requisites installation

### Installing Git, Golang on your host
Your package manager should install these for you, e.g.
```
sudo yum install git go
``` 

### Installing Redis on your host

Redis does not currently seem to be available via a package manager on EC2 Linux.

Follow the QuickStart guide [here](https://redis.io/topics/quickstart), following the "Installing Redis" steps
and "Installing Redis more properly" steps. EC2 Linux does not have the
update-rc.d command, use `sudo chkconfig redis_6379 on` instead.

Here's the detailed steps. This project assumes port `6379` on your host.                           

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
```

Check you now have `redis` installed and working.
```
redis-cli --version
```
you should see `redis-cli 5.0.5` or similar
```
redis-cli PING
```
you should see `PONG`


### Installing and configuring nginx proxy
This provides protection for your application server.

```
sudo yum install nginx
sudo vim /etc/nginx/conf.d/server1.conf
```
Paste in the contents of `server1.conf` from this project, and modify to suit your server address and environment.

If you wish to use port 80 for tracking:
- Check the main config file `/etc/nginx/nginx.conf` is not serving ordinary files by default.
- If it is, you may need to change or remove the existing `server { .. }` stanza. Then
```
sudo service nginx start
```

Check the endpoint is active from another host, using `curl` as above, but using your external host address and port number.

Ensure nginx starts on boot:
```
sudo chkconfig nginx on
```
