# Go Live Data Engine

This project explores features in Golang to simulate aggregating and streaming live data from local CSVs and external APIs, as well as creating a Server Sent Event API

## Features

- Allows a user to connect to an SSE Endpoint and stream data in real-time
- Parses information from CSVs and external APIs through frequent polling to return results
- Monitors the service health in real time as data is processed from each service

## Installation

Make sure Docker Desktop is installed, then run the following:

```bash
docker-compose up -d
```
 This will start all the services required for the project to run, including creating initial database users

## Quick Start

```bash
curl -N http://localhost:8080/stream?user_id=1
```

You can use any user ID up to 1000, but this stream will stay alive and receive the data.
