#!/bin/env bash

if [ -f .env ]; then
    export $(cat .env | xargs)
    API_URL=${SPOTIFY_API_URL:-$SPOTIFY_API_URL}
    REFETCH_INTERVAL=${REFETCH_INTERVAL:-$REFETCH_INTERVAL}
    MIN_LISTEN_INTERVAL=${MIN_LISTEN_INTERVAL:-$MIN_LISTEN_INTERVAL}
else
    echo ".env file not found."
    exit 1
fi

go build -ldflags "-s -w \
                   -X main.APIUrl=$API_URL \
                   -X main.RefetchInterval=$REFETCH_INTERVAL \
                   -X main.MinListenInterval=$MIN_LISTEN_INTERVAL" \
         -o spotify-lyrics