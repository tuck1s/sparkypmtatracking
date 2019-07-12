#!/usr/bin/env bash
cd src/acct_etl; go build; cd ../..
cd src/feeder; go build; cd ../..
cd src/tracker; go build; cd ../..