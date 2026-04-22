package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/service"
)

const sessionCookieName = "cmaas_session"

type ctxKey int

const ctxUserKey ctxKey = 0

type Server struct {
	svc *service.Service
}

func NewServer(svc *service.Service) *Server { return &Server{svc: svc} }

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	origins := []string{"http://localhost:3900"}
	if o := os.Getenv("ALLOWED_ORIGIN"); o != "" {
		origins = strings.Split(o, ",")
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true, // セッションCookieをやり取りするため必須
		MaxAge:           300,
	}))

	// public
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/flavors", s.listFlavors)
	r.Post("/auth/register", s.register)
	r.Post("/auth/login", s.login)
	r.Post("/auth/logout", s.logout)

	// protected
	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Get("/auth/me", s.me)

		r.Post("/accounts", s.openAccount)
		r.Get("/accounts/me", s.listMyAccounts)
		r.Get("/accounts/search", s.searchAccount)
		r.Get("/accounts/{id}", s.getAccount)
		r.Post("/accounts/{id}/deposit", s.deposit)
		r.Post("/accounts/{id}/withdraw", s.withdraw)
		r.Get("/accounts/{id}/transactions", s.listTransactions)
		r.Post("/transfers", s.transfer)
		r.Post("/exchanges", s.exchange)
	})

	return r
}

// ------- middleware -------

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err != nil || c.Value == "" {
			writeErr(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		u, err := s.svc.Authenticate(r.Context(), c.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func viewerFrom(r *http.Request) *domain.User {
	u, _ := r.Context().Value(ctxUserKey).(*domain.User)
	return u
}

// ------- handlers: public -------

func (s *Server) listFlavors(w http.ResponseWriter, r *http.Request) {
	type flavorDTO struct {
		Key   string `json:"key"`
		Label string `json:"label"`
		Rate  string `json:"rate"`
	}
	labels := map[domain.Flavor]string{
		domain.FlavorVanilla:   "バニラ",
		domain.FlavorChocolate: "チョコ",
		domain.FlavorMatcha:    "抹茶",
	}
	rates := map[domain.Flavor]string{
		domain.FlavorVanilla:   "1.0",
		domain.FlavorChocolate: "1.2",
		domain.FlavorMatcha:    "1.5",
	}
	out := make([]flavorDTO, 0, len(domain.AllFlavors))
	for _, f := range domain.AllFlavors {
		out = append(out, flavorDTO{Key: string(f), Label: labels[f], Rate: rates[f]})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, sess, err := s.svc.Register(r.Context(), service.RegisterInput{
		Name: body.Name, Email: body.Email, Password: body.Password,
	})
	if err != nil {
		httpError(w, err)
		return
	}
	setSessionCookie(w, sess)
	writeJSON(w, http.StatusCreated, userDTO(u))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, sess, err := s.svc.Login(r.Context(), service.LoginInput{Email: body.Email, Password: body.Password})
	if err != nil {
		httpError(w, err)
		return
	}
	setSessionCookie(w, sess)
	writeJSON(w, http.StatusOK, userDTO(u))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		_ = s.svc.Logout(r.Context(), c.Value)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ------- handlers: protected -------

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	writeJSON(w, http.StatusOK, userDTO(u))
}

func (s *Server) listMyAccounts(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	accs, err := s.svc.ListMyAccounts(r.Context(), u.ID)
	if err != nil {
		httpError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(accs))
	for _, a := range accs {
		out = append(out, accountDTO(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) searchAccount(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	flavorStr := r.URL.Query().Get("flavor")
	if email == "" || flavorStr == "" {
		writeErr(w, http.StatusBadRequest, "email and flavor are required")
		return
	}
	flavor, err := domain.ParseFlavor(flavorStr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	a, u, err := s.svc.FindAccountByEmailAndFlavor(r.Context(), email, flavor)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"account":    accountDTO(a),
		"user_name":  u.Name,
		"user_email": u.Email,
	})
}

func (s *Server) openAccount(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	var body struct {
		Flavor string `json:"flavor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	fl, err := domain.ParseFlavor(body.Flavor)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	a, err := s.svc.OpenAccount(r.Context(), u.ID, fl)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, accountDTO(a))
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	a, err := s.svc.GetMyAccount(r.Context(), u.ID, chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountDTO(a))
}

func (s *Server) deposit(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	var body struct {
		Amount int64  `json:"amount"`
		Memo   string `json:"memo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	a, err := s.svc.Deposit(r.Context(), u.ID, chi.URLParam(r, "id"), body.Amount, body.Memo)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountDTO(a))
}

func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	var body struct {
		Amount int64  `json:"amount"`
		Memo   string `json:"memo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	a, err := s.svc.Withdraw(r.Context(), u.ID, chi.URLParam(r, "id"), body.Amount, body.Memo)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountDTO(a))
}

func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	var body struct {
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		Amount        int64  `json:"amount"`
		Memo          string `json:"memo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.svc.Transfer(r.Context(), service.TransferInput{
		ViewerID:      u.ID,
		FromAccountID: body.FromAccountID,
		ToAccountID:   body.ToAccountID,
		Amount:        body.Amount,
		Memo:          body.Memo,
	}); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) exchange(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	var body struct {
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		Amount        int64  `json:"amount"`
		Memo          string `json:"memo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	res, err := s.svc.Exchange(r.Context(), service.ExchangeInput{
		ViewerID:      u.ID,
		FromAccountID: body.FromAccountID,
		ToAccountID:   body.ToAccountID,
		Amount:        body.Amount,
		Memo:          body.Memo,
	})
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from_amount": res.FromAmount,
		"to_amount":   res.ToAmount,
	})
}

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	u := viewerFrom(r)
	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	txs, err := s.svc.ListMyTransactions(r.Context(), u.ID, chi.URLParam(r, "id"), limit)
	if err != nil {
		httpError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(txs))
	for _, t := range txs {
		out = append(out, txDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

// ------- helpers -------

func setSessionCookie(w http.ResponseWriter, sess *domain.Session) {
	secure := os.Getenv("SESSION_COOKIE_SECURE") == "1"
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func httpError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidUser),
		errors.Is(err, domain.ErrExchangeTooSmall),
		errors.Is(err, domain.ErrFlavorMismatch),
		errors.Is(err, domain.ErrSelfTransfer),
		errors.Is(err, domain.ErrForeignExchange),
		errors.Is(err, domain.ErrWeakPassword):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrInsufficientBalance):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeErr(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrUnauthenticated):
		writeErr(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeErr(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrAccountNotFound), errors.Is(err, domain.ErrSessionNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrDuplicateUserEmail), errors.Is(err, domain.ErrDuplicateAccount):
		writeErr(w, http.StatusConflict, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

func userDTO(u *domain.User) map[string]any {
	return map[string]any{
		"id":         u.ID,
		"name":       u.Name,
		"email":      u.Email,
		"created_at": u.CreatedAt,
	}
}

func accountDTO(a *domain.Account) map[string]any {
	return map[string]any{
		"id":         a.ID,
		"user_id":    a.UserID,
		"flavor":     string(a.Flavor),
		"balance":    a.Balance,
		"created_at": a.CreatedAt,
	}
}

func txDTO(t *domain.Transaction) map[string]any {
	return map[string]any{
		"id":                      t.ID,
		"account_id":              t.AccountID,
		"counterparty_account_id": t.CounterpartyAccountID,
		"type":                    string(t.Type),
		"amount":                  t.Amount,
		"memo":                    t.Memo,
		"created_at":              t.CreatedAt,
	}
}
