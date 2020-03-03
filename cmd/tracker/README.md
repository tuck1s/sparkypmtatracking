# tracker
The tracker task runs a web service that decodes client email opens and clicks, provides http responses, and queues events for the feeder task.

```
 ./tracker -h
Web service that decodes client email opens and clicks
Runs in plain mode, it should proxied (e.g. by nginx) to provide https and protection.
Usage of ./tracker:
  -in_hostport string
        host:port to serve incoming HTTP requests (default ":8888")
  -logfile string
        File written with message logs
```

If you omit `-logfile`, output will go to the console (stdout).

The logfile records the action (open/click), target URL, datetime, user_agent, and remote (client) IP address:

```log
2020/01/09 15:40:27 Timestamp 1578584427, IPAddress 127.0.0.1, UserAgent Mozilla/5.0 (Linux; Android 4.4.2; XMP-6250 Build/HAWK) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/30.0.0.0 Safari/537.36 ADAPI/2.0 (UUID:9e7df0ed-2a5c-4a19-bec7-2cc54800f99d) RK3188-ADAPI/1.2.84.533 (MODEL:XMP-6250), Action c, URL http://example.com/index.html, MsgID 00006449175e39c767c2
2020/01/09 15:40:27 Timestamp 1578584427, IPAddress 127.0.0.1, UserAgent Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.157 Safari/537.36, Action o, URL , MsgID 00006449175eea2bd529
```

You can test your service endpoint locally using `curl` to a link address, such as 

```
curl -v http://localhost:8888/eJxUzLEOQiEMRuF3-WciGAaTTr4JwbaIUSKBMhnf_Ybxnv18P2Q2EBgOltb4gFDN-iTvraotfs8Lfxsc2nyml4AQdqJZHqqlhCDXGG9wGNw3VYbK_fT-jwAAAP__f2Mg1g==
```

You should see response headers such as
```
< HTTP/1.1 302 Found
< Content-Type: text/plain
< Location: https://thetucks.com
< Server: msys-http
```

You can make your own test link addresses using [linktool](#linktool).

### Tracker internals
The tracker web service receives URL requests with the path carrying base64-encoded (URL safe), Zlib-compressed, minified JSON.

Each event is augmented with:
- event type (open, initial_open, click)
- user agent
- timestamp (time of opening / clicking)
- client IP address

and sent to the Redis queue for the feeder task (using `RPUSH`).

It's usual to deploy a proxy such as `NGINX` in front of this service; more [here](#NGINX).