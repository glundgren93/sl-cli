package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/glundgren/sl-cli/internal/model"
)

const (
	TransportBaseURL      = "https://transport.integration.sl.se/v1"
	DeviationsBaseURL     = "https://deviations.integration.sl.se/v1"
	JourneyPlannerBaseURL = "https://journeyplanner.integration.sl.se/v2"

	DefaultTimeout = 15 * time.Second
)

// Client is the SL API client.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new SL API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// NewClientWithTimeout creates a client with a custom timeout.
func NewClientWithTimeout(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}
		defer gr.Close()
		reader = gr
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// --- Transport API ---

// GetSites returns all sites (stops/stations) in SL's network.
func (c *Client) GetSites(ctx context.Context) ([]model.Site, error) {
	body, err := c.get(ctx, TransportBaseURL+"/sites?expand=true")
	if err != nil {
		return nil, err
	}
	var sites []model.Site
	if err := json.Unmarshal(body, &sites); err != nil {
		return nil, fmt.Errorf("parsing sites: %w", err)
	}
	return sites, nil
}

// GetLines returns all lines, optionally filtered by transport authority.
func (c *Client) GetLines(ctx context.Context, transportAuthorityID int) ([]model.Line, error) {
	u := TransportBaseURL + "/lines"
	if transportAuthorityID > 0 {
		u += fmt.Sprintf("?transport_authority_id=%d", transportAuthorityID)
	}
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var lines []model.Line
	if err := json.Unmarshal(body, &lines); err != nil {
		return nil, fmt.Errorf("parsing lines: %w", err)
	}
	return lines, nil
}

// DepartureOptions configures a departures request.
type DepartureOptions struct {
	SiteID        int
	TransportMode string // BUS, METRO, TRAM, TRAIN, SHIP, FERRY
	Line          string // filter by line designation
	Direction     int    // 1 or 2
}

// GetDepartures returns departures from a site.
func (c *Client) GetDepartures(ctx context.Context, opts DepartureOptions) (*model.DeparturesResponse, error) {
	if opts.SiteID == 0 {
		return nil, fmt.Errorf("site ID is required")
	}
	u := fmt.Sprintf("%s/sites/%d/departures", TransportBaseURL, opts.SiteID)

	params := url.Values{}
	if opts.TransportMode != "" {
		params.Set("transport", strings.ToUpper(opts.TransportMode))
	}
	if opts.Direction > 0 {
		params.Set("direction", strconv.Itoa(opts.Direction))
	}
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp model.DeparturesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing departures: %w", err)
	}

	// Filter by line if specified
	if opts.Line != "" {
		var filtered []model.Departure
		for _, d := range resp.Departures {
			if d.Line != nil && strings.EqualFold(d.Line.Designation, opts.Line) {
				filtered = append(filtered, d)
			}
		}
		resp.Departures = filtered
	}

	return &resp, nil
}

// --- Deviations API ---

// DeviationOptions configures a deviations request.
type DeviationOptions struct {
	Future         bool
	SiteIDs        []int
	LineIDs        []int
	TransportModes []string // BUS, METRO, TRAM, TRAIN, SHIP, FERRY, TAXI
}

// GetDeviations returns current service deviations.
func (c *Client) GetDeviations(ctx context.Context, opts DeviationOptions) ([]model.Deviation, error) {
	params := url.Values{}
	if opts.Future {
		params.Set("future", "true")
	}
	for _, id := range opts.SiteIDs {
		params.Add("site", strconv.Itoa(id))
	}
	for _, id := range opts.LineIDs {
		params.Add("line", strconv.Itoa(id))
	}
	for _, mode := range opts.TransportModes {
		params.Add("transport_mode", strings.ToUpper(mode))
	}

	u := DeviationsBaseURL + "/messages"
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var devs []model.Deviation
	if err := json.Unmarshal(body, &devs); err != nil {
		return nil, fmt.Errorf("parsing deviations: %w", err)
	}
	return devs, nil
}

// --- Journey Planner API ---

// FindStops searches for stops by name.
func (c *Client) FindStops(ctx context.Context, query string) ([]model.Location, error) {
	params := url.Values{}
	params.Set("name_sf", query)
	params.Set("type_sf", "any")
	params.Set("any_obj_filter_sf", "2") // stops only

	u := JourneyPlannerBaseURL + "/stop-finder?" + params.Encode()
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp model.StopFinderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing stop finder: %w", err)
	}
	return resp.Locations, nil
}

// TripOptions configures a trip planning request.
type TripOptions struct {
	OriginID      string
	OriginName    string
	OriginCoord   [2]float64 // [lat, lon]
	DestID        string
	DestName      string
	DestCoord     [2]float64
	NumTrips      int
	Language      string // "sv" or "en"
	MaxChanges    int    // -1 = unset
	RouteType     string // "leasttime", "leastinterchange", "leastwalking"
}

// PlanTrip plans a journey between two locations.
func (c *Client) PlanTrip(ctx context.Context, opts TripOptions) (*model.JourneyResponse, error) {
	params := url.Values{}

	// Origin
	if opts.OriginID != "" {
		params.Set("type_origin", "any")
		params.Set("name_origin", opts.OriginID)
	} else if opts.OriginCoord[0] != 0 {
		params.Set("type_origin", "coord")
		params.Set("name_origin", fmt.Sprintf("%f:%f:WGS84", opts.OriginCoord[0], opts.OriginCoord[1]))
	} else if opts.OriginName != "" {
		params.Set("type_origin", "any")
		params.Set("name_origin", opts.OriginName)
	}

	// Destination
	if opts.DestID != "" {
		params.Set("type_destination", "any")
		params.Set("name_destination", opts.DestID)
	} else if opts.DestCoord[0] != 0 {
		params.Set("type_destination", "coord")
		params.Set("name_destination", fmt.Sprintf("%f:%f:WGS84", opts.DestCoord[0], opts.DestCoord[1]))
	} else if opts.DestName != "" {
		params.Set("type_destination", "any")
		params.Set("name_destination", opts.DestName)
	}

	if opts.NumTrips > 0 {
		params.Set("calc_number_of_trips", strconv.Itoa(opts.NumTrips))
	}
	if opts.Language != "" {
		params.Set("language", opts.Language)
	}
	if opts.MaxChanges >= 0 {
		params.Set("max_changes", strconv.Itoa(opts.MaxChanges))
	}
	if opts.RouteType != "" {
		params.Set("route_type", opts.RouteType)
	}

	u := JourneyPlannerBaseURL + "/trips?" + params.Encode()
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp model.JourneyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing trips: %w", err)
	}
	return &resp, nil
}

// FindAddress searches for addresses/streets/POIs (broader than FindStops).
func (c *Client) FindAddress(ctx context.Context, query string) ([]model.Location, error) {
	params := url.Values{}
	params.Set("name_sf", query)
	params.Set("type_sf", "any")
	params.Set("any_obj_filter_sf", "46") // stops + addresses + POI

	u := JourneyPlannerBaseURL + "/stop-finder?" + params.Encode()
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp model.StopFinderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing stop finder: %w", err)
	}
	return resp.Locations, nil
}
