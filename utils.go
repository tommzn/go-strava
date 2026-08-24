package strava

import (
	"context"

	"golang.org/x/oauth2"
)

// TokenSourceFromAuthorizationCode is a helper to get a token source for an
// authorization code. See the Strava OAuth guide for details on how to obtain
// an authorization code for your app.
// Please persist the returned refresh token, because an authorization code can
// only be used once.
func TokenSourceFromAuthorizationCode(oauthConfig oauth2.Config, authCode string) (oauth2.TokenSource, error) {
	ctx := context.Background()
	token, err := oauthConfig.Exchange(ctx, authCode)
	if err != nil {
		return nil, err
	}
	return oauthConfig.TokenSource(ctx, token), nil
}

// TokenSourceFromRefreshToken is a helper to create a token source for an
// existing refresh token.
func TokenSourceFromRefreshToken(oauthConfig oauth2.Config, refreshToken string) (oauth2.TokenSource, error) {
	return oauthConfig.TokenSource(context.Background(), &oauth2.Token{RefreshToken: refreshToken}), nil
}
