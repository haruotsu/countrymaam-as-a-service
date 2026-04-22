package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/repository"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/service"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	svc := service.New(repository.NewMemoryStore())
	return NewServer(svc).Router()
}

// do は cookie jar を自前で持つ小さなクライアント。
type client struct {
	h       http.Handler
	cookies map[string]string
}

// newClient は新規サーバで新規クライアントを作る。
func newClient(t *testing.T) *client {
	return &client{h: newTestServer(t), cookies: map[string]string{}}
}

// anotherClient は同じサーバに対する別セッションのクライアントを作る。
// 同一テスト内で複数ユーザーが同じ世界に同居する状況を作るのに使う。
func anotherClient(h http.Handler) *client {
	return &client{h: h, cookies: map[string]string{}}
}

func (c *client) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	for k, v := range c.cookies {
		r.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	for _, sc := range w.Result().Cookies() {
		if sc.MaxAge < 0 || sc.Value == "" {
			delete(c.cookies, sc.Name)
		} else {
			c.cookies[sc.Name] = sc.Value
		}
	}
	return w
}

func parseObj(w *httptest.ResponseRecorder) map[string]any {
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func parseArr(w *httptest.ResponseRecorder) []map[string]any {
	var m []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func TestE2E_AuthHappyPath(t *testing.T) {
	c := newClient(t)

	// 未認証は 401
	if w := c.do(t, "GET", "/auth/me", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}

	// Register → 自動でセッションCookieが付く
	w := c.do(t, "POST", "/auth/register", `{"name":"Alice","email":"alice@x.com","password":"password-1"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	if _, ok := c.cookies[sessionCookieName]; !ok {
		t.Fatal("session cookie should be set")
	}

	// /auth/me が自分を返す
	me := parseObj(c.do(t, "GET", "/auth/me", ""))
	if me["email"] != "alice@x.com" {
		t.Fatalf("me: %v", me)
	}

	// 口座開設
	w = c.do(t, "POST", "/accounts", `{"flavor":"vanilla"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("open: %d %s", w.Code, w.Body.String())
	}
	acc := parseObj(w)
	accID := acc["id"].(string)

	// 入金→残高確認
	w = c.do(t, "POST", "/accounts/"+accID+"/deposit", `{"amount":10}`)
	if w.Code != http.StatusOK {
		t.Fatalf("deposit: %d", w.Code)
	}

	// ログアウト
	_ = c.do(t, "POST", "/auth/logout", "")
	if w := c.do(t, "GET", "/auth/me", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout: want 401, got %d", w.Code)
	}
}

func TestE2E_Forbidden_OthersAccount(t *testing.T) {
	ca := newClient(t)
	_ = ca.do(t, "POST", "/auth/register", `{"name":"A","email":"a@x.com","password":"password-1"}`)
	wa := ca.do(t, "POST", "/accounts", `{"flavor":"vanilla"}`)
	aID := parseObj(wa)["id"].(string)

	cb := anotherClient(ca.h)
	_ = cb.do(t, "POST", "/auth/register", `{"name":"B","email":"b@x.com","password":"password-1"}`)

	// B が A の口座を見ようとする → 403
	if w := cb.do(t, "GET", "/accounts/"+aID, ""); w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", w.Code, w.Body.String())
	}
	// B が A の口座に入金しようとする → 403
	if w := cb.do(t, "POST", "/accounts/"+aID+"/deposit", `{"amount":1}`); w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
	// B が A の口座の履歴を見る → 403
	if w := cb.do(t, "GET", "/accounts/"+aID+"/transactions", ""); w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestE2E_TransferByEmailLookup(t *testing.T) {
	// A と B が同じ vanilla 口座を持っている。A が B にメールで送金する。
	ca := newClient(t)
	_ = ca.do(t, "POST", "/auth/register", `{"name":"A","email":"a@x.com","password":"password-1"}`)
	aAcc := parseObj(ca.do(t, "POST", "/accounts", `{"flavor":"vanilla"}`))
	_ = ca.do(t, "POST", "/accounts/"+aAcc["id"].(string)+"/deposit", `{"amount":20}`)

	cb := anotherClient(ca.h)
	_ = cb.do(t, "POST", "/auth/register", `{"name":"B","email":"b@x.com","password":"password-1"}`)
	bAcc := parseObj(cb.do(t, "POST", "/accounts", `{"flavor":"vanilla"}`))

	// A が /accounts/search で B の vanilla 口座を解決
	w := ca.do(t, "GET", "/accounts/search?email=b@x.com&flavor=vanilla", "")
	if w.Code != http.StatusOK {
		t.Fatalf("search: %d %s", w.Code, w.Body.String())
	}
	body := parseObj(w)
	resolved, _ := body["account"].(map[string]any)
	if resolved["id"].(string) != bAcc["id"].(string) {
		t.Fatalf("resolved account id mismatch")
	}

	// A が送金
	w = ca.do(t, "POST", "/transfers",
		`{"from_account_id":"`+aAcc["id"].(string)+`","to_account_id":"`+bAcc["id"].(string)+`","amount":5}`)
	if w.Code != http.StatusOK {
		t.Fatalf("transfer: %d %s", w.Code, w.Body.String())
	}

	// B の残高確認（自分でログインして）
	w = cb.do(t, "GET", "/accounts/"+bAcc["id"].(string), "")
	if int64(parseObj(w)["balance"].(float64)) != 5 {
		t.Fatalf("B balance wrong: %v", parseObj(w))
	}
}

func TestE2E_WeakPassword(t *testing.T) {
	c := newClient(t)
	w := c.do(t, "POST", "/auth/register", `{"name":"A","email":"a@x.com","password":"short"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

func TestE2E_MyAccountsList(t *testing.T) {
	c := newClient(t)
	_ = c.do(t, "POST", "/auth/register", `{"name":"A","email":"a@x.com","password":"password-1"}`)
	_ = c.do(t, "POST", "/accounts", `{"flavor":"vanilla"}`)
	_ = c.do(t, "POST", "/accounts", `{"flavor":"chocolate"}`)
	list := parseArr(c.do(t, "GET", "/accounts/me", ""))
	if len(list) != 2 {
		t.Fatalf("want 2, got %d", len(list))
	}
}
