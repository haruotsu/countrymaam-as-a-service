package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/service"
)

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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/flavors", s.listFlavors)

	r.Route("/users", func(r chi.Router) {
		r.Get("/", s.listUsers)
		r.Post("/", s.createUser)
		r.Get("/{id}", s.getUser)
		r.Get("/{id}/accounts", s.listUserAccounts)
	})

	r.Route("/accounts", func(r chi.Router) {
		r.Post("/", s.openAccount)
		r.Get("/{id}", s.getAccount)
		r.Post("/{id}/deposit", s.deposit)
		r.Post("/{id}/withdraw", s.withdraw)
		r.Get("/{id}/transactions", s.listTransactions)
	})

	r.Post("/transfers", s.transfer)
	r.Post("/exchanges", s.exchange)

	return r
}

// ------- handlers -------

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

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := s.svc.CreateUser(r.Context(), service.CreateUserInput{Name: body.Name, Email: body.Email})
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, userDTO(u))
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.svc.ListUsers(r.Context())
	if err != nil {
		httpError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, userDTO(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := s.svc.GetUser(r.Context(), id)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userDTO(u))
}

func (s *Server) listUserAccounts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	accs, err := s.svc.ListAccountsByUser(r.Context(), id)
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

func (s *Server) openAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
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
	a, err := s.svc.OpenAccount(r.Context(), service.OpenAccountInput{UserID: body.UserID, Flavor: fl})
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, accountDTO(a))
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	a, err := s.svc.GetAccount(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountDTO(a))
}

func (s *Server) deposit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Amount int64  `json:"amount"`
		Memo   string `json:"memo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	a, err := s.svc.Deposit(r.Context(), id, body.Amount, body.Memo)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountDTO(a))
}

func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Amount int64  `json:"amount"`
		Memo   string `json:"memo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	a, err := s.svc.Withdraw(r.Context(), id, body.Amount, body.Memo)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountDTO(a))
}

func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
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
		FromAccountID: body.FromAccountID, ToAccountID: body.ToAccountID,
		Amount: body.Amount, Memo: body.Memo,
	}); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) exchange(w http.ResponseWriter, r *http.Request) {
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
		FromAccountID: body.FromAccountID, ToAccountID: body.ToAccountID,
		Amount: body.Amount, Memo: body.Memo,
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
	id := chi.URLParam(r, "id")
	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	txs, err := s.svc.ListTransactions(r.Context(), id, limit)
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// httpError はドメインエラーを適切な HTTP ステータスにマップする。
func httpError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidUser),
		errors.Is(err, domain.ErrExchangeTooSmall),
		errors.Is(err, domain.ErrFlavorMismatch),
		errors.Is(err, domain.ErrSelfTransfer),
		errors.Is(err, domain.ErrForeignExchange):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrInsufficientBalance):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrAccountNotFound):
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
