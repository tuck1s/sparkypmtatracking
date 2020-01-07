<a href="https://www.sparkpost.com"><img alt="SparkPost, Inc." src="img/sp-pmta-logo.png" width="200px"/></a>

[Sign up](https://app.sparkpost.com/join?plan=free-0817?src=Social%20Media&sfdcid=70160000000pqBb&pc=GitHubSignUp&utm_source=github&utm_medium=social-media&utm_campaign=github&utm_content=sign-up) for a SparkPost account and visit our [Developer Hub](https://developers.sparkpost.com) for even more content.

# sparkyPmtaTracking
[![Build Status](https://travis-ci.org/tuck1s/sparkyPMTATracking.svg?branch=master)](https://travis-ci.org/tuck1s/sparkyPMTATracking)
[![Coverage Status](https://coveralls.io/repos/github/tuck1s/sparkyPMTATracking/badge.svg?branch=master)](https://coveralls.io/github/tuck1s/sparkyPMTATracking?branch=master)

Open and click tracking modules for PMTA and SparkPost Signals:

<img src="doc-img/open_click_tracking_for_signals_for_powermta.svg"/>

|app|description|
|---|---|
|`feeder`|Takes open & click events, correlates them and feeds them to the SparkPost Signals [Ingest API](https://developers.sparkpost.com/api/events-ingest/)|
|`wrapper`|SMTP proxy service that adds engagement information to email
|`acct_etl`|Extract, transform, load piped PMTA custom accounting stream into Redis|
|`tracker`|Web service that decodes client email opens and clicks|

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
go get gopkg.in/natefinch/lumberjack.v2
```

Run the `./build.sh` script included in the project, to build each app.

## feeder

The feeder task reads events from the Redis queue and feeds them to the SparkPost Ingest API.

```
./feeder -h
Takes the opens and clicks from the Redis queue and feeds them to the SparkPost Ingest API
Requires environment variable SPARKPOST_API_KEY_INGEST and optionally SPARKPOST_HOST_INGEST
Usage of ./feeder:
  -logfile string
        File written with message logs
```

If you omit-logfile, output will go to the console (stdout).
The SparkPost ingest API key (and optionally, the host base URL) is passed in environment variables:

```
export SPARKPOST_API_KEY_INGEST=###your API key here##
export SPARKPOST_HOST_INGEST=api.sparkpost.com
```

You’ll typically want to run this as a background process on startup - see the project cronfile and start.sh for examples of how to do that.

The logfile shows number of events, GZipped upload size, Ingest API response and Batch ID.

```
2020/01/07 16:00:44 Uploaded 82559 bytes raw, 4881 bytes gzipped. SparkPost Ingest response: 200 OK, results.id=deea5e3e-7e03-4b3c-831b-1b2851190db1
2020/01/07 16:10:41 Uploaded 84612 bytes raw, 5104 bytes gzipped. SparkPost Ingest response: 200 OK, results.id=a567ec74-c1e0-4546-86bd-dbd838315e71
2020/01/07 16:20:41 Uploaded 31974 bytes raw, 2265 bytes gzipped. SparkPost Ingest response: 200 OK, results.id=36e9b2d7-ea54-4fc5-8ed0-7f5696623464
``` 

## wrapper

SMTP proxy that accepts incoming messages from your downstream client, applies engagement-tracking (wrapping links and adding open tracking pixels) and relays on to an upstream server.

TLS with your own local certificate/private key is supported. Each phase of the SMTP conversation, including STARTTLS connection negotiation with the upstream server, proceeds in step with your client requests.

Usage is shown with `-h`, for example `cmd/wrapper/wrapper -h`
```
Usage of cmd/wrapper/wrapper:
  -certfile string
        Certificate file for this server
  -downstream_debug string
        File to write downstream server SMTP conversation for debugging
  -engagement_url string
        Engagement tracking URL used in html email body for opens and clicks
  -in_hostport string
        Port number to serve incoming SMTP requests (default "localhost:587")
  -insecure_skip_verify
        Skip check of peer cert on upstream side
  -out_hostport string
        host:port for onward routing of SMTP requests (default "smtp.sparkpostmail.com:587")
  -privkeyfile string
        Private key file for this server
  -upstream_data_debug string
        File to write upstream DATA for debugging
  -verbose
        print out lots of messages
```

Example:

```bash
cmd/wrapper/wrapper -in_hostport :5587 -out_hostport pmta.signalsdemo.trymsys.net:587 -privkeyfile privkey.pem -certfile fullchain.pem -downstream_debug debug_downstream.log -upstream_data_debug debug_upstream.eml --insecure_skip_verify
```

Localhost port 5587 now accepts incoming SMTP messages. You can now submit messages using e.g. `swaks`

```
swaks --server 127.0.0.1:5587 --auth-user SMTP_Injection --auth-pass ##your password here## --to bob.lumreeker@gmail.com --from proxytest@pmta.signalsdemo.trymsys.net --tls --data ../sparkySMTPProxy/test-emails/messenger-tracked.eml 
```

Startup messages are logged to `wrapper.log`, with a line written each time a message is processed.

```log
2019/10/21 20:12:10 Incoming host:port set to :5587
2019/10/21 20:12:10 Outgoing host:port set to pmta.signalsdemo.trymsys.net:587
2019/10/21 20:12:10 Proxy writing upstream DATA to debug_upstream.eml
2019/10/21 20:12:10 Engagement tracking URL: 
2019/10/21 20:12:10 insecure_skip_verify (Skip check of peer cert on upstream side): true
2019/10/21 20:12:10 Gathered certificate fullchain.pem and key privkey.pem
2019/10/21 20:12:10 Proxy will advertise itself as smtp.proxy.trymsys.net
2019/10/21 20:12:10 Verbose SMTP conversation logging: false
2019/10/21 20:12:10 Proxy logging SMTP commands, responses and downstream DATA to debug_downstream.log
```

In default (non-verbose) mode, the line `Message Data upstream` shows the upstream message size (bytes), upstream server SMTP response code and text.

```log
2019/10/21 20:12:16 Message DATA upstream,49496,250,2.6.0 message received
```

In verbose mode, logfile shows downstream and upstream SMTP conversation traces, in a similar manner to the progress messages shown by `swaks` 

```log
2019/10/21 21:40:21 ---Connecting upstream
2019/10/21 21:40:22 	<- Connection success pmta.signalsdemo.trymsys.net:587
2019/10/21 21:40:22 -> EHLO
2019/10/21 21:40:22 	<- EHLO success
2019/10/21 21:40:22 	Upstream capabilities: [8BITMIME AUTH CRAM-MD5 AUTH=CRAM-MD5 CHUNKING DSN ENHANCEDSTATUSCODES PIPELINING SIZE 0 SMTPUTF8 STARTTLS VERP XACK]
2019/10/21 21:40:22 -> STARTTLS
2019/10/21 21:40:22 	<~ 220 2.0.0 ready to start TLS
2019/10/21 21:40:22 ~> EHLO
2019/10/21 21:40:23 	<~ EHLO success
2019/10/21 21:40:23 	Upstream capabilities: [8BITMIME AUTH CRAM-MD5 PLAIN LOGIN AUTH=CRAM-MD5 PLAIN LOGIN CHUNKING DSN ENHANCEDSTATUSCODES PIPELINING SIZE 0 SMTPUTF8 VERP XACK]
2019/10/21 21:40:23 ~> AUTH PLAIN xyzzy=
2019/10/21 21:40:23 	<~ 235 2.7.0 authentication succeeded
2019/10/21 21:40:23 ~> MAIL FROM:<proxytest@pmta.signalsdemo.trymsys.net>
2019/10/21 21:40:23 	<~ 250 2.1.0 MAIL ok
2019/10/21 21:40:23 ~> RCPT TO:<bob.lumreeker@gmail.com>
2019/10/21 21:40:23 	<~ 250 2.1.5 <bob.lumreeker@gmail.com> ok
2019/10/21 21:40:23 ~> DATA
2019/10/21 21:40:24 	<~ DATA accepted, bytes written = 49496
2019/10/21 21:40:24 	<~ 250 2.6.0 message received
2019/10/21 21:40:24 ~> QUIT 
2019/10/21 21:40:25 	<~ 221 2.0.0 pmta.signalsdemo.trymsys.net says goodbye
```

### Interfaces to listen on

Note that `-in_hostport localhost:587` accepts traffic sources only from the local machine. To accept traffic on all network interfaces, use `-in_hostport 0.0.0.0:587`.

### STARTTLS and certificates

STARTTLS support for your downstream client requires:
- a pair of files, containing matching public certificate & private keys, for your  proxy domain, in `.pem` format;
- an upstream host that supports STARTTLS;
- specify these files using the `-privkeyfile` and `-certfile` command line flags.

The proxy simply passes SMTP options from the upstream server connection to the downstream client.
Your client, of course, can choose whether to proceed with a plain (insecure) connection or not.

If you have no certificates for your proxy domain, then omit the `-privkeyfile` and `-certfile` flags. 

### Upstream server certificate validity

The proxy TLS library checks validity of upstream certificates used with TLS.
If your upstream server has a self-signed, or otherwise invalid certificate, you'll see

```log
2019/10/21 21:37:03 	<~ EHLO error x509: certificate is valid for ip-172-31-25-101.us-west-2.compute.internal, localhost, not pmta.signalsdemo.trymsys.net
```
Proper solution: install a valid certificate on your upstream server.

Workaround: you can use the `-insecure_skip_verify` flag to make the proxy tolerant of your upstream server cert.

### downstream_debug

This option captures the entire conversation on the downstream (client) side, including SMTP cmmands and responses and the DATA phase containing message headers and body.

The file is created afresh each time the program is started (i.e. not appended to).
Use with caution as debug files can get large.

### upstream_data_debug

This option captures the DATA phase on the upstream (server) side, containing message headers and body. When engagement tracking is being used, the upstream content will be different to the downstream content as a header is added, links are tracked, and open pixels added.

The file is created afresh each time the program is started (i.e. not appended to).
Use with caution as debug files can get large.

### example email files

Some example `.eml` content can be used to send through the wrapper proxy with `swaks`:
 
```bash
swaks --server smtp.proxy.trymsys.net:587 --auth-user SMTP_Injection --auth-pass YOUR_KEY_HERE --to bob.lumreeker@gmail.com --from via_proxy@email.thetucks.com --data example2.eml --tls
``` 

---

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


test your build worked OK on example file. This should write log entries in your current dir.
```
./acct_etl --logfile acct_etl.log --infile example.csv
cat acct_etl.log
```

copy executable to a place where PMTA can run it, and set owner. Need to temporarily stop PMTA

```
sudo service pmta stop
sudo cp acct_etl /usr/local/bin/acct_etl
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

---

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

---


## Starting these applications on boot
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
