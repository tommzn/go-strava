package strava

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) TestAssignAthleteId() {

	client := clientForTest()

	var athleteId int64 = 12345567890
	client.WithAthleteId(athleteId)
	suite.Equal(athleteId, *client.athleteId)
}

func (suite *ClientTestSuite) TestFormatApiEndpoint() {

	client := clientForTest()

	suite.Equal(BASE_URL+"/xxx", client.apiEndpoint("/xxx"))
	suite.Equal(BASE_URL+"/xxx/12345/yyy", client.apiEndpoint("/xxx/%s/yyy", "12345"))
}

func (suite *ClientTestSuite) TestAppendTimeFilter() {

	query := url.Values{}
	appendTimeFilter(&query, nil)
	suite.Equal("", query.Encode())

	before := time.Unix(1667694696, 0).UTC()
	after := time.Unix(1667694696, 0).UTC()
	after = after.Add(-24 * time.Hour)
	timeFilter := &TimeFilter{Before: &before, After: &after}
	appendTimeFilter(&query, timeFilter)
	suite.Equal("after=1667608296&before=1667694696", query.Encode())
}

func (suite *ClientTestSuite) TestAuthorizedAthlete() {

	client := clientForTest()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "fixtures/authorizedAthlete.json")
	}))
	defer ts.Close()
	client.WithBaseUrl(ts.URL)

	detailedAthlete, err := client.AuthorizedAthlete()
	suite.Nil(err)
	suite.NotNil(detailedAthlete)
	suite.Equal(int64(1234567890987654321), detailedAthlete.Id)
}

func (suite *ClientTestSuite) TestAuthorizedAthleteWithError() {

	client := clientForTest()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveError(w, r)
	}))
	defer ts.Close()
	client.WithBaseUrl(ts.URL)

	detailedAthlete, err := client.AuthorizedAthlete()
	suite.NotNil(err)
	suite.Nil(detailedAthlete)
}

func (suite *ClientTestSuite) TestAthleteActivities() {

	client := clientForTest()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "fixtures/athleteactivities.json")
	}))
	defer ts.Close()
	client.WithBaseUrl(ts.URL)

	athleteActivities, err := client.AthleteActivities(nil, nil)
	suite.Nil(err)
	suite.NotNil(athleteActivities)
	suite.Len(*athleteActivities, 2)
}

func (suite *ClientTestSuite) TestAthleteActivitiesWithError() {

	client := clientForTest()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveError(w, r)
	}))
	defer ts.Close()
	client.WithBaseUrl(ts.URL)

	athleteActivities, err := client.AthleteActivities(nil, nil)
	suite.NotNil(err)
	suite.Nil(athleteActivities)
}

func (suite *ClientTestSuite) TestAthleteStats() {

	client := clientForTest()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "fixtures/athletestats.json")
	}))
	defer ts.Close()
	client.WithBaseUrl(ts.URL)

	activityStats, err := client.AthleteStats()
	suite.Nil(err)
	suite.NotNil(activityStats)
	suite.True(activityStats.RecentRideTotals.Distance > 0)
}

func (suite *ClientTestSuite) TestAthleteStatsWithError() {

	client := clientForTest()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveError(w, r)
	}))
	defer ts.Close()
	client.WithBaseUrl(ts.URL)
	client.WithAthleteId(1234567890)

	activityStats, err := client.AthleteStats()
	suite.NotNil(err)
	suite.Nil(activityStats)
}

func (suite *ClientTestSuite) TestAppendPagination() {

	query := url.Values{}
	appendPagination(&query, nil)
	suite.Equal("", query.Encode())

	appendPagination(&query, NewPagination(0, 0))
	suite.Equal("", query.Encode())

	pagination := NewPagination(3, 25)
	appendPagination(&query, pagination)
	suite.Equal("page=3&per_page=25", query.Encode())

	query = url.Values{}
	pagination.NextPage()
	appendPagination(&query, pagination)
	suite.Equal("page=4&per_page=25", query.Encode())
}

func (suite *ClientTestSuite) TestTokenSourceError() {

	client := clientForTest()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "fixtures/athletestats.json")
	}))
	defer ts.Close()
	client.WithBaseUrl(ts.URL)
	client.tokenSource = newTokenSourceMock(true)

	activityStats, err := client.AthleteStats()
	suite.NotNil(err)
	suite.Nil(activityStats)
}

func (suite *ClientTestSuite) TestRequestError() {

	client := clientForTest()
	client.WithBaseUrl("http://127.0.0.1:59524")

	detailedAthlete, err := client.AuthorizedAthlete()
	suite.NotNil(err)
	suite.Nil(detailedAthlete)
}

func clientForTest() *Client {
	return New(newTokenSourceMock(false))
}

func serveFile(w http.ResponseWriter, r *http.Request, filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}

func serveError(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("fixtures/fault.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}
