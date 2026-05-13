package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yeniklas/memos-tui/internal/model"
)

// serve returns a test server that calls handler and closes on cleanup.
func serve(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-token")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// --- extractTags (tested indirectly via ListMemos) ---

func TestExtractTags(t *testing.T) {
	memoJSON := map[string]any{
		"name":        "memos/1",
		"content":     "Hello #world and #golang today",
		"visibility":  "PRIVATE",
		"pinned":      false,
		"createTime":  "2026-04-25T10:00:00Z",
		"updateTime":  "2026-04-25T10:00:00Z",
		"displayTime": "2026-04-25T10:00:00Z",
		"creator":     "users/1",
	}
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"memos":         []any{memoJSON},
			"nextPageToken": "",
		})
	}))

	memos, _, err := c.ListMemos("", "", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	if len(memos) != 1 {
		t.Fatalf("want 1 memo, got %d", len(memos))
	}
	tags := memos[0].Tags
	if len(tags) != 2 {
		t.Fatalf("want 2 tags, got %v", tags)
	}
	if tags[0] != "world" || tags[1] != "golang" {
		t.Errorf("tags = %v, want [world golang]", tags)
	}
}

func TestExtractTags_hierarchical(t *testing.T) {
	memoJSON := map[string]any{
		"name":        "memos/1",
		"content":     "Deploy #tracing/jaeger in prod alongside #tracing/prometheus",
		"visibility":  "PRIVATE",
		"pinned":      false,
		"createTime":  "2026-04-25T10:00:00Z",
		"updateTime":  "2026-04-25T10:00:00Z",
		"displayTime": "2026-04-25T10:00:00Z",
		"creator":     "users/1",
	}
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"memos": []any{memoJSON}, "nextPageToken": ""})
	}))

	memos, _, err := c.ListMemos("", "", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	tags := memos[0].Tags
	if len(tags) != 2 {
		t.Fatalf("want 2 tags, got %v", tags)
	}
	if tags[0] != "tracing/jaeger" {
		t.Errorf("tags[0] = %q, want tracing/jaeger", tags[0])
	}
	if tags[1] != "tracing/prometheus" {
		t.Errorf("tags[1] = %q, want tracing/prometheus", tags[1])
	}
}

func TestExtractTags_deepHierarchy(t *testing.T) {
	memoJSON := map[string]any{
		"name":        "memos/1",
		"content":     "Notes on #infra/k8s/prod setup",
		"visibility":  "PRIVATE",
		"pinned":      false,
		"createTime":  "2026-04-25T10:00:00Z",
		"updateTime":  "2026-04-25T10:00:00Z",
		"displayTime": "2026-04-25T10:00:00Z",
		"creator":     "users/1",
	}
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"memos": []any{memoJSON}, "nextPageToken": ""})
	}))

	memos, _, err := c.ListMemos("", "", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	tags := memos[0].Tags
	if len(tags) != 1 || tags[0] != "infra/k8s/prod" {
		t.Errorf("tags = %v, want [infra/k8s/prod]", tags)
	}
}

func TestExtractTags_duplicates(t *testing.T) {
	memoJSON := map[string]any{
		"name":        "memos/1",
		"content":     "#foo bar #foo baz #foo",
		"visibility":  "PRIVATE",
		"pinned":      false,
		"createTime":  "2026-04-25T10:00:00Z",
		"updateTime":  "2026-04-25T10:00:00Z",
		"displayTime": "2026-04-25T10:00:00Z",
		"creator":     "users/1",
	}
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"memos": []any{memoJSON}, "nextPageToken": ""})
	}))

	memos, _, err := c.ListMemos("", "", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	if len(memos[0].Tags) != 1 || memos[0].Tags[0] != "foo" {
		t.Errorf("expected deduplication, got tags %v", memos[0].Tags)
	}
}

func TestExtractTags_noTags(t *testing.T) {
	memoJSON := map[string]any{
		"name":        "memos/1",
		"content":     "No hashtags here",
		"visibility":  "PRIVATE",
		"pinned":      false,
		"createTime":  "2026-04-25T10:00:00Z",
		"updateTime":  "2026-04-25T10:00:00Z",
		"displayTime": "2026-04-25T10:00:00Z",
		"creator":     "users/1",
	}
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"memos": []any{memoJSON}, "nextPageToken": ""})
	}))

	memos, _, err := c.ListMemos("", "", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	if len(memos[0].Tags) != 0 {
		t.Errorf("expected no tags, got %v", memos[0].Tags)
	}
}

// --- New ---

func TestNew(t *testing.T) {
	c := New("https://example.com/", "my-token")
	if c.BaseURL != "https://example.com" { // trailing slash stripped
		t.Errorf("BaseURL = %q, want trailing slash stripped", c.BaseURL)
	}
	if c.Token != "my-token" {
		t.Errorf("Token = %q, want %q", c.Token, "my-token")
	}
	if c.HTTP == nil {
		t.Error("HTTP client is nil")
	}
}

// --- ListMemos ---

func TestListMemos_requestShape(t *testing.T) {
	var gotPath, gotFilter, gotToken, gotAuth string
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotFilter = r.URL.Query().Get("filter")
		gotToken = r.URL.Query().Get("pageToken")
		gotAuth = r.Header.Get("Authorization")
		writeJSON(w, map[string]any{"memos": []any{}, "nextPageToken": ""})
	}))

	_, _, err := c.ListMemos(`content.contains("hello")`, "tok123", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	if gotPath != "/api/v1/memos" {
		t.Errorf("path = %q, want /api/v1/memos", gotPath)
	}
	if gotFilter != `content.contains("hello")` {
		t.Errorf("filter = %q", gotFilter)
	}
	if gotToken != "tok123" {
		t.Errorf("pageToken = %q", gotToken)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
}

func TestListMemos_pagination(t *testing.T) {
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"memos":         []any{},
			"nextPageToken": "page2",
		})
	}))

	_, nextToken, err := c.ListMemos("", "", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	if nextToken != "page2" {
		t.Errorf("nextPageToken = %q, want page2", nextToken)
	}
}

func TestListMemos_noFilter(t *testing.T) {
	var rawQuery string
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		writeJSON(w, map[string]any{"memos": []any{}, "nextPageToken": ""})
	}))

	_, _, err := c.ListMemos("", "", false)
	if err != nil {
		t.Fatalf("ListMemos: %v", err)
	}
	params, _ := url.ParseQuery(rawQuery)
	if params.Get("filter") != "" {
		t.Errorf("expected no filter param, got %q", params.Get("filter"))
	}
}

// --- GetMemo ---

func TestGetMemo(t *testing.T) {
	ts := mustParseTime("2026-04-25T10:00:00Z")
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/memos/42" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{
			"name":        "memos/42",
			"content":     "Content #tag",
			"visibility":  "PUBLIC",
			"pinned":      true,
			"createTime":  ts.Format(time.RFC3339),
			"updateTime":  ts.Format(time.RFC3339),
			"displayTime": ts.Format(time.RFC3339),
			"creator":     "users/1",
		})
	}))

	m, err := c.GetMemo("memos/42")
	if err != nil {
		t.Fatalf("GetMemo: %v", err)
	}
	if m.Name != "memos/42" {
		t.Errorf("Name = %q", m.Name)
	}
	if m.Visibility != model.VisibilityPublic {
		t.Errorf("Visibility = %q", m.Visibility)
	}
	if !m.Pinned {
		t.Error("Pinned should be true")
	}
	if len(m.Tags) != 1 || m.Tags[0] != "tag" {
		t.Errorf("Tags = %v", m.Tags)
	}
}

// --- CreateMemo ---

func TestCreateMemo(t *testing.T) {
	var gotBody map[string]any
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		writeJSON(w, map[string]any{
			"name":        "memos/99",
			"content":     gotBody["content"],
			"visibility":  gotBody["visibility"],
			"pinned":      false,
			"createTime":  "2026-04-25T10:00:00Z",
			"updateTime":  "2026-04-25T10:00:00Z",
			"displayTime": "2026-04-25T10:00:00Z",
			"creator":     "users/1",
		})
	}))

	m, err := c.CreateMemo("New memo #test", model.VisibilityPrivate)
	if err != nil {
		t.Fatalf("CreateMemo: %v", err)
	}
	if m.Name != "memos/99" {
		t.Errorf("Name = %q", m.Name)
	}
	if gotBody["content"] != "New memo #test" {
		t.Errorf("body content = %v", gotBody["content"])
	}
	if gotBody["visibility"] != string(model.VisibilityPrivate) {
		t.Errorf("body visibility = %v", gotBody["visibility"])
	}
}

// --- UpdateMemo ---

func TestUpdateMemo(t *testing.T) {
	var gotPath, gotUpdateMask string
	var gotBody map[string]any
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUpdateMask = r.URL.Query().Get("updateMask")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		writeJSON(w, map[string]any{
			"name":        "memos/1",
			"content":     gotBody["content"],
			"visibility":  "PRIVATE",
			"pinned":      false,
			"createTime":  "2026-04-25T10:00:00Z",
			"updateTime":  "2026-04-25T10:00:00Z",
			"displayTime": "2026-04-25T10:00:00Z",
			"creator":     "users/1",
		})
	}))

	m, err := c.UpdateMemo("memos/1", "Updated content")
	if err != nil {
		t.Fatalf("UpdateMemo: %v", err)
	}
	if gotPath != "/api/v1/memos/1" {
		t.Errorf("path = %q", gotPath)
	}
	if gotUpdateMask != "content" {
		t.Errorf("updateMask = %q, want content", gotUpdateMask)
	}
	if m.Content != "Updated content" {
		t.Errorf("Content = %q", m.Content)
	}
}

// --- UpdateVisibility ---

func TestUpdateVisibility(t *testing.T) {
	var gotBody map[string]any
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		writeJSON(w, map[string]any{
			"name":        "memos/1",
			"content":     "",
			"visibility":  gotBody["visibility"],
			"pinned":      false,
			"createTime":  "2026-04-25T10:00:00Z",
			"updateTime":  "2026-04-25T10:00:00Z",
			"displayTime": "2026-04-25T10:00:00Z",
			"creator":     "users/1",
		})
	}))

	m, err := c.UpdateVisibility("memos/1", model.VisibilityPublic)
	if err != nil {
		t.Fatalf("UpdateVisibility: %v", err)
	}
	if m.Visibility != model.VisibilityPublic {
		t.Errorf("Visibility = %q", m.Visibility)
	}
	if gotBody["updateMask"] != nil {
		// updateMask is in query, not body
	}
}

// --- UpdatePinned ---

func TestUpdatePinned(t *testing.T) {
	var gotBody map[string]any
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		pinned, _ := gotBody["pinned"].(bool)
		writeJSON(w, map[string]any{
			"name":        "memos/1",
			"content":     "",
			"visibility":  "PRIVATE",
			"pinned":      pinned,
			"createTime":  "2026-04-25T10:00:00Z",
			"updateTime":  "2026-04-25T10:00:00Z",
			"displayTime": "2026-04-25T10:00:00Z",
			"creator":     "users/1",
		})
	}))

	m, err := c.UpdatePinned("memos/1", true)
	if err != nil {
		t.Fatalf("UpdatePinned: %v", err)
	}
	if !m.Pinned {
		t.Error("Pinned should be true")
	}
	if gotBody["pinned"] != true {
		t.Errorf("body pinned = %v", gotBody["pinned"])
	}
}

// --- DeleteMemo ---

func TestDeleteMemo(t *testing.T) {
	var gotMethod, gotPath string
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))

	err := c.DeleteMemo("memos/5")
	if err != nil {
		t.Fatalf("DeleteMemo: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/api/v1/memos/5" {
		t.Errorf("path = %q", gotPath)
	}
}

// --- Error handling ---

func TestAPIError_400(t *testing.T) {
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"message": "invalid filter"})
	}))

	_, _, err := c.ListMemos("bad filter", "", false)
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
	if !strings.Contains(err.Error(), "invalid filter") {
		t.Errorf("error %q does not contain API message", err.Error())
	}
}

func TestAPIError_500_noBody(t *testing.T) {
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, _, err := c.ListMemos("", "", false)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestAuthHeader(t *testing.T) {
	var gotAuth string
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		writeJSON(w, map[string]any{})
	}))

	_ = c.DeleteMemo("memos/1")
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", gotAuth)
	}
}

func TestListShortcuts(t *testing.T) {
	c := serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/1/shortcuts" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		writeJSON(w, model.ListShortcutsResponse{
			Shortcuts: []model.Shortcut{
				{
					Name:   "users/1/shortcuts/abc",
					Title:  "Work memos",
					Filter: `content.contains("#work")`,
				},
			},
		})
	}))

	shortcuts, err := c.ListShortcuts("users/1")
	if err != nil {
		t.Fatalf("ListShortcuts() error: %v", err)
	}
	if len(shortcuts) != 1 {
		t.Fatalf("expected 1 shortcut, got %d", len(shortcuts))
	}
	if shortcuts[0].Title != "Work memos" {
		t.Errorf("Title = %q, want Work memos", shortcuts[0].Title)
	}
	if shortcuts[0].Filter != `content.contains("#work")` {
		t.Errorf("Filter = %q, want content.contains(\"#work\")", shortcuts[0].Filter)
	}
}
