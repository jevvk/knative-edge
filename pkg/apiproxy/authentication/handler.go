package authentication

import (
	"fmt"
	"net/http"
)

type handler struct {
	next http.Handler
	auth Authenticator
}

func NewHandler(h http.Handler) *handler {
	return &handler{
		next: h,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get(AuthHeader)

	if err := h.auth.Authorize(authHeader); err != nil {
		fmt.Printf("Unauthorized access: %s", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet || r.URL.Path != "/" {
		fmt.Printf("Bad request: %s %s", r.Method, r.URL.Path)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.next.ServeHTTP(w, r)
}
