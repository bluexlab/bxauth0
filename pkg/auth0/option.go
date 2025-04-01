package auth0

import "regexp"

type Option func(*Server)

func WithHostPort(host string, port int) Option {
	return func(s *Server) {
		s.host = host
		s.port = port
	}
}

var httpPrefix = regexp.MustCompile("^https?://")

func WithEndpoint(endpoint string) Option {
	if !httpPrefix.MatchString(endpoint) {
		endpoint = "https://" + endpoint
	}

	return func(s *Server) {
		s.endpoint = endpoint
	}
}

func WithClient(clientID string, clientSecret string, clientEmail string) Option {
	return func(s *Server) {
		s.clientID = clientID
		s.clientSecret = clientSecret
		s.clientEmail = clientEmail
	}
}
