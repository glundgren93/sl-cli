# sl-cli

A command-line interface for Stockholm's public transport (SL).

**No API key required.** Uses [SL's open APIs](https://www.trafiklab.se/api/our-apis/sl/).

> **🤖 AI Agent?** Read [`SKILL.md`](SKILL.md) — it has everything you need in one file.

## Install

```bash
# From source (requires Go 1.21+)
go install github.com/glundgren93/sl-cli@latest

# Or clone and build
git clone https://github.com/glundgren93/sl-cli.git
cd sl-cli && go build -o sl .
```

## Quick start

```bash
# Departures near an address (all modes, all stops)
sl departures --address "Drottninggatan 45"

# Next bus 55 from a specific address
sl departures --address "Magnus Ladulåsgatan 7" --line 55

# Trip from A to B
sl trip --from "Slussen" --to "Arlanda"

# Nearby stops
sl nearby --address "Stureplan"

# What lines serve a stop?
sl stop-info --stop "Medborgarplatsen"

# Search for a stop
sl search "Medborg"

# Service disruptions
sl deviations --mode METRO

# List lines
sl lines --mode BUS
```

## Stop addressing

Three ways to identify a stop — pick whichever fits:

| Flag | Input | Speed |
|------|-------|-------|
| `--site <id>` | Exact site ID (e.g. `9530`) | Fastest |
| `--stop "<name>"` | Fuzzy name search | Fast |
| `--address "<street>"` | Street address, geocoded | Slower (geocoding call) |

`--address` is the most flexible — it accepts any street, landmark, or place name.

## Commands

### `sl departures`

Real-time departures from a stop.

```bash
sl departures --site 9530
sl departures --stop "T-Centralen"
sl departures --address "Stureplan" --mode METRO
sl departures --address "Magnus Ladulåsgatan 7" --line 55
```

With `--address` and no filter: returns departures from ALL nearby stops (up to 5) — buses, trains, metro in one call.

With `--line` or `--mode`: finds the nearest stop serving that specific line/mode. Deviations shown inline.

### `sl trip`

Journey planning between two locations.

```bash
sl trip --from "Slussen" --to "T-Centralen"
sl trip --from "Magnus Ladulåsgatan 7" --to "Arlanda" --results 5
sl trip --from "Slussen" --to "Kista" --max-changes 0
sl trip --from "Slussen" --to "Kista" --route-type leastwalking
```

| Flag | Description |
|------|-------------|
| `--results <n>` | Number of journey alternatives (default 5) |
| `--max-changes <n>` | Max interchanges (0 = direct only) |
| `--route-type` | `leastwalking` or `leastchanges` |

### `sl nearby`

Find stops near a location.

```bash
sl nearby --address "Stureplan"
sl nearby --lat 59.3121 --lon 18.0643 --radius 1.0
sl nearby --address "Stureplan" --lines    # also shows which lines serve each stop (slower)
```

### `sl stop-info`

Lines serving a stop (based on real-time departures).

```bash
sl stop-info --stop "Medborgarplatsen"
sl stop-info --address "Magnus Ladulåsgatan 7"
```

### `sl search`

Search stops by name.

```bash
sl search "Stockholm City"
```

### `sl deviations`

Current service disruptions.

```bash
sl deviations                   # all
sl deviations --mode METRO      # metro only
sl deviations --line 55         # line 55 only
sl deviations --future          # include planned disruptions
```

### `sl lines`

List transit lines.

```bash
sl lines                        # all lines
sl lines --mode TRAM            # trams only
```

## JSON output

All commands support `--json` for structured, machine-readable output.

```bash
sl departures --address "Stureplan" --json
sl trip --from "Slussen" --to "Kista" --json
```

Errors go to stderr as `{"error": "message"}`. Empty results are always `[]`, never `null`.

## Transport modes

| Flag value | Description |
|-----------|-------------|
| `BUS` | City & regional buses |
| `METRO` | Tunnelbana (subway) |
| `TRAIN` | Pendeltåg & regional trains |
| `TRAM` | Tvärbanan, Lidingöbanan, Spårväg City |
| `SHIP` | Waxholmsbolaget ferries |

Values are case-sensitive.

## Development

```bash
go test ./...
go build -o sl .
```

## License

MIT
