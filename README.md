# Canonical Specifications

A service for syncing, managing, and viewing Canonical specifications from Google Drive.

## Overview

This project provides a system to automatically synchronize specification documents from Google Drive, store them in a PostgreSQL database, and make them available through a web UI. It consists of:

- A sync service that periodically fetches specifications from Google Drive
- A REST API for accessing the specifications
- A web UI for browsing and searching specifications
- A PostgreSQL database for storage


## Requirements

- Canonical Workshop: https://snapcraft.io/workshop
- Google Drive API credentials
- Taskfile: https://taskfile.dev/installation/

## Setup

### Environment Configuration

Create a `.env.local` file in the root of the project with the following environment variables:

```bash
# Used for Google Drive API
GOOGLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\nREPLACE_ME\n-----END PRIVATE KEY-----"
GOOGLE_PRIVATE_KEY_ID=1234

# Used for Google OAuth for UI login
GOOGLE_OAUTH_CLIENT_ID=1234
GOOGLE_OAUTH_CLIENT_SECRET=REPLACE_ME
```

### Database Setup

1. Launch the workshop:
```sh
workshop launch --verbose
```

2. Build the application and run the database migrations:
```sh
workshop run specs task build
workshop run specs task migrate
```

## Development
3. Start the sync service:
```sh
workshop run specs task run_sync
```

4. Start the web server for the API and UI:
```sh
workshop run specs task run
```

Or, for hot reloading:
```sh
workshop run specs task dev
```

Access the web application at http://localhost:3000.

## Production Deployment
The project includes a Rockfile for deploying as Charm on Juju:

```yaml
services:
  go:
    override: replace
    command: /usr/bin/specs-api
    startup: enabled
  sync-scheduler:
    override: replace
    startup: enabled
    command: /usr/bin/specs-sync
    environment:
      SYNC_INTERVAL: 30m
  reject-scheduler:
    override: replace
    startup: enabled
    command: /usr/bin/specs-reject
    environment:
      REJECT_INTERVAL: 24h
```

### Sync Process

The sync service:
1. Fetches all subfolders from the configured root Google Drive folder
2. For each subfolder, fetches all Google Docs
3. Parses metadata from the first table in each document
4. Updates the database with the specification information
5. Deletes specifications that are no longer present in Google Drive
