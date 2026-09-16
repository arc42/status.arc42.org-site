package auth

import (
	"embed"
	"html/template"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

const (
	sessionLifetime = 8 * time.Hour
	stateLifetime   = 10 * time.Minute
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

var pageTemplate = template.Must(template.ParseFS(templatesFS, "templates/authMessage.gohtml"))

// Auth is the login in front of protected pages.
type Auth struct {
	cfg    Config
	oauth  *oauth2.Config
	client *http.Client
	now    func() time.Time
}

// New builds the login from a configuration. An unusable configuration is
// not an error here: every handler answers 503 instead (Config.Problem).
func New(cfg Config) *Auth {
	return &Auth{
		cfg: cfg,
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:   cfg.AuthorizeURL,
				TokenURL:  cfg.TokenURL,
				AuthStyle: oauth2.AuthStyleInParams,
			},
			RedirectURL: cfg.PublicBaseURL + "/auth/callback",
			// no Scopes: the push check reads a public repository only
		},
		client: &http.Client{Timeout: 10 * time.Second},
		now:    time.Now,
	}
}
