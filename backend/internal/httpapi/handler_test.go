package httpapi

import (
	"bytes"
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

func doJSON(t *testing.T, h http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var data map[string]any
	if w.Body.Len() > 0 && strings.HasPrefix(w.Body.String(), "{") {
		_ = json.Unmarshal(w.Body.Bytes(), &data)
	}
	return w, data
}

func doJSONArray(t *testing.T, h http.Handler, method, path, body string) (*httptest.ResponseRecorder, []map[string]any) {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var data []map[string]any
	if w.Body.Len() > 0 && strings.HasPrefix(strings.TrimSpace(w.Body.String()), "[") {
		_ = json.Unmarshal(w.Body.Bytes(), &data)
	}
	return w, data
}

func TestE2E_HappyPath(t *testing.T) {
	h := newTestServer(t)

	// ユーザー作成
	w, u := doJSON(t, h, "POST", "/users", `{"name":"Alice","email":"a@example.com"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: status=%d body=%s", w.Code, w.Body.String())
	}
	userID := u["id"].(string)

	// 口座開設
	w, a := doJSON(t, h, "POST", "/accounts",
		`{"user_id":"`+userID+`","flavor":"vanilla"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("open account: status=%d body=%s", w.Code, w.Body.String())
	}
	accID := a["id"].(string)

	// 入金
	w, a = doJSON(t, h, "POST", "/accounts/"+accID+"/deposit", `{"amount":10,"memo":"start"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("deposit: %d %s", w.Code, w.Body.String())
	}
	if int64(a["balance"].(float64)) != 10 {
		t.Fatalf("balance after deposit: %v", a["balance"])
	}

	// 引出 超過
	w, _ = doJSON(t, h, "POST", "/accounts/"+accID+"/withdraw", `{"amount":100}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 on insufficient, got %d", w.Code)
	}

	// 履歴確認（入金1件のみ）
	_, list := doJSONArray(t, h, "GET", "/accounts/"+accID+"/transactions", "")
	if len(list) != 1 {
		t.Fatalf("want 1 transaction, got %d", len(list))
	}
}

func TestE2E_DuplicateEmail(t *testing.T) {
	h := newTestServer(t)
	_, _ = doJSON(t, h, "POST", "/users", `{"name":"A","email":"dup@example.com"}`)
	w, _ := doJSON(t, h, "POST", "/users", `{"name":"B","email":"dup@example.com"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestE2E_TransferFlavorMismatch(t *testing.T) {
	h := newTestServer(t)
	_, u1 := doJSON(t, h, "POST", "/users", `{"name":"A","email":"a@x.com"}`)
	_, u2 := doJSON(t, h, "POST", "/users", `{"name":"B","email":"b@x.com"}`)
	_, a1 := doJSON(t, h, "POST", "/accounts",
		`{"user_id":"`+u1["id"].(string)+`","flavor":"vanilla"}`)
	_, a2 := doJSON(t, h, "POST", "/accounts",
		`{"user_id":"`+u2["id"].(string)+`","flavor":"chocolate"}`)
	_, _ = doJSON(t, h, "POST", "/accounts/"+a1["id"].(string)+"/deposit", `{"amount":5}`)
	w, _ := doJSON(t, h, "POST", "/transfers",
		`{"from_account_id":"`+a1["id"].(string)+`","to_account_id":"`+a2["id"].(string)+`","amount":1}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}
