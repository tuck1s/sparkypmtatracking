#!/usr/bin/env bash
. setenvs.sh
cmd/tracker/tracker &
cmd/feeder/feeder &
cmd/wrapper/wrapper -in_hostport :5587 -out_hostport pmta.signalsdemo.trymsys.net:587 \
 -privkeyfile privkey.pem -certfile fullchain.pem \
 -engagement_url http://smtp.proxy.trymsys.net \
 -logfile wrapper.log -verbose \
 -downstream_debug debug_downstream.log -upstream_data_debug debug_upstream.eml \
 -insecure_skip_verify
