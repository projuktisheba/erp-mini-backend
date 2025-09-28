#!/bin/bash

rm -f app

go build -ldflags="-s -w" -o app

scp app samiul@192.250.228.113:/home/samiul/apps/bin/main-erp-mini-backend