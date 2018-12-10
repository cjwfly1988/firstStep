package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/zmb3/spotify"
)

const redirectURI = "http://localhost:8080/callback"

var (
	client       *spotify.Client
	auth         = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	clientID     = getenv("SPOTIFY_CLIENT_ID")
	clientSecret = getenv("SPOTIFY_CLIENT_SECRET")
	ch           = make(chan *spotify.Client)
	state        = "test"
	html         = `
<br/>
<a href="/player/play">Play</a><br/>
<a href="/player/pause">Pause</a><br/>
<a href="/player/next">Next track</a><br/>
<a href="/player/previous">Previous Track</a><br/>
<a href="/player/shuffle">Shuffle</a><br/>
`
)

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic("missing required environment variable " + name)
	}
	return v
}
func main() {
	auth.SetAuthInfo(clientID, clientSecret)

	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())

	})
	http.HandleFunc("/player/", control)
	go func() {
		url := auth.AuthURL(state)
		fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

		// wait for auth to complete
		client = <-ch

		// use the client to make calls that require authorization
		user, err := client.CurrentUser()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("You are logged in as:", user.ID)

		var playerState *spotify.PlayerState
		playerState, err = client.PlayerState()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
	}()

	http.ListenAndServe(":8080", nil)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	token, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(token)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Login Completed!"+html)
	ch <- &client
}

func control(w http.ResponseWriter, r *http.Request) {
	playerState, err := client.PlayerState()
	if err != nil {
		log.Fatal(err)
	}
	action := strings.TrimPrefix(r.URL.Path, "/player/")
	fmt.Println("Got request for:", action)
	switch action {
	case "play":
		err = client.Play()
	case "pause":
		err = client.Pause()
	case "next":
		err = client.Next()
	case "previous":
		err = client.Previous()
	case "shuffle":
		playerState.ShuffleState = !playerState.ShuffleState
		err = client.Shuffle(playerState.ShuffleState)
	}
	if err != nil {
		log.Print(err)
	}

	w.Header().Set("Content-Type", "text/html")

	var deviceType string
	var deviceName string
	var artists []string
	deviceType = playerState.Device.Type
	deviceName = playerState.Device.Name
	for _, artist := range playerState.CurrentlyPlaying.Item.Artists {
		artists = append(artists, artist.Name)
	}
	fmt.Printf("Found your %s (%s)\n", deviceType, deviceName)
	fmt.Fprint(w, html, artists, " ", deviceName, " ", deviceType)
}
