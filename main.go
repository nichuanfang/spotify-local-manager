package main

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"
	"github.com/zmb3/spotify/v2/auth"
)

func main() {
	spotifyauth.New(spotifyauth.WithRedirectURL())
}
