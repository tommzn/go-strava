package strava

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// oauthConfigForTest returns an oauth2.Config pointed at the given httptest
// server, using the Strava-style token endpoint.
func oauthConfigForTest(ts *httptest.Server) oauth2.Config {
	return oauth2.Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  ts.URL + "/oauth/authorize",
			TokenURL: ts.URL + "/oauth/token",
		},
	}
}

func TestTokenSourceFromAuthorizationCode(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"acc","refresh_token":"ref","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(ts.Close)

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(ts2.Close)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		src, err := TokenSourceFromAuthorizationCode(oauthConfigForTest(ts), "code")
		require.NoError(t, err)
		require.NotNil(t, src)
		tok, err := src.Token()
		require.NoError(t, err)
		require.Equal(t, "acc", tok.AccessToken)
	})

	t.Run("exchange-error", func(t *testing.T) {
		t.Parallel()
		src, err := TokenSourceFromAuthorizationCode(oauthConfigForTest(ts2), "code")
		require.Error(t, err)
		require.Nil(t, src)
	})
}

func TestTokenSourceFromRefreshToken(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"acc2","refresh_token":"ref2","token_type":"Bearer","expires_in":3600}`))
	}))
	defer ts.Close()

	src, err := TokenSourceFromRefreshToken(oauthConfigForTest(ts), "some-refresh-token")
	require.NoError(t, err)
	require.NotNil(t, src)

	tok, err := src.Token()
	require.NoError(t, err)
	require.Equal(t, "acc2", tok.AccessToken)
}
