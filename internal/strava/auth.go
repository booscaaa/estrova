package strava

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/booscaaa/estrova/internal/db"
	"golang.org/x/oauth2"
)

const (
	authURL  = "https://www.strava.com/oauth/authorize"
	tokenURL = "https://www.strava.com/oauth/token"
	// Strava requires comma-separated scopes
	stravaScope = "read,activity:read_all,profile:read_all"
)

func oauthConfig(clientID, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:8765/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}

func GetAuthURL(clientID string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", "http://localhost:8765/callback")
	params.Set("response_type", "code")
	params.Set("approval_prompt", "auto")
	params.Set("scope", stravaScope)
	return authURL + "?" + params.Encode()
}

func ExchangeCode(ctx context.Context, clientID, clientSecret, code string) (*oauth2.Token, error) {
	cfg := oauthConfig(clientID, clientSecret)
	return cfg.Exchange(ctx, code)
}

func SaveToken(database *db.DB, token *oauth2.Token) error {
	return database.SaveToken(db.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	})
}

func LoadToken(database *db.DB) (*oauth2.Token, error) {
	t, err := database.LoadToken()
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       t.Expiry,
	}, nil
}

// StartCallbackServer starts a local server to receive the OAuth callback.
// Returns the authorization code via channel.
func StartCallbackServer() (codeChan chan string, stopFn func()) {
	codeChan = make(chan string, 1)
	mux := http.NewServeMux()
	srv := &http.Server{Addr: ":8765", Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, `<html><body style="font-family:sans-serif;text-align:center;padding:60px">
				<h2>✅ Autenticação concluída!</h2>
				<p>Pode fechar esta aba e voltar ao Claude.</p>
			</body></html>`)
			codeChan <- code
		} else {
			errMsg := r.URL.Query().Get("error")
			fmt.Fprintf(w, `<html><body style="font-family:sans-serif;text-align:center;padding:60px">
				<h2>❌ Erro na autenticação</h2><p>%s</p>
			</body></html>`, errMsg)
		}
	})

	go func() {
		_ = srv.ListenAndServe()
	}()

	stopFn = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}

	return codeChan, stopFn
}

func RefreshTokenIfNeeded(ctx context.Context, database *db.DB, clientID, clientSecret string, token *oauth2.Token) (*oauth2.Token, error) {
	cfg := oauthConfig(clientID, clientSecret)
	ts := cfg.TokenSource(ctx, token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, err
	}
	if newToken.AccessToken != token.AccessToken {
		if err := SaveToken(database, newToken); err != nil {
			return nil, err
		}
	}
	return newToken, nil
}
