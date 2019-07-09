#!/usr/bin/env bash
. setenvs.sh
~/go/src/github.com/sparkyPmtaTracking/src/tracker/tracker &
~/go/src/github.com/sparkyPmtaTracking/src/feeder/feeder &