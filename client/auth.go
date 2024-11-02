package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func FetchDriveService() (*drive.Service, error) {
	// Replace with your client ID, client secret, and redirect URI
	ctx := context.Background()
	clientID := "ENTER CLIENT ID"
	clientSecret := "ENTER CLIENT SECRET"
	redirectURL := "http://localhost:8080/auth/callback"

	// Set up OAuth 2.0 configuration
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: redirectURL,
	}

	// Check if token file exists
	tokenFile := "token.json"
	var token *oauth2.Token
	if _, err := os.Stat(tokenFile); err == nil {
		// Read token from file
		f, err := os.Open(tokenFile)
		if err != nil {
			log.Fatalf("Unable to cache token: %v", err)
		}
		defer f.Close()
		token = &oauth2.Token{}
		err = json.NewDecoder(f).Decode(token)
		if err != nil {
			log.Fatalf("Unable to parse token: %v", err)
		}
	}

	// If token is expired or doesn't exist, initiate OAuth flow
	if token == nil || time.Now().After(token.Expiry) {
		var wg sync.WaitGroup
		wg.Add(1)
		// Start a simple web server to handle the callback
		http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.FormValue("code")
			if code == "" {
				http.Error(w, "Missing authorization code", http.StatusBadRequest)
				wg.Done()
				return
			}

			// Exchange the authorization code for tokens
			token, err := config.Exchange(ctx, code)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
				wg.Done()
				return
			}

			// Save the token to a file for future use
			f, err := os.Create(tokenFile)
			if err != nil {
				log.Fatalf("Unable to cache token: %v", err)
			}
			defer f.Close()
			json.NewEncoder(f).Encode(token)

			http.Redirect(w, r, "/", http.StatusFound)
			wg.Done()
		})


		// Redirect user to Google's authorization endpoint
		authURL := config.AuthCodeURL(redirectURL)
		fmt.Printf("Go to the following link in your browser: %v\n", authURL)

		// Start the web server
		go func() {
			log.Fatal(http.ListenAndServe(":8080", nil))
		}()
		wg.Wait()
	}

	// Path to Service Account Key
	// serviceAccountKey := "C:\\Users\\sridh\\Downloads\\creds.json"
	client := config.Client(ctx, token)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	return service, err
}