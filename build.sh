#!/usr/bin/env bash
# flag -ldflags "-s -w" could be used to reduce size of binaries slightly
go build -v ./cmd/acct_etl
go build -v ./cmd/feeder
go build -v ./cmd/tracker
go build -v ./cmd/wrapper
go build -v ./cmd/linktool
