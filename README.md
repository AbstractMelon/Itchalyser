# Itchalyser Scraper

A command-line tool for extracting data from itch.io game jams while being kind to their servers.

## Features

This tool extracts the following data:

- Jam metadata (title, start/end dates, hosts, etc.)
- Game metadata (title, description, tags, etc.)
- Game media (cover images, screenshots, etc.)
- Game files (if the jam allows it)

## Installation

### Requirements

- Go 1.16 or higher

### Building from source

```bash
git clone https://github.com/AbstractMelon/Itchalyser.git
cd itchjam
go build
```

## Usage

### Basic Usage

```bash
./Itchalyser -jam https://itch.io/jam/brackeys-13 -output json -dir ./data
```

### Options

- `-jam`: Comma-separated list of jam URLs (required)
- `-output`: Output format (json, jsonl, markdown) - default: json
- `-dir`: Directory to store output - default: ./data
- `-workers`: Number of concurrent workers - default: 2
- `-media`: Download media files (true/false) - default: true
- `-games`: Download game files (true/false) - default: false
- `-user-agent`: User agent string for HTTP requests - default: DefaultUserAgent
- `-delay`: Delay between requests in milliseconds - default: 1500

### Examples

Process a single jam:

```bash
./Itchalyser -jam https://itch.io/jam/brackeys-13
```

Process multiple jams:

```bash
./Itchalyser -jam "https://itch.io/jam/brackeys-13,https://itch.io/jam/game-off-2024"
```

Generate markdown reports:

```bash
./Itchalyser -jam https://itch.io/jam/brackeys-13 -output markdown
```

Increase worker count for faster processing:

```bash
./Itchalyser -jam https://itch.io/jam/brackeys-13 -workers 5
```

## Output Structure

```
./data/
  jams/
    {jam-id}/
      meta.json
      cover.png
      submissions/
        {game-id}/
          game.json
          media/
            cover.png
            screenshot1.jpg
            screenshot2.jpg
          files/
            game.zip
  reports/
    {jam-id}-report.md
```

## License

GPL v3
