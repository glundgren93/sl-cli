# SKILL.md — sl-cli

Stockholm public transport CLI. No API key required.

## Install

```bash
go install github.com/glundgren93/sl-cli@latest
```

Binary: `sl` — Requires: Go 1.21+

## Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `sl departures` | Real-time departures | `sl departures --address "Stureplan" --json` |
| `sl trip` | A→B journey planning | `sl trip --from "Slussen" --to "Arlanda" --json` |
| `sl nearby` | Find stops near a location | `sl nearby --address "Stureplan" --json` |
| `sl stop-info` | Lines serving a stop | `sl stop-info --stop "Slussen" --json` |
| `sl search` | Find stops by name | `sl search "Medborg" --json` |
| `sl deviations` | Service disruptions | `sl deviations --mode METRO --json` |
| `sl lines` | List all transit lines | `sl lines --mode BUS --json` |

Always pass `--json`.

## Stop addressing (pick one)

- `--site <id>` — exact site ID, fastest
- `--stop "<name>"` — fuzzy name match
- `--address "<street>"` — geocodes to coordinates, finds nearest stop

## Key flags

- `--line <designation>` — filter by line (e.g. `55`, `14`)
- `--mode <MODE>` — filter by transport: `BUS`, `METRO`, `TRAIN`, `TRAM`, `SHIP`
- `--results <n>` — number of trip results (default 5)
- `--max-changes <n>` — max interchanges for trip (0 = direct only)
- `--route-type` — `leastwalking` or `leastchanges`
- `--radius <km>` — nearby search radius (default 0.5)
- `--lines` — show which lines serve each nearby stop (slower, extra API calls)
- `--future` — include planned/future deviations

## Output shapes

**departures --address (no filter)** → `[{ stop, site_id, distance_m, departures: [{ line, transport_mode, destination, minutes_left }], deviations }]`

**departures --address --line/--mode** → `{ stop, site_id, distance_m, departures, deviations }`

**trip** → `{ from, to, journeys: [{ tripDuration, interchanges, legs }] }`

**stop-info** → `{ stop, site_id, lines: [{ designation, transport_mode, destinations }] }`

**nearby** → `[{ name, site_id, distance_m, lat, lon, lines? }]`

**search** → `[{ name, site_id, lat, lon }]`

**deviations** → `[{ header, details, scope, from_date, to_date }]`

**lines** → `[{ designation, transport_mode, group_of_lines }]`

**Errors** → stderr: `{"error": "message"}` — Empty results: `[]`, never `null`.

## Gotchas

- Mode and line values are case-sensitive: `BUS` not `bus`
- `nearby --lines` makes one API call per stop — use only when needed
- `--address` geocodes via API — slightly slower than `--site`/`--stop`
- No rate limiting on SL APIs, but no retry logic either — transient failures return errors
- Swedish characters work: "Södermalm", "Kärrtorp"
- `departures` without `--line`/`--mode` returns up to 5 nearby stops — one call covers all modes

## This tool does NOT

- Buy tickets or manage payments
- Track vehicles in real-time on a map
- Show historical data or schedules (only real-time)
- Cover regions outside Stockholm (SL network only)
