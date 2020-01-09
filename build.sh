#!/usr/bin/env bash
go build -v ./cmd/acct_etl
go build -v ./cmd/feeder
go build -v ./cmd/tracker
go build -v ./cmd/wrapper
go build -v ./cmd/linktool