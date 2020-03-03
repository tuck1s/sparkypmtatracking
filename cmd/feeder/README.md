# feeder
The feeder task reads events from the Redis queue in an internal format, and feeds them to the SparkPost Ingest API, with additional attributes from the local database where found.

```
./feeder -h
Takes the opens and clicks from the Redis queue and feeds them to the SparkPost Ingest API
Requires environment variable SPARKPOST_API_KEY_INGEST and optionally SPARKPOST_HOST_INGEST
Usage of ./feeder:
  -logfile string
        File written with message logs
```

If you omit `-logfile`, output will go to the console (stdout).
The SparkPost ingest API key (and optionally, the host base URL) is passed in environment variables:

```
export SPARKPOST_API_KEY_INGEST=###your API key here##
export SPARKPOST_HOST_INGEST=api.sparkpost.com
```

You’ll typically want to run this as a background process on startup - see the project cronfile and [start.sh](../../start.sh) for examples of how to do that.

The logfile shows number of events, GZipped upload size, Ingest API response and Batch ID.

```
2020/01/07 16:00:44 Uploaded 82559 bytes raw, 4881 bytes gzipped. SparkPost Ingest response: 200 OK, results.id=deea5e3e-7e03-4b3c-831b-1b2851190db1
2020/01/07 16:10:41 Uploaded 84612 bytes raw, 5104 bytes gzipped. SparkPost Ingest response: 200 OK, results.id=a567ec74-c1e0-4546-86bd-dbd838315e71
2020/01/07 16:20:41 Uploaded 31974 bytes raw, 2265 bytes gzipped. SparkPost Ingest response: 200 OK, results.id=36e9b2d7-ea54-4fc5-8ed0-7f5696623464
```