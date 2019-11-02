#!/usr/bin/env bash
. setenvs.sh

./tracker -logfile tracker.log &

./feeder -logfile feeder.log &

# needs to be started as root if in_hostport is in range 1..1024
sudo ./wrapper -in_hostport :587 -out_hostport :5587 \
 -privkeyfile trymsys.net.key -certfile trymsys.net.crt \
 -engagement_url http://pmta.signalsdemo.trymsys.net \
 -logfile wrapper.log \
 -insecure_skip_verify &

 # -verbose \
 # -downstream_debug debug_downstream.log -upstream_data_debug debug_upstream.eml \

# acct_etl is run directly by PowerMTA - refer to README.md for how to set this up