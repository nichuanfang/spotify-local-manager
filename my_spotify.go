package main

import (
	"net/http"

	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

// 生成授权URL
func generateAuthorizationURL() (authorizationURL string) {
	//生成授权URL
	authScopes := []string{spotifyauth.ScopeUserReadPrivate}
	//认证器初始化
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithClientID(spotifyClientId),
		spotifyauth.WithClientSecret(spotifyClientSecret),
		spotifyauth.WithScopes(authScopes...))
	authorizationURL = auth.AuthURL(state)
	return
}

// 通过code交换token
func exchangeCodeForToken(code string, w http.ResponseWriter, r *http.Request) string {
	return ""
}
