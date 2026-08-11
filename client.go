package strava

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// New returna an api client with BASE_URL as default.
func New(tokenSource oauth2.TokenSource) *Client {
	return &Client{
		baseUrl:     BASE_URL,
		tokenSource: tokenSource,
		httpClient:  &http.Client{},
	}
}

// WithBaseUrl set given url as base for all api calls.
func (client *Client) WithBaseUrl(baseUrl string) {
	client.baseUrl = baseUrl
}

// WithAthleteId assigns given athlete id. This id will be used for all further requests.
func (client *Client) WithAthleteId(athleteId int64) {
	client.athleteId = &athleteId
}

// AuthorizedAthlete try to fetch current athelete, defined by used auth tokens, from Strava.
func (client *Client) AuthorizedAthlete() (*DetailedAthlete, error) {

	req, _ := http.NewRequest("GET", client.apiEndpoint("/athlete"), nil)
	responseBody, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	detailedAthlete := &DetailedAthlete{}
	jsonErr := json.Unmarshal(responseBody, detailedAthlete)
	return detailedAthlete, jsonErr
}

// AthleteActivities lists available activities for an athlete.
// You can use timeFilter to retrice time range activities should be requested for. Pagination param can be
// used if retrieve activities step by step if there're a lot of them.
func (client *Client) AthleteActivities(timeFilter *TimeFilter, pagination *Pagination) (*[]SummaryActivity, error) {

	req, _ := http.NewRequest("GET", client.apiEndpoint("/athlete/activities"), nil)
	query := req.URL.Query()
	appendTimeFilter(&query, timeFilter)
	appendPagination(&query, pagination)
	req.URL.RawQuery = query.Encode()

	responseBody, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	summaryActivity := &[]SummaryActivity{}
	jsonErr := json.Unmarshal(responseBody, summaryActivity)
	return summaryActivity, jsonErr
}

// AthleteStats returns summarited athlete stats, related to current year or in total.
func (client *Client) AthleteStats() (*ActivityStats, error) {

	athleteId, err := client.getAthleteId()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", client.apiEndpoint("/athletes/%d/stats", *athleteId), nil)
	responseBody, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	activityStats := &ActivityStats{}
	jsonErr := json.Unmarshal(cleanEmptyStrings(responseBody), activityStats)
	return activityStats, jsonErr
}

// SendRequest will add auth token and performs given request to Strava's API.
func (client *Client) sendRequest(req *http.Request) ([]byte, error) {

	if err := client.addToken(req); err != nil {
		return []byte{}, err
	}

	res, err := client.httpClient.Do(req)
	if err != nil {
		return []byte{}, err
	}

	if res.StatusCode > 399 {
		return []byte{}, faultReponseAsError(res)
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// AddToken ewtrieves an OAuth2 token from assigned token source and
// add it as Authorization header to given request.
func (client *Client) addToken(req *http.Request) error {
	token, err := client.tokenSource.Token()
	if err != nil {
		return err
	}
	token.SetAuthHeader(req)
	return nil
}

// ApiEndpoint add base url prefix to given api endpoint.
func (client *Client) apiEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf(client.baseUrl+path, args...)
}

// GetAthleteId will return local athlete id, assigned by WithAthleteId method, or
// try to request it directly from Strave using AuthorizedAthlete method.
func (client *Client) getAthleteId() (*int64, error) {
	if client.athleteId == nil {
		detailedAthlete, err := client.AuthorizedAthlete()
		if err != nil {
			return nil, err
		}
		client.athleteId = &detailedAthlete.Id
	}
	return client.athleteId, nil
}

// FaultReponseAsError converts API response body of type Fault into an error.
func faultReponseAsError(res *http.Response) error {

	defer res.Body.Close()
	responseBody, _ := io.ReadAll(res.Body)

	fault := &Fault{}
	json.Unmarshal(responseBody, fault)
	errMsg := fmt.Sprintf("%d %s", res.StatusCode, fault.Message)
	if len(fault.Errors) > 0 {
		errMsg = fmt.Sprintf("%s: %s %s %s", errMsg, fault.Errors[0].Resource, fault.Errors[0].Field, fault.Errors[0].Code)
	}
	return errors.New(errMsg)
}

// AppendTimeFilter appends given time filter to passed query.
// Nil values for Before and After will be skipped.
func appendTimeFilter(query *url.Values, timeFilter *TimeFilter) {
	if timeFilter != nil {
		if timeFilter.Before != nil {
			query.Add("before", fmt.Sprintf("%d", timeFilter.Before.Unix()))
		}
		if timeFilter.After != nil {
			query.Add("after", fmt.Sprintf("%d", timeFilter.After.Unix()))
		}
	}
}

// AppendPagination will add given page or per page values to passed query.
// Nil values for page and per page will be skipped.
func appendPagination(query *url.Values, pagination *Pagination) {
	if pagination != nil {
		if pagination.Page != nil {
			query.Add("page", fmt.Sprintf("%d", *pagination.Page))
		}
		if pagination.PerPage != nil {
			query.Add("per_page", fmt.Sprintf("%d", *pagination.PerPage))
		}
	}
}

// CleanEmptyStrings replaces all occurrences of "" in given content with an empty Json object {}.
func cleanEmptyStrings(content []byte) []byte {
	return []byte(strings.ReplaceAll(string(content), "\"\"", "{}"))
}
