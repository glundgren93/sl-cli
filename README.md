# sl-cli

A command-line interface for Stockholm's public transport (SL). Designed for both humans and AI agents.

**No API key required.** Data sourced from [SL via Trafiklab](https://www.trafiklab.se/api/our-apis/sl/).

## Features

- 🚌 **Real-time departures** from any stop
- 🗺️ **Trip planning** between any two locations
- 📍 **Nearby stops** by coordinates or address
- 🔍 **Stop search** by name
- ⚠️ **Service deviations** and disruptions
- 🚇 **Line listings** across all transport modes
- 🤖 **JSON output** for agent/machine consumption (`--json`)

## Installation

```bash
# From source
go install github.com/glundgren/sl-cli@latest

# Or build locally
git clone https://github.com/glundgren/sl-cli.git
cd sl-cli
go build -o sl .
```

## Usage

### Departures

Get real-time departures from a stop:

```bash
# By site ID
sl departures --site 9530

# By name (fuzzy search)
sl departures --stop "Medborgarplatsen"

# Filter by line
sl departures --site 9530 --line 55

# Filter by transport mode
sl departures --site 9191 --mode METRO

# JSON output (for agents)
sl departures --site 9530 --line 55 --json
```

### Trip Planning

Plan a journey between two locations:

```bash
sl trip --from "Medborgarplatsen" --to "T-Centralen"
sl trip --from "Magnus Ladulåsgatan" --to "Slussen" --results 5
sl trip --from "Medborgarplatsen" --to "Arlanda" --max-changes 2
```

### Nearby Stops

Find stops near a location:

```bash
# By coordinates
sl nearby --lat 59.3121 --lon 18.0643

# By address
sl nearby --address "Magnus Ladulåsgatan"

# Custom radius (km)
sl nearby --lat 59.3121 --lon 18.0643 --radius 1.0
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
```

### Lines

List transit lines:

```bash
sl lines                         # All lines
sl lines --mode BUS              # Buses only
sl lines --mode METRO --json     # Metro lines as JSON
```

## Agent Integration

All commands support `--json` for structured output. This makes `sl-cli` ideal for AI agents and automation:

```bash
# An agent checking departures for a user
sl departures --site 9530 --line 55 --json

# An agent finding the nearest stops to a user's location
sl nearby --lat 59.3121 --lon 18.0643 --json

# An agent planning a route
sl trip --from "Medborgarplatsen" --to "Arlanda" --json

# An agent checking for disruptions on a user's commute
sl deviations --line 55 --json
```

## Transport Modes

| Mode    | Flag value | Description         |
|---------|-----------|---------------------|
| Bus     | `BUS`     | City & regional buses |
| Metro   | `METRO`   | Tunnelbana (subway)  |
| Train   | `TRAIN`   | Pendeltåg & trains   |
| Tram    | `TRAM`    | Tvärbanan, Lidingöbanan, etc. |
| Ship    | `SHIP`    | Waxholmsbolaget ferries |

## APIs Used

| API | Base URL | Auth |
|-----|----------|------|
| [SL Transport](https://www.trafiklab.se/api/our-apis/sl/transport/) | `transport.integration.sl.se/v1` | None |
| [SL Deviations](https://www.trafiklab.se/api/our-apis/sl/deviations/) | `deviations.integration.sl.se/v1` | None |
| [SL Journey Planner v2](https://www.trafiklab.se/api/our-apis/sl/journey-planner-2/) | `journeyplanner.integration.sl.se/v2` | None |

## License

MIT
