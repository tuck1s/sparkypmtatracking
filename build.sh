#!/usr/bin/env bash
go build ./cmd/acct_etl
go build ./cmd/feeder
go build ./cmd/tracker
go build ./cmd/wrapper