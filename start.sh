#!/usr/bin/env bash
# Customise the following file to set up your environment vars 
. setenvs.sh

./tracker -logfile tracker.log &

./feeder -logfile feeder.log &

# needs to be started as root if in_hostport is in range 1..1024
sudo ./wrapper -in_hostport :587 -out_hostport :5587 \
 -privkeyfile trymsys.net.key -certfile trymsys.net.crt \
 -logfile wrapper.log \
 -insecure_skip_verify \
 -tracking_url http://pmta.signalsdemo.trymsys.net \
 -track_open -track_initial_open -track_click &

 # -verbose \
 # -downstream_debug debug_downstream.log -upstream_data_debug debug_upstream.eml \

# acct_etl is run directly by PowerMTA - refer to README.md for how to set this up
sudo service pmta stop
sudo cp acct_etl /usr/local/bin/acct_etl
sudo chown pmta:pmta /usr/local/bin/acct_etl
sudo service pmta start