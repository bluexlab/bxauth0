package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

type Store struct {
	name  string
	store *sessions.CookieStore
}

func NewStore(name string) *Store {
	store := sessions.NewCookieStore([]byte("secret"))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
	}
	return &Store{name: name, store: store}
}

func (s *Store) Get(r *http.Request, name string) (interface{}, error) {
	session, err := s.store.Get(r, s.name)
	if err != nil {
		return nil, err
	}

	return session.Values[name], nil
}

func (s *Store) Set(w http.ResponseWriter, r *http.Request, name string, value interface{}) error {
	session, err := s.store.Get(r, s.name)
	if err != nil {
		return err
	}
	session.Values[name] = value
	return session.Save(r, w)
}

func (s *Store) Delete(w http.ResponseWriter, r *http.Request, name string) error {
	session, err := s.store.Get(r, s.name)
	if err != nil {
		return err
	}
	delete(session.Values, name)
	return session.Save(r, w)
}
