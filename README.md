# Canonical Specifications

A service for syncing, managing, and viewing Canonical specifications from Google Drive.


## Overview

This project provides a system to automatically synchronize specification documents from Google Drive, store them in a PostgreSQL database, and make them available through a web UI. It consists of:

- A sync service that periodically fetches specifications from Google Drive
- A REST API for accessing the specifications
- A web UI for browsing and searching specifications
- A PostgreSQL database for storage


## Requirements

- Taskfile: https://taskfile.dev/installation/
- Go 1.23 or higher: https://go.dev/doc/install
- Bun.js: https://bun.sh/docs/installation
- Docker & Docker Compose (for local development): https://docs.docker.com/get-docker/
- Google Drive API credentials

## Setup



### Environment Configuration

Create a `.env.local` file in the root of the project with the following environment variables:

```bash
# Used for Google Drive API
GOOGLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\nREFPLACE_ME\n-----END PRIVATE KEY-----"
GOOGLE_PRIVATE_KEY_ID=1234

# Used for Google OAuth for UI login
GOOGLE_OAUTH_CLIENT_ID=1234
GOOGLE_OAUTH_CLIENT_SECRET=REPLACE_ME
```

### Database Setup

1. Start the PostgreSQL database using Docker:

```bash
docker-compose up -d
```

2. Run the database migrations:

```bash
task migrate
```

## Development

This project uses [Task](https://taskfile.dev/) (user friendly Makefile alterntive) for managing development tasks:

### Run the Google Drive Sync Service

```bash
task run_sync
```

### Run the Web Server for the API and UI

```bash
task run
```

Or, for hot reloading:

```bash
task dev
```

## Production Deployment
The project includes a Rockfile for deploying as Charm on Juju:

```yaml
services:
  go:
    override: replace
    command: /usr/bin/specs-api
    startup: enabled
  go-scheduler:
    override: replace
    startup: enabled
    command: /usr/bin/specs-sync
    environment:
      SYNC_INTERVAL: 30m
```

### Sync Process

The sync service:
1. Fetches all subfolders from the configured root Google Drive folder
2. For each subfolder, fetches all Google Docs
3. Parses metadata from the first table in each document
4. Updates the database with the specification information
5. Deletes specifications that are no longer present in Google Drive
