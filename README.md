<h1 align="center">
  Minialert
</h1>

<h3 align="center">
  A lightweight <a href="https://prometheus.io">Prometheus</a> alert manager.

  Designed as an alternative to <a href="https://github.com/prometheus/alertmanager">Alertmanager</a>.

[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/yukitsune/minialert/Build%20and%20Test)](https://github.com/yukitsune/minialert/actions?query=workflow:Build%20and%20Test)
[![Go Report Card](https://goreportcard.com/badge/github.com/yukitsune/minialert)](https://goreportcard.com/report/github.com/yukitsune/minialert)
[![License](https://img.shields.io/github/license/YuKitsune/minialert)](https://github.com/YuKitsune/minialert/blob/main/LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/YuKitsune/minialert?include_prereleases)](https://github.com/YuKitsune/minialert/releases)
[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/yukitsune/minialert)](https://pkg.go.dev/mod/github.com/yukitsune/minialert)

  <img src="demo.png" />
</h3>

# Installation

## Docker

The easiest way to run Minialert is via [Docker](https://www.docker.com).

```shell
docker run \
  --name minialert \
  -e MINIALERT_DATABASE_URI=mongodb://username:password@host:27017
  -e MINIALERT_DATABASE_NAME=minialert \
  ghcr.io/yukitsune/minialert:latest
```

## Precompiled Binary

Precompiled binaries can be downloaded from the [releases](https://github.com/yukitsune/minialert/releases) section on GitHub.

## Compiling from Source

Minialert can be combined from source by running the following command:

```shell
# Using make
make build

# Without make
mkdir -p bin
go build -ldflags="$(LD_FLAGS)" -o ./bin/  ./cmd/...
```

# Configuration

Minialert can be configured using a yaml config file, or environment variables.
By default, minialert will check for a `minialert.yaml` file in the current working directory, or `/etc/minialert/`.

A config file can be specified manually using the `--config` flag. Example:
```shell
minialert --config ./somewhere/else/custom-config.yaml
```

Here is an example config file, along with corresponding environment variables.

```yaml
database:

  # The MongoDB URI.
  # MINIALERT_DATABASE_URI
  uri:

  # The name of the database to use.
  # MINIALERT_DATABASE_NAME
  name:

  # Whether or not to use an in-memory database.
  # Note: All data will be lost after the program has exited.
  # MINIALERT_DATABASE_INMEMORY
  inMemory: false

bot:

  # The bot token.
  # MINIALERT_BOT_TOKEN
  token:

  # The client ID.
  # MINIALERT_BOT_CLIENTID
  clientId:

  # The permissions integer represented as a string.
  # Discord provides resources to help determine the permissions number:
  # https://discord.com/developers/docs/topics/community-resources#permission-calculators
  # MINIALERT_BOT_PERMISSIONS
  permissions:

  # (Optional) The list of scopes which the bot requires.
  # Defaults to "bot", "application.commands".
  # MINIALERT_BOT_SCOPES
  scopes:
    - bot
    - applications.commands

log:

  # (Optional) The level of logging.
  # Defaults to "info"
  # MINIALERT_LOG_LEVEL
  level: info

```

# Setup

When the bot starts, an invite link is written to the logs.
Once the bot has joined a server, a scrape config needs to be created. This can be done using the `/create-scrape-config` command.

Once the scrape config is created, minialert will begin scraping using the given config.
Multiple scrape configs can be specified. A separate goroutine is created for each scrape config, allowing the bot to scrape multiple endpoints at once.

Scrape configs can be updated using `/update-scrape-config`, and removed using `/remove-scrape-config`.

`/get-alerts` can be used to get all currently firing alerts for a particular scrape config.

# Contributing

Contributions are what make the open source community such an amazing place to be, learn, inspire, and create.
Any contributions are **greatly appreciated**!
