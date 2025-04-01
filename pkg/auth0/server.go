package auth0

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bluexlab/bxauth0/pkg/helper/httputil"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

type Server struct {
	privateKey   *rsa.PrivateKey
	publicKey    *rsa.PublicKey
	signer       jose.Signer
	endpoint     string
	clientID     string
	clientSecret string
	clientEmail  string
	host         string
	port         int
	*http.Server
}

func NewServer(opts ...Option) *Server {
	privateKey, signer := newSigner()
	s := &Server{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		signer:     signer,
		endpoint:   "http://localhost:3000",
		host:       "0.0.0.0",
		port:       3000,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Server) Run(ctx context.Context) error {
	hostPort := net.JoinHostPort(s.host, strconv.Itoa(s.port))
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer func(c io.Closer) { _ = c.Close() }(ln)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", s.loginGet)
	mux.HandleFunc("POST /login", s.loginPost)
	mux.HandleFunc("GET /authorize", s.authorizeGet)
	mux.HandleFunc("POST /authorize", s.authorizePost)
	mux.HandleFunc("GET /.well-known/openid-configuration", s.openidConfigGet)
	mux.HandleFunc("GET /.well-known/jwks.json", s.jwksGet)
	mux.HandleFunc("POST /token", s.tokenPost)

	s.Server = &http.Server{
		Handler:           httputil.LogHandler(mux),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	return s.Server.Serve(ln)
}

func (s *Server) Stop(ctx context.Context) error {
	s.Server.SetKeepAlivesEnabled(false)
	return s.Server.Shutdown(ctx)
}

func (s *Server) loginGet(w http.ResponseWriter, r *http.Request) {
}

func (s *Server) loginPost(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) authorizeGet(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{
		"client_id":     r.URL.Query().Get("client_id"),
		"response_type": r.URL.Query().Get("response_type"),
		"redirect_uri":  lo.CoalesceOrEmpty(r.URL.Query().Get("redirect_uri"), r.Header.Get("Referer")),
		"prompt":        r.URL.Query().Get("prompt"),
		"scope":         r.URL.Query().Get("scope"),
		"state":         r.URL.Query().Get("state"),
		"auth0Client":   r.URL.Query().Get("auth0Client"),
	}
	if params["redirect_uri"] == "" {
		http.Error(w, "Missing redirect_uri", http.StatusBadRequest)
		return
	}
	if params["client_id"] == "" {
		http.Error(w, "Missing client_id", http.StatusBadRequest)
		return
	}
	if params["client_id"] != s.clientID {
		http.Error(w, "Invalid client_id", http.StatusBadRequest)
		return
	}
	if params["response_type"] != "code" {
		http.Error(w, "Unsupported response_type", http.StatusBadRequest)
		return
	}

	redirectUri, _ := url.Parse(params["redirect_uri"])
	values := redirectUri.Query()
	values.Add("code", "code")
	values.Add("state", params["state"])
	values.Add("oauthServiceName", "Auth0")
	redirectUri.RawQuery = values.Encode()

	http.Redirect(w, r, redirectUri.String(), http.StatusFound)
}

func (s *Server) authorizePost(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) openidConfigGet(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"issuer":                                s.endpoint + "/",
		"authorization_endpoint":                s.endpoint + "/authorize",
		"token_endpoint":                        s.endpoint + "/token",
		"device_authorization_endpoint":         s.endpoint + "/device/authorize",
		"jwks_uri":                              s.endpoint + "/.well-known/jwks.json",
		"userinfo_endpoint":                     s.endpoint + "/userinfo",
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_signing_alg_values_supported": []string{"RS256"},
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err := buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) jwksGet(w http.ResponseWriter, r *http.Request) {
	jwksInfo := map[string]interface{}{
		"keys": []jose.JSONWebKey{
			{
				Key:       s.publicKey,
				Use:       "sig",
				Algorithm: "RS256",
			},
		},
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(jwksInfo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err := buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type tokenClaims struct {
	jwt.Claims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (s *Server) tokenPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logrus.Debugf("Failed to parse form: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !s.isAuthorized(r) {
		logrus.Debugf("Unauthorized")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		logrus.Debugf("Unsupported grant_type: %s", grantType)
		http.Error(w, "Unsupported grant_type", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		logrus.Debugf("Missing code")
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	now := time.Now()
	email := s.clientEmail
	claim := tokenClaims{
		Claims: jwt.Claims{
			Issuer:    s.endpoint + "/",
			Subject:   genOpenID(email),
			Audience:  jwt.Audience{s.clientID},
			Expiry:    jwt.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        "",
		},
		Email:         email,
		EmailVerified: true,
	}
	idToken, err := jwt.Signed(s.signer).Claims(claim).Serialize()
	if err != nil {
		logrus.Debugf("Failed to sign token: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := map[string]interface{}{
		"access_token":  "access_token",
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "refresh_token",
		"id_token":      idToken,
		//"error":             "",
		//"error_description": "",
		//"error_uri":         "",
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) isAuthorized(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		username = r.FormValue("client_id")
		password = r.FormValue("client_secret")
	}

	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	expectedUsernameHash := sha256.Sum256([]byte(s.clientID))
	expectedPasswordHash := sha256.Sum256([]byte(s.clientSecret))
	usernameMatched := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
	passwordMatched := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1

	if usernameMatched && passwordMatched {
		return true
	}
	return false
}

func newSigner() (*rsa.PrivateKey, jose.Signer) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	signKey := jose.SigningKey{Algorithm: jose.RS256, Key: key}
	signOptions := jose.SignerOptions{EmbedJWK: true}
	signOptions.WithType("JWT")
	signer, err := jose.NewSigner(signKey, &signOptions)
	if err != nil {
		panic(err)
	}
	return key, signer
}

func genOpenID(email string) string {
	m := sha256.New()
	m.Write([]byte(strings.TrimSpace(strings.ToLower(email))))
	return base58.Encode(m.Sum(nil))
}
