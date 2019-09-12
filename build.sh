#!/usr/bin/env bash
cd cmd/acct_etl; go build; cd ../..
cd cmd/feeder; go build; cd ../..
cd cmd/tracker; go build; cd ../..
cd cmd/wrapper; go build; cd ../..