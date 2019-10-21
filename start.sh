#!/usr/bin/env bash
. setenvs.sh
~/go/src/github.com/sparkyPmtaTracking/src/tracker/tracker &
~/go/src/github.com/sparkyPmtaTracking/src/feeder/feeder &
~/go/src/github.com/sparkyPmtaTracking/src/feeder/wrapper -in_hostport :5587 -out_hostport pmta.signalsdemo.trymsys.net:587 -privkeyfile privkey.pem -certfile fullchain.pem -downstream_debug debug_downstream.log -upstream_data_debug debug_upstream.eml --insecure_skip_verify --verbose &
