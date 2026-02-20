package model

import "time"

// Site represents a transit stop/station in SL's network.
type Site struct {
	ID           int       `json:"id"`
	GID          int64     `json:"gid"`
	Name         string    `json:"name"`
	Note         string    `json:"note,omitempty"`
	Abbreviation string    `json:"abbreviation,omitempty"`
	Aliases      []string  `json:"alias,omitempty"`
	Lat          float64   `json:"lat"`
	Lon          float64   `json:"lon"`
	StopAreas    []int     `json:"stop_areas,omitempty"`
	Valid        *Validity `json:"valid,omitempty"`
}

type Validity struct {
	From string `json:"from"`
	Upto string `json:"upto,omitempty"`
}

// Line represents a transit line.
type Line struct {
	ID                   int    `json:"id"`
	Designation          string `json:"designation"`
	TransportAuthorityID int    `json:"transport_authority_id"`
	TransportMode        string `json:"transport_mode"`
	GroupOfLines         string `json:"group_of_lines,omitempty"`
}

// Departure represents a single departure from a stop.
type Departure struct {
	Destination   string     `json:"destination"`
	DirectionCode int        `json:"direction_code"`
	Direction     string     `json:"direction"`
	State         string     `json:"state"`
	Display       string     `json:"display"`
	Scheduled     string     `json:"scheduled"`
	Expected      string     `json:"expected"`
	Journey       *Journey   `json:"journey,omitempty"`
	StopArea      *StopArea  `json:"stop_area,omitempty"`
	StopPoint     *StopPoint `json:"stop_point,omitempty"`
	Line          *Line      `json:"line,omitempty"`
	Deviations    []any      `json:"deviations,omitempty"`
}

type Journey struct {
	ID              int64  `json:"id"`
	State           string `json:"state"`
	PredictionState string `json:"prediction_state"`
}

type StopArea struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type StopPoint struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Designation string `json:"designation,omitempty"`
}

// DeparturesResponse is the API response for departures.
type DeparturesResponse struct {
	Departures    []Departure `json:"departures"`
	StopDeviations []any      `json:"stop_deviations,omitempty"`
}

// Deviation represents a service disruption.
type Deviation struct {
	Version         int               `json:"version"`
	Created         string            `json:"created"`
	Modified        string            `json:"modified,omitempty"`
	DeviationCaseID int               `json:"deviation_case_id"`
	Publish         *PublishWindow    `json:"publish,omitempty"`
	Priority        *Priority         `json:"priority,omitempty"`
	MessageVariants []MessageVariant  `json:"message_variants,omitempty"`
	Scope           *DeviationScope   `json:"scope,omitempty"`
}

type PublishWindow struct {
	From string `json:"from"`
	Upto string `json:"upto"`
}

type Priority struct {
	ImportanceLevel int `json:"importance_level"`
	InfluenceLevel  int `json:"influence_level"`
	UrgencyLevel    int `json:"urgency_level"`
}

type MessageVariant struct {
	Header     string `json:"header"`
	Details    string `json:"details"`
	ScopeAlias string `json:"scope_alias"`
	Language   string `json:"language"`
}

type DeviationScope struct {
	StopAreas []DeviationStopArea `json:"stop_areas,omitempty"`
	Lines     []Line              `json:"lines,omitempty"`
}

type DeviationStopArea struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	TransportMode string `json:"transport_mode"`
}

// StopFinderResponse is the response from the journey planner stop-finder endpoint.
type StopFinderResponse struct {
	Locations      []Location       `json:"locations"`
	SystemMessages []SystemMessage  `json:"systemMessages,omitempty"`
}

type Location struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	DisassembledName string     `json:"disassembledName"`
	Type             string     `json:"type"`
	Coord            [2]float64 `json:"coord"`
	IsBest           bool       `json:"isBest"`
	MatchQuality     int        `json:"matchQuality"`
	Parent           *Parent    `json:"parent,omitempty"`
}

type Parent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type SystemMessage struct {
	Type    string `json:"type"`
	Module  string `json:"module"`
	Code    int    `json:"code"`
	Text    string `json:"text"`
	SubType string `json:"subType"`
}

// JourneyResponse is the response from the journey planner trips endpoint.
type JourneyResponse struct {
	SystemMessages []SystemMessage `json:"systemMessages,omitempty"`
	Journeys       []JourneyTrip   `json:"journeys,omitempty"`
}

type JourneyTrip struct {
	TripDuration   int           `json:"tripDuration"`
	TripRtDuration int           `json:"tripRtDuration"`
	Rating         int           `json:"rating"`
	Interchanges   int           `json:"interchanges"`
	IsAdditional   bool          `json:"isAdditional"`
	Legs           []JourneyLeg  `json:"legs"`
}

type JourneyLeg struct {
	Duration    int              `json:"duration"`
	Origin      *JourneyStop     `json:"origin"`
	Destination *JourneyStop     `json:"destination"`
	Transport   *JourneyTransport `json:"transportation,omitempty"`
	Infos       []any            `json:"infos,omitempty"`
	IsRealtimeControlled bool    `json:"isRealtimeControlled"`
}

type JourneyStop struct {
	ProductClasses         []int      `json:"productClasses,omitempty"`
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	DisassembledName       string     `json:"disassembledName"`
	Type                   string     `json:"type"`
	Coord                  [2]float64 `json:"coord"`
	DepartureTimePlanned   string     `json:"departureTimePlanned,omitempty"`
	DepartureTimeEstimated string     `json:"departureTimeEstimated,omitempty"`
	ArrivalTimePlanned     string     `json:"arrivalTimePlanned,omitempty"`
	ArrivalTimeEstimated   string     `json:"arrivalTimeEstimated,omitempty"`
	Parent                 *Parent    `json:"parent,omitempty"`
	Properties             map[string]any `json:"properties,omitempty"`
}

type JourneyTransport struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Number      string          `json:"number"`
	Description string          `json:"description"`
	Product     *TransportProduct `json:"product,omitempty"`
	Destination *TransportDest  `json:"destination,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

type TransportProduct struct {
	ID       int    `json:"id"`
	Class    int    `json:"class"`
	Name     string `json:"name"`
	IconID   int    `json:"iconId"`
	CatCode  int    `json:"catCode"`
	CatOutS  string `json:"catOutS"`
	CatOutL  string `json:"catOutL"`
}

type TransportDest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ParsedDeparture is a processed departure with parsed times.
type ParsedDeparture struct {
	Line          string        `json:"line"`
	TransportMode string        `json:"transport_mode"`
	GroupOfLines  string        `json:"group_of_lines,omitempty"`
	Destination   string        `json:"destination"`
	Direction     string        `json:"direction"`
	Display       string        `json:"display"`
	Scheduled     time.Time     `json:"scheduled"`
	Expected      time.Time     `json:"expected"`
	MinutesLeft   int           `json:"minutes_left"`
	State         string        `json:"state"`
	StopArea      string        `json:"stop_area"`
	StopPoint     string        `json:"stop_point"`
	Platform      string        `json:"platform,omitempty"`
	Deviations    []string      `json:"deviations,omitempty"`
}
