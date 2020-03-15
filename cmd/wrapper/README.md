# wrapper
SMTP proxy service that accepts incoming messages from your downstream client, applies engagement-tracking (wrapping links and adding open tracking pixels) and relays on to an upstream server.

TLS with your own local certificate/private key is supported.

```
SMTP proxy that accepts incoming messages from your downstream client, applies engagement-tracking
(wrapping links and adding open tracking pixels) and relays on to an upstream server.
Usage of ./wrapper:
  -certfile string
    	Certificate file for this server
  -downstream_debug string
    	File to write downstream server SMTP conversation for debugging
  -in_hostport string
    	Port number to serve incoming SMTP requests (default "localhost:587")
  -insecure_skip_verify
    	Skip check of peer cert on upstream side
  -logfile string
    	File written with message logs (also to stdout)
  -out_hostport string
    	host:port for onward routing of SMTP requests (default "smtp.sparkpostmail.com:587")
  -privkeyfile string
    	Private key file for this server
  -track_click
    	Wrap links in HTML mail, to track clicks
  -track_initial_open
    	Insert an initial_open tracking pixel at top of HTML mail
  -track_open
    	Insert an open tracking pixel at bottom of HTML mail (default true)
  -tracking_url string
    	URL of your tracking service endpoint (default "http://localhost:8888")
  -upstream_data_debug string
    	File to write upstream DATA for debugging
  -verbose
    	print out lots of messages
```

Example setup that will:
- Receive connections from downstream clients on port 5587 (IP version agnostic, i.e. incoming connections using IPv4 or IPv6 will be accepted, if your host supports it).
- Forward emails to an upstream server on port 587
- Offer its own TLS using the supplied certificates
- Log activity to a file
- All tracking options enabled.

```bash
./wrapper -in_hostport :5587 -out_hostport pmta.signalsdemo.trymsys.net:587 \
 -privkeyfile privkey.pem -certfile fullchain.pem \
 -logfile wrapper.log \
 -tracking_url http://pmta.signalsdemo.trymsys.net \
 -track_open -track_initial_open -track_click
```

Each phase of the SMTP conversation, including STARTTLS connection negotiation with the upstream server, proceeds in step with your downstream client requests.

```
Client      Proxy       Server
       ->
                   ->
                   <-
       <-
```

Response codes from the upstream server are echoed back to the downstream client as transparently as possible.

Another example (using `sudo` to serve reserved ports below 1024) is in [start.sh](start.sh).

On startup, a brief message is written to `stdout`.
```
Starting smtp proxy service on port :5587 , logging to wrapper.log
```

If you omit `-logfile`,  log output is written to `stdout`.

Example message submission using `swaks`:

```
swaks --server localhost:5587 --auth-user ##YOUR_USER_HERE## --auth-pass ##YOUR_PASSWORD_HERE## --to bob@example.com --from proxytest@yourdomain.com --tls
```

You'll see client output that ends in something like:
```
<~  250 2.6.0 message received
 ~> QUIT
<~  221 2.0.0 pmta.signalsdemo.trymsys.net says goodbye
=== Connection closed with remote host.
```

Details are logged to `wrapper.log`:
```log
2020/02/25 18:22:03 Starting smtp proxy service on port :5587
2020/02/25 18:22:03 Outgoing host:port set to pmta.signalsdemo.trymsys.net:587
2020/02/25 18:22:03 Engagement tracking URL: http://pmta.signalsdemo.trymsys.net, track_open true, track_initial_open true, track_click true
2020/02/25 18:22:03 Gathered certificate fullchain.pem and key privkey.pem
2020/02/25 18:22:03 Proxy will advertise itself as smtp.proxy.trymsys.net
2020/02/25 18:22:03 Verbose SMTP conversation logging: false
2020/02/25 18:22:03 insecure_skip_verify (Skip check of peer cert on upstream side): false
2020/02/25 18:44:53 Message DATA upstream,328,250,2.6.0 message received
```

In default (non-verbose) mode, the `Message DATA upstream` log line shows the message size delivered to the upstream server (bytes), upstream server SMTP response code, and text. 

### Authentication
The proxy passes the authentication methods and credentials through, between your upstream server and client; the proxy does not check your client's credentials. I have tested passthrough of `AUTH LOGIN`, `AUTH PLAIN` and `AUTH CRAM-MD5`.

### verbose
In verbose mode, your logfile shows the proxy downstream and upstream SMTP conversation sides, in a similar manner to the progress messages shown by `swaks` client. This is useful during setup and testing.

```log
2020/02/25 18:55:04 ---Connecting upstream
2020/02/25 18:55:04 < Connection success pmta.signalsdemo.trymsys.net:587
2020/02/25 18:55:04 -> EHLO
2020/02/25 18:55:04 	<- EHLO success
2020/02/25 18:55:04 	Upstream capabilities: [8BITMIME AUTH CRAM-MD5 AUTH=CRAM-MD5 CHUNKING DSN ENHANCEDSTATUSCODES PIPELINING SIZE 0 SMTPUTF8 STARTTLS VERP XACK XMRG]
2020/02/25 18:55:04 -> STARTTLS
2020/02/25 18:55:05 	<~ 220 2.0.0 ready to start TLS
2020/02/25 18:55:05 ~> EHLO
2020/02/25 18:55:05 	<~ EHLO success
2020/02/25 18:55:05 	Upstream capabilities: [8BITMIME AUTH CRAM-MD5 PLAIN LOGIN AUTH=CRAM-MD5 PLAIN LOGIN CHUNKING DSN ENHANCEDSTATUSCODES PIPELINING SIZE 0 SMTPUTF8 VERP XACK XMRG]
2020/02/25 18:55:05 ~> AUTH CRAM-MD5
2020/02/25 18:55:05 	<~ 334 ##REDACTED##
2020/02/25 18:55:05 ~> ##REDACTED## 
2020/02/25 18:55:05 	<~ 235 2.7.0 authentication succeeded
2020/02/25 18:55:05 ~> MAIL FROM:<test@example.com>
2020/02/25 18:55:05 	<~ 250 2.1.0 MAIL ok
2020/02/25 18:55:05 ~> RCPT TO:<test@bouncy-sink.trymsys.net>
2020/02/25 18:55:06 	<~ 250 2.1.5 <test@bouncy-sink.trymsys.net> ok
2020/02/25 18:55:06 ~> DATA
2020/02/25 18:55:06 	<~ DATA accepted, bytes written = 328
2020/02/25 18:55:06 ~> QUIT 
2020/02/25 18:55:06 	<~ 221 2.0.0 pmta.signalsdemo.trymsys.net says goodbye
```

### STARTTLS and certificates
STARTTLS requires:
- A pair of files, containing matching public certificate & private keys, for your proxy domain, in [.pem](https://en.wikipedia.org/wiki/Privacy-Enhanced_Mail) format. [LetsEncrypt](https://letsencrypt.org/) is a possible source for these;
- An upstream host that supports STARTTLS;
- A downstream client that will negotiate STARTTLS when offered.

Specify these files using the `-privkeyfile` and `-certfile` command line flags.

The proxy passes SMTP options from the upstream server connection to the downstream client.
Your client, of course, can choose whether to proceed with a plain (insecure) connection or not.

If you have no certificates for your proxy domain, then omit the `-privkeyfile` and `-certfile` flags.

### Upstream server certificate validity
The proxy checks validity of upstream certificates used with TLS.
If your upstream server has a self-signed, or otherwise invalid certificate, you'll see an error such as:

```log
2019/10/21 21:37:03 	<~ EHLO error x509: certificate is valid for ip-172-31-25-101.us-west-2.compute.internal, localhost, not pmta.signalsdemo.trymsys.net
```

You can either install a valid certificate on your upstream server (preferred!) or use the proxy `-insecure_skip_verify` flag to make the proxy tolerant of your invalid upstream server cert.

### Choosing your listener interface
Note that `-in_hostport localhost:x` accepts traffic sources only from your local machine. To listen for traffic on all your network interfaces on port x, use `-in_hostport 0.0.0.0:x`.

### downstream_debug
This option captures the conversation on the downstream (client) side, including SMTP cmmands and responses and the DATA phase containing message headers and body.

The file is created afresh each time the program is started (i.e. not appended to). Use with caution as debug files can get large.

### upstream_data_debug
This option captures the DATA phase on the upstream (server) side, containing message headers and body. When engagement tracking is being used, the upstream content will be different to the downstream content as a header is added, links are tracked, and open pixels added.

The file is created afresh each time the program is started (i.e. not appended to). A file containing a single test message captured in this way (e.g. ` -upstream_data_debug debug_up.eml`) is RFC822 compliant and can be viewed directly in a mail client.

Use with caution as debug files can get large.

### track_click, track_initial_open, track_open
The `-track_open` flag defaults to `true`, as the whole point of this project is to provide _some_ engagement tracking. The others default `false`.

If you wish to disable `track_open`, , use the `--track_open=false` form, as per usual [Go flags](https://golang.org/pkg/flag/#hdr-Command_line_flag_syntax) syntax.

### example email files
The project includes an [example file](../../example.eml) you can send with `swaks`. Adjust the `From:` and `To:` address to suit your configuration.

