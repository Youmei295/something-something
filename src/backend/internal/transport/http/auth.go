package http

import (
	"net/http"
	"time"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	User      userDTO   `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req, 8<<10); err != nil {
		writeError(w, err)
		return
	}

	result, err := a.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     a.cfg.Auth.CookieName,
		Value:    result.Session.ID.String(),
		Path:     "/",
		Domain:   a.cfg.Auth.CookieDomain,
		Expires:  result.ExpiresAt,
		HttpOnly: true,
		Secure:   a.cfg.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, loginResponse{User: toUserDTO(result.User), ExpiresAt: result.ExpiresAt})
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	if id, ok := sessionIDFrom(r.Context()); ok {
		_ = a.auth.Logout(r.Context(), id)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     a.cfg.Auth.CookieName,
		Value:    "",
		Path:     "/",
		Domain:   a.cfg.Auth.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.cfg.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(user))
}
