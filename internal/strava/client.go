package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/booscaaa/estrova/internal/db"
	"golang.org/x/oauth2"
)

const baseURL = "https://www.strava.com/api/v3"

type Client struct {
	httpClient *http.Client
}

func NewClient(ctx context.Context, database *db.DB, clientID, clientSecret string, token *oauth2.Token) (*Client, error) {
	refreshed, err := RefreshTokenIfNeeded(ctx, database, clientID, clientSecret, token)
	if err != nil {
		return nil, fmt.Errorf("falha ao atualizar token: %w", err)
	}

	cfg := oauthConfig(clientID, clientSecret)
	httpClient := cfg.Client(ctx, refreshed)

	return &Client{httpClient: httpClient}, nil
}

func (c *Client) get(path string, params url.Values, out interface{}) error {
	fullURL := baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("strava retornou status %d: %v", resp.StatusCode, errBody)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) GetAthlete() (*Athlete, error) {
	var athlete Athlete
	if err := c.get("/athlete", nil, &athlete); err != nil {
		return nil, err
	}
	return &athlete, nil
}

func (c *Client) GetAthleteStats(athleteID int64) (*AthleteStats, error) {
	var stats AthleteStats
	path := fmt.Sprintf("/athletes/%d/stats", athleteID)
	if err := c.get(path, nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *Client) GetAthleteZones() (*AthleteZones, error) {
	var zones AthleteZones
	if err := c.get("/athlete/zones", nil, &zones); err != nil {
		return nil, err
	}
	return &zones, nil
}

func (c *Client) ListActivities(page, perPage int, before, after int64) ([]ActivitySummary, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		params.Set("per_page", strconv.Itoa(perPage))
	}
	if before > 0 {
		params.Set("before", strconv.FormatInt(before, 10))
	}
	if after > 0 {
		params.Set("after", strconv.FormatInt(after, 10))
	}

	var activities []ActivitySummary
	if err := c.get("/athlete/activities", params, &activities); err != nil {
		return nil, err
	}
	return activities, nil
}

func (c *Client) GetActivity(id int64) (*ActivityDetail, error) {
	var activity ActivityDetail
	path := fmt.Sprintf("/activities/%d", id)
	params := url.Values{}
	params.Set("include_all_efforts", "true")
	if err := c.get(path, params, &activity); err != nil {
		return nil, err
	}
	return &activity, nil
}
