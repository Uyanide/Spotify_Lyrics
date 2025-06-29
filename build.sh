#!/bin/env bash

if [ -f .env ]; then
    export $(cat .env | xargs)
    SP_DC=${SP_DC:-$SP_DC}
    REFETCH_INTERVAL=${REFETCH_INTERVAL:-$REFETCH_INTERVAL}
    MIN_LISTEN_INTERVAL=${MIN_LISTEN_INTERVAL:-$MIN_LISTEN_INTERVAL}
else
    echo ".env file not found."
    exit 1
fi

go build -ldflags "-s -w \
                   -X main.SP_DC=$SP_DC \
                   -X main.MinListenInterval=$MIN_LISTEN_INTERVAL" \
         -o spotify-lyrics