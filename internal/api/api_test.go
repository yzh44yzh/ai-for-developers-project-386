package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yzh44yzh/bookmeet/internal/api"
	"github.com/yzh44yzh/bookmeet/internal/domain"
)

func newMux() (http.Handler, *fakeUserStore, *fakeMeetingStore) {
	us := newFakeUserStore()
	ms := newFakeMeetingStore()
	return api.NewMux(us, ms), us, ms
}

// do sends a request to h. A nil body sends no body, a string body is sent
// verbatim (for malformed-JSON tests), anything else is JSON-marshaled.
func do(t *testing.T, h http.Handler, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rdr = strings.NewReader(b)
	default:
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, target, rdr)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return out
}

func mustCreateUser(t *testing.T, h http.Handler, email string) string {
	t.Helper()
	rec := do(t, h, "POST", "/users", map[string]any{
		"first_name": "Ada",
		"last_name":  "Lovelace",
		"email":      email,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create user: got %d, body %s", rec.Code, rec.Body)
	}
	return decodeBody(t, rec)["id"].(string)
}

func meetingPayload(ownerID string) map[string]any {
	return map[string]any{
		"owner_id":         ownerID,
		"title":            "Weekly sync",
		"start":            time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		"duration_minutes": 30,
	}
}

func mustCreateMeeting(t *testing.T, h http.Handler, ownerID string) string {
	t.Helper()
	rec := do(t, h, "POST", "/meetings", meetingPayload(ownerID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create meeting: got %d, body %s", rec.Code, rec.Body)
	}
	return decodeBody(t, rec)["id"].(string)
}

func editPayload(title string) map[string]any {
	return map[string]any{
		"title":            title,
		"start":            time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339),
		"duration_minutes": 45,
	}
}

func TestCreateUser(t *testing.T) {
	h, _, _ := newMux()

	rec := do(t, h, "POST", "/users", map[string]any{
		"first_name":  "Ada",
		"last_name":   "Lovelace",
		"description": "First programmer",
		"email":       "ada@example.com",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q", ct)
	}
	body := decodeBody(t, rec)
	if body["id"] == "" || body["first_name"] != "Ada" || body["last_name"] != "Lovelace" ||
		body["description"] != "First programmer" || body["email"] != "ada@example.com" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestCreateUserValidation(t *testing.T) {
	h, _, _ := newMux()

	rec := do(t, h, "POST", "/users", map[string]any{"email": "ada@example.com"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing names: got %d", rec.Code)
	}
	rec = do(t, h, "POST", "/users", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: got %d", rec.Code)
	}
	rec = do(t, h, "POST", "/users", map[string]any{
		"first_name": "Ada", "last_name": "Lovelace", "email": "ada@example.com", "admin": true,
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unknown field: got %d", rec.Code)
	}
	if err := decodeBody(t, rec)["error"]; err == nil || err == "" {
		t.Errorf("error body missing message: %v", rec.Body.String())
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	h, _, _ := newMux()
	mustCreateUser(t, h, "ada@example.com")

	rec := do(t, h, "POST", "/users", map[string]any{
		"first_name": "Ada", "last_name": "Other", "email": "ada@example.com",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("got %d, body %s", rec.Code, rec.Body)
	}
}

func TestGetUser(t *testing.T) {
	h, _, _ := newMux()
	id := mustCreateUser(t, h, "ada@example.com")

	rec := do(t, h, "GET", "/users/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	if body := decodeBody(t, rec); body["email"] != "ada@example.com" {
		t.Errorf("unexpected body: %v", body)
	}

	rec = do(t, h, "GET", "/users/not-a-uuid", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad uuid: got %d", rec.Code)
	}
	rec = do(t, h, "GET", "/users/"+uuid.NewString(), nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown id: got %d", rec.Code)
	}
}

func TestGetUserByEmail(t *testing.T) {
	h, _, _ := newMux()
	mustCreateUser(t, h, "ada@example.com")

	rec := do(t, h, "GET", "/users?email=ada@example.com", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	if body := decodeBody(t, rec); body["first_name"] != "Ada" {
		t.Errorf("unexpected body: %v", body)
	}

	rec = do(t, h, "GET", "/users", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing email param: got %d", rec.Code)
	}
	rec = do(t, h, "GET", "/users?email=nobody@example.com", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown email: got %d", rec.Code)
	}
}

func TestCreateMeeting(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")

	rec := do(t, h, "POST", "/meetings", meetingPayload(ownerID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	body := decodeBody(t, rec)
	if body["id"] == "" || body["owner_id"] != ownerID || body["title"] != "Weekly sync" {
		t.Errorf("unexpected body: %v", body)
	}
	if body["status"] != "draft" {
		t.Errorf("status: got %v, want draft", body["status"])
	}
	if guests, ok := body["guests"].([]any); !ok || len(guests) != 0 {
		t.Errorf("guests: got %v, want empty array", body["guests"])
	}
	if body["duration_minutes"] != 30.0 {
		t.Errorf("duration_minutes: got %v", body["duration_minutes"])
	}
	if body["cancelled_at"] != nil {
		t.Errorf("cancelled_at: got %v, want null", body["cancelled_at"])
	}
}

func TestCreateMeetingErrors(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")

	bad := func(mutate func(map[string]any), want int) {
		t.Helper()
		payload := meetingPayload(ownerID)
		mutate(payload)
		if rec := do(t, h, "POST", "/meetings", payload); rec.Code != want {
			t.Errorf("got %d, want %d, body %s", rec.Code, want, rec.Body)
		}
	}
	bad(func(p map[string]any) { p["owner_id"] = uuid.NewString() }, http.StatusBadRequest)
	bad(func(p map[string]any) { p["owner_id"] = "not-a-uuid" }, http.StatusBadRequest)
	bad(func(p map[string]any) { p["start"] = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339) }, http.StatusBadRequest)
	bad(func(p map[string]any) { p["duration_minutes"] = 0 }, http.StatusBadRequest)
	bad(func(p map[string]any) { p["title"] = "" }, http.StatusBadRequest)
}

func TestListMeetings(t *testing.T) {
	h, _, _ := newMux()

	rec := do(t, h, "GET", "/meetings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("empty list: got %q, want []", body)
	}

	ownerID := mustCreateUser(t, h, "owner@example.com")
	mustCreateMeeting(t, h, ownerID)
	rec = do(t, h, "GET", "/meetings", nil)
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 || list[0]["title"] != "Weekly sync" {
		t.Errorf("unexpected list: %v", list)
	}
}

func TestGetMeeting(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")
	id := mustCreateMeeting(t, h, ownerID)

	rec := do(t, h, "GET", "/meetings/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	if body := decodeBody(t, rec); body["id"] != id {
		t.Errorf("unexpected body: %v", body)
	}

	rec = do(t, h, "GET", "/meetings/not-a-uuid", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad uuid: got %d", rec.Code)
	}
	rec = do(t, h, "GET", "/meetings/"+uuid.NewString(), nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown id: got %d", rec.Code)
	}
}

func TestEditMeeting(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")
	id := mustCreateMeeting(t, h, ownerID)

	rec := do(t, h, "PATCH", "/meetings/"+id, editPayload("Renamed sync"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "GET", "/meetings/"+id, nil)
	if body := decodeBody(t, rec); body["title"] != "Renamed sync" || body["duration_minutes"] != 45.0 {
		t.Errorf("edit not applied: %v", body)
	}

	rec = do(t, h, "PATCH", "/meetings/"+id, map[string]any{"title": "Partial"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("partial body: got %d", rec.Code)
	}
	rec = do(t, h, "PATCH", "/meetings/"+uuid.NewString(), editPayload("Nope"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown id: got %d", rec.Code)
	}
}

func TestEditCancelledMeeting(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")
	id := mustCreateMeeting(t, h, ownerID)
	do(t, h, "POST", "/meetings/"+id+"/cancel", nil)

	if rec := do(t, h, "PATCH", "/meetings/"+id, editPayload("Too late")); rec.Code != http.StatusConflict {
		t.Errorf("got %d, body %s", rec.Code, rec.Body)
	}
}

func TestEditFrozenMeeting(t *testing.T) {
	h, _, ms := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")

	frozen, err := domain.MeetingFromSnapshot(domain.MeetingSnapshot{
		ID:       domain.NewMeetingID(),
		OwnerID:  domain.UserID(ownerID),
		Title:    "Long gone",
		Start:    time.Now().Add(-2 * time.Hour).Truncate(time.Minute),
		Duration: 30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("MeetingFromSnapshot: %v", err)
	}
	if err := ms.Create(t.Context(), frozen); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if rec := do(t, h, "PATCH", "/meetings/"+string(frozen.ID()), editPayload("Too late")); rec.Code != http.StatusConflict {
		t.Errorf("got %d, body %s", rec.Code, rec.Body)
	}
}

func TestCancelMeeting(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")
	id := mustCreateMeeting(t, h, ownerID)

	rec := do(t, h, "POST", "/meetings/"+id+"/cancel", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "GET", "/meetings/"+id, nil)
	body := decodeBody(t, rec)
	if body["status"] != "cancelled" || body["cancelled_at"] == nil {
		t.Errorf("unexpected body: %v", body)
	}

	rec = do(t, h, "POST", "/meetings/"+id+"/cancel", nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("double cancel: got %d", rec.Code)
	}
	rec = do(t, h, "POST", "/meetings/"+uuid.NewString()+"/cancel", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown id: got %d", rec.Code)
	}
}

func TestAddGuest(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")
	guestID := mustCreateUser(t, h, "guest@example.com")
	id := mustCreateMeeting(t, h, ownerID)

	rec := do(t, h, "POST", "/meetings/"+id+"/guests", map[string]any{"user_id": guestID})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "GET", "/meetings/"+id, nil)
	body := decodeBody(t, rec)
	if body["status"] != "scheduled" {
		t.Errorf("status: got %v, want scheduled", body["status"])
	}
	if guests, ok := body["guests"].([]any); !ok || len(guests) != 1 || guests[0] != guestID {
		t.Errorf("guests: got %v", body["guests"])
	}
}

func TestAddGuestErrors(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")
	guestID := mustCreateUser(t, h, "guest@example.com")
	id := mustCreateMeeting(t, h, ownerID)

	rec := do(t, h, "POST", "/meetings/"+id+"/guests", map[string]any{"user_id": ownerID})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("owner as guest: got %d", rec.Code)
	}
	rec = do(t, h, "POST", "/meetings/"+id+"/guests", map[string]any{"user_id": uuid.NewString()})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unknown user: got %d", rec.Code)
	}
	rec = do(t, h, "POST", "/meetings/"+id+"/guests", map[string]any{"user_id": "not-a-uuid"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad uuid: got %d", rec.Code)
	}
	rec = do(t, h, "POST", "/meetings/"+uuid.NewString()+"/guests", map[string]any{"user_id": guestID})
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown meeting: got %d", rec.Code)
	}

	do(t, h, "POST", "/meetings/"+id+"/guests", map[string]any{"user_id": guestID})
	rec = do(t, h, "POST", "/meetings/"+id+"/guests", map[string]any{"user_id": guestID})
	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate guest: got %d", rec.Code)
	}
}

func TestRemoveGuest(t *testing.T) {
	h, _, _ := newMux()
	ownerID := mustCreateUser(t, h, "owner@example.com")
	guestID := mustCreateUser(t, h, "guest@example.com")
	id := mustCreateMeeting(t, h, ownerID)
	do(t, h, "POST", "/meetings/"+id+"/guests", map[string]any{"user_id": guestID})

	rec := do(t, h, "DELETE", "/meetings/"+id+"/guests/"+guestID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d, body %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "GET", "/meetings/"+id, nil)
	body := decodeBody(t, rec)
	if body["status"] != "draft" {
		t.Errorf("status: got %v, want draft after last guest left", body["status"])
	}

	rec = do(t, h, "DELETE", "/meetings/"+id+"/guests/"+guestID, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("not a guest: got %d", rec.Code)
	}
	rec = do(t, h, "DELETE", "/meetings/"+id+"/guests/not-a-uuid", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad uuid: got %d", rec.Code)
	}
}
