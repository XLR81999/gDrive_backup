package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/skratchdot/open-golang/open"
	"sync"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type AuthHandler struct {
	config oauth2.Config
	wg *sync.WaitGroup
	tokenFile string
	token oauth2.Token
	ctx context.Context
}

func FetchDriveService() (*drive.Service, error) {
	// Replace with your client ID, client secret, and redirect URI
	ctx := context.Background()
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURL := "http://localhost:8080/auth/callback"
	tokenFile := "token.json"

	// Set up OAuth 2.0 configuration
	config := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
		Endpoint: google.Endpoint,
		RedirectURL: redirectURL,
	}

	//New Auth struct
	authHandler := AuthHandler{config: *config, ctx: ctx, tokenFile: tokenFile}

	// Check if token file exists
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

	// If token is doesn't exist or refresh Token is missing, initiate OAuth flow
	if token == nil ||  (time.Now().After(token.Expiry) && token.RefreshToken == "") {
		var wg sync.WaitGroup
		authHandler.wg = &wg
		authHandler.wg.Add(2)
		
		// Define Routes
		http.HandleFunc("/login", authHandler.handleLogin)
		http.HandleFunc("/auth/callback", authHandler.handleCallback)
		http.HandleFunc("/success", handleSuccess)

		fmt.Printf("Redirecting to login page to get access token via OAuth\n\n\n")
		time.Sleep(3 * time.Second)

		err := open.Run("http://localhost:8080/login")
		if err != nil{
			fmt.Printf("Could not open login url in browser, closing app....")
			time.Sleep(5*time.Second)
			os.Exit(1)
		}

		// Start the web server
		go func() {
			log.Fatal(http.ListenAndServe(":8080", nil))
		}()
		time.Sleep(7 * time.Second)
		authHandler.wg.Wait()
	}else{
		authHandler.token = *token
	}

	// Path to Service Account Key
	client := config.Client(ctx, &authHandler.token)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	return service, err
}

func (authHandler *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	defer authHandler.wg.Done()
	url := authHandler.config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleSuccess(w http.ResponseWriter, _ *http.Request){
	fmt.Fprintf(w, "Authorization successful!\n") 
	fmt.Fprintf(w, "Go back to application to continue backing up")
}

func (authHandler *AuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	defer authHandler.wg.Done()
	code := r.FormValue("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return
		}

		// Exchange the authorization code for tokens
		token, err := authHandler.config.Exchange(authHandler.ctx, code)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
			return
		}

		authHandler.token = *token

		// Save the token to a file for future use
		f, err := os.Create(authHandler.tokenFile)
		if err != nil {
			log.Fatalf("Unable to cache token: %v", err)
		}
		defer f.Close()
		json.NewEncoder(f).Encode(token)

		http.Redirect(w, r, "/success", http.StatusFound)
	}