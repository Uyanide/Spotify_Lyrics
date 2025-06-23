#!/bin/env bash

if [ -f .env ]; then
    export $(cat .env | xargs)
    API_URL=${SPOTIFY_API_URL:-$SPOTIFY_API_URL}
    INTERVAL=${LISTEN_INTERVAL:-$LISTEN_INTERVAL}
else
    echo ".env file not found."
    exit 1
fi

go build -ldflags "-X main.APIUrl=$API_URL -X main.ListenInterval=$INTERVAL" -o spotify-lyrics