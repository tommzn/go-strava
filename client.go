package strava

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
)

// New returns an api client with BASE_URL as default.
func New(tokenSource oauth2.TokenSource) *Client {
	return &Client{
		baseUrl:     BASE_URL,
		tokenSource: tokenSource,
		httpClient:  &http.Client{},
	}
}

// WithBaseUrl sets the given url as the base for all api calls.
func (client *Client) WithBaseUrl(baseUrl string) {
	client.baseUrl = baseUrl
}

// WithAthleteId assigns the given athlete id. This id will be used for all further requests.
func (client *Client) WithAthleteId(athleteId int64) {
	client.athleteId = &athleteId
}

// AuthorizedAthlete tries to fetch the current athlete, defined by the used
// auth tokens, from Strava.
func (client *Client) AuthorizedAthlete() (*DetailedAthlete, error) {

	req, err := http.NewRequest(http.MethodGet, client.apiEndpoint("/athlete"), nil)
	if err != nil {
		return nil, err
	}
	responseBody, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	detailedAthlete := &DetailedAthlete{}
	if err := json.Unmarshal(responseBody, detailedAthlete); err != nil {
		return nil, err
	}
	return detailedAthlete, nil
}

// AthleteActivities lists available activities for an athlete. You can use
// timeFilter to restrict the time range activities should be requested for.
// Pagination can be used to retrieve activities step by step if there are a
// lot of them.
func (client *Client) AthleteActivities(timeFilter *TimeFilter, pagination *Pagination) (*[]SummaryActivity, error) {

	req, err := http.NewRequest(http.MethodGet, client.apiEndpoint("/athlete/activities"), nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	appendTimeFilter(&query, timeFilter)
	appendPagination(&query, pagination)
	req.URL.RawQuery = query.Encode()

	responseBody, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	summaryActivity := &[]SummaryActivity{}
	if err := json.Unmarshal(responseBody, summaryActivity); err != nil {
		return nil, err
	}
	return summaryActivity, nil
}

// AthleteStats returns summarized athlete stats, related to the current year
// or in total.
func (client *Client) AthleteStats() (*ActivityStats, error) {

	athleteId, err := client.getAthleteId()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, client.apiEndpoint("/athletes/%d/stats", *athleteId), nil)
	if err != nil {
		return nil, err
	}
	responseBody, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	activityStats := &ActivityStats{}
	if err := json.Unmarshal(cleanEmptyStrings(responseBody), activityStats); err != nil {
		return nil, err
	}
	return activityStats, nil
}

// sendRequest adds the auth token and performs the given request against
// Strava's API.
func (client *Client) sendRequest(req *http.Request) ([]byte, error) {

	if err := client.addToken(req); err != nil {
		return nil, err
	}

	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode > 399 {
		return nil, faultResponseAsError(res)
	}
	return io.ReadAll(res.Body)
}

// addToken retrieves an OAuth2 token from the assigned token source and adds
// it as an Authorization header to the given request.
func (client *Client) addToken(req *http.Request) error {
	token, err := client.tokenSource.Token()
	if err != nil {
		return err
	}
	token.SetAuthHeader(req)
	return nil
}

// apiEndpoint prepends the base url to the given api endpoint path.
func (client *Client) apiEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf(client.baseUrl+path, args...)
}

// getAthleteId returns the local athlete id, assigned by the WithAthleteId
// method, or requests it directly from Strava using the AuthorizedAthlete
// method.
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

// faultResponseAsError converts an API response body of type Fault into an error.
func faultResponseAsError(res *http.Response) error {

	responseBody, _ := io.ReadAll(res.Body)

	fault := &Fault{}
	_ = json.Unmarshal(responseBody, fault)

	msg := fmt.Sprintf("%d %s", res.StatusCode, fault.Message)
	if len(fault.Errors) > 0 {
		first := fault.Errors[0]
		msg = fmt.Sprintf("%s: %s %s %s", msg, first.Resource, first.Field, first.Code)
	}
	return errors.New(msg)
}

// appendTimeFilter appends the given time filter to the passed query. Nil
// values for Before and After are skipped.
func appendTimeFilter(query *url.Values, timeFilter *TimeFilter) {
	if timeFilter == nil {
		return
	}
	if timeFilter.Before != nil {
		query.Add("before", strconv.FormatInt(timeFilter.Before.Unix(), 10))
	}
	if timeFilter.After != nil {
		query.Add("after", strconv.FormatInt(timeFilter.After.Unix(), 10))
	}
}

// appendPagination adds the given page or per_page values to the passed query.
// Nil values for page and per_page are skipped.
func appendPagination(query *url.Values, pagination *Pagination) {
	if pagination == nil {
		return
	}
	if pagination.Page != nil {
		query.Add("page", strconv.Itoa(*pagination.Page))
	}
	if pagination.PerPage != nil {
		query.Add("per_page", strconv.Itoa(*pagination.PerPage))
	}
}

// cleanEmptyStrings replaces all occurrences of "" in the given content with
// an empty JSON object {}.
func cleanEmptyStrings(content []byte) []byte {
	return []byte(strings.ReplaceAll(string(content), "\"\"", "{}"))
}
