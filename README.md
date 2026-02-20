# sl-cli

A command-line interface for Stockholm's public transport (SL). Designed for both humans and AI agents.

**No API key required.** Data sourced from [SL via Trafiklab](https://www.trafiklab.se/api/our-apis/sl/).

## Features

- 🚌 **Real-time departures** from any stop — by ID, name, or street address
- 🗺️ **Trip planning** between any two locations (stops or addresses)
- 📍 **Nearby stops** by coordinates or address, with optional line info
- ℹ️ **Stop info** — see which lines serve a stop
- 🔍 **Stop search** by name
- ⚠️ **Service deviations** — standalone or inline with departures
- 🚇 **Line listings** across all transport modes
- 🤖 **JSON output** for agent/machine consumption (`--json`)

## Installation

```bash
# From source (requires Go 1.21+)
go install github.com/glundgren93/sl-cli@latest

# Or build locally
git clone https://github.com/glundgren93/sl-cli.git
cd sl-cli
go build -o sl .
```

## Usage

### Departures

Get real-time departures from a stop. Supports site ID, stop name, or street address:

```bash
# By site ID
sl departures --site 9530

# By stop name (fuzzy search)
sl departures --stop "Medborgarplatsen"

# By street address — finds nearest stop serving the requested line
sl departures --address "Magnus Ladulåsgatan 7" --line 55

# By address — nearest train station
sl departures --address "Drottninggatan 45" --mode TRAIN

# By address — nearest metro station
sl departures --address "Stureplan" --mode METRO

# JSON output (for agents)
sl departures --address "Magnus Ladulåsgatan 7" --line 55 --json
```

When using `--address`, the CLI:
1. Geocodes the address via SL's journey planner
2. Finds nearby stops (within `--radius`, default 1km)
3. Scans them to find the closest one serving the requested `--line` or `--mode`
4. Fetches relevant service deviations and shows them inline

### Trip Planning

Plan a journey from A to B. Accepts stop names, stop IDs, or street addresses:

```bash
sl trip --from "Medborgarplatsen" --to "T-Centralen"
sl trip --from "Magnus Ladulåsgatan 7" --to "Stureplan"
sl trip --from "Drottninggatan 45" --to "Arlanda" --results 5
sl trip --from "Medborgarplatsen" --to "T-Centralen" --max-changes 0
sl trip --from "Slussen" --to "Kista" --route-type leastwalking
```

### Stop Info

Show which transit lines serve a specific stop:

```bash
# By site ID
sl stop-info --site 9530

# By stop name
sl stop-info --stop "Medborgarplatsen"

# By street address (finds nearest stop)
sl stop-info --address "Magnus Ladulåsgatan 7"

# JSON output
sl stop-info --address "Magnus Ladulåsgatan 7" --json
```

Uses real-time departure data, so results reflect currently operating lines.

### Nearby Stops

Find stops near a location:

```bash
# By coordinates
sl nearby --lat 59.3121 --lon 18.0643

# By address
sl nearby --address "Magnus Ladulåsgatan 7"

# Show which lines serve each stop (makes API calls per stop, slower)
sl nearby --address "Stureplan" --lines

# Custom radius (km)
sl nearby --lat 59.3121 --lon 18.0643 --radius 1.0 --json
```

### Search

Search for stops by name:

```bash
sl search Medborgarplatsen
sl search "Stockholm City"
sl search Slussen --json
```

### Deviations

Check service disruptions:

```bash
sl deviations                    # All deviations
sl deviations --mode METRO       # Metro only
sl deviations --line 55          # Line 55 only
sl deviations --future           # Include planned
sl deviations --json             # JSON output
```

### Lines

List transit lines:

```bash
sl lines                         # All lines
sl lines --mode BUS              # Buses only
sl lines --mode METRO --json     # Metro lines as JSON
```

## Agent Integration

All commands support `--json` for structured output. This makes `sl-cli` ideal for AI agents and automation.

### Error Handling

When `--json` is set, errors are output as JSON on stderr:

```json
{"error": "55 not found at any stop within 1000m of \"Stureplan\""}
```

Empty results always return `[]` (never `null`), so `len()` works safely.

### Departures JSON

```json
{
  "stop": "Rosenlundsgatan ( M Ladulåsg)",
  "site_id": 1363,
  "distance_m": 119,
  "departures": [
    {
      "line": "55",
      "transport_mode": "BUS",
      "destination": "Henriksdalsberget",
      "display": "6 min",
      "minutes_left": 6,
      "state": "EXPECTED",
      "stop_area": "Rosenlundsgatan ( M Ladulåsg)",
      "platform": "A"
    }
  ],
  "deviations": [
    {
      "line": "55",
      "header": "Reduced service",
      "details": "..."
    }
  ]
}
```

### Trip JSON

```json
{
  "from": "Magnus Ladulåsgatan 7",
  "to": "Stureplan",
  "journeys": [
    {
      "tripDuration": 960,
      "interchanges": 1,
      "legs": [...]
    }
  ]
}
```

### Stop Info JSON

```json
{
  "stop": "Rosenlundsgatan ( M Ladulåsg)",
  "site_id": 1363,
  "distance_m": 119,
  "lines": [
    {
      "designation": "55",
      "transport_mode": "BUS",
      "destinations": ["Tanto", "Henriksdalsberget"]
    }
  ]
}
```

### Example agent workflows

```bash
# "When is my bus coming?"
sl departures --address "Magnus Ladulåsgatan 7" --line 55 --json

# "What buses are near me?"
sl nearby --lat 59.3121 --lon 18.0643 --lines --json

# "Which lines serve Medborgarplatsen?"
sl stop-info --stop "Medborgarplatsen" --json

# "How do I get to Arlanda from home?"
sl trip --from "Magnus Ladulåsgatan 7" --to "Arlanda" --json

# "Any disruptions on the 55?"
sl deviations --line 55 --json

# "Find nearest stop with metro"
sl nearby --address "Drottninggatan 45" --lines --json
```

## Transport Modes

| Mode  | Flag value | Description                          |
|-------|-----------|--------------------------------------|
| Bus   | `BUS`     | City & regional buses                |
| Metro | `METRO`   | Tunnelbana (subway)                  |
| Train | `TRAIN`   | Pendeltåg (commuter rail) & trains   |
| Tram  | `TRAM`    | Tvärbanan, Lidingöbanan, Spårväg City |
| Ship  | `SHIP`    | Waxholmsbolaget ferries              |

## APIs Used

| API | Base URL | Auth |
|-----|----------|------|
| [SL Transport](https://www.trafiklab.se/api/our-apis/sl/transport/) | `transport.integration.sl.se/v1` | None |
| [SL Deviations](https://www.trafiklab.se/api/our-apis/sl/deviations/) | `deviations.integration.sl.se/v1` | None |
| [SL Journey Planner v2](https://www.trafiklab.se/api/our-apis/sl/journey-planner-2/) | `journeyplanner.integration.sl.se/v2` | None |

## Development

```bash
# Run tests
go test ./...

# Build
go build -o sl .

# Build with version info
go build -ldflags "-X github.com/glundgren93/sl-cli/cmd.Version=1.0.0 -X github.com/glundgren93/sl-cli/cmd.Commit=$(git rev-parse --short HEAD)" -o sl .
```

## License

MIT
