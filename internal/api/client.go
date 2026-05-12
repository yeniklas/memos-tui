package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/yeniklas/memos-tui/internal/model"
)

var tagRE = regexp.MustCompile(`#(\w+(?:/\w+)*)`)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{},
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	return c.HTTP.Do(req)
}

func (c *Client) get(path string, params url.Values) (*http.Response, error) {
	u := c.BaseURL + "/api/v1" + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) post(path string, body any) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1"+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) patch(path string, body any) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPatch, c.BaseURL+"/api/v1"+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) delete(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+"/api/v1"+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func decode[T any](resp *http.Response) (T, error) {
	var v T
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var e struct{ Message string `json:"message"` }
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Message != "" {
			return v, fmt.Errorf("API %d: %s", resp.StatusCode, e.Message)
		}
		return v, fmt.Errorf("API error %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}

func extractTags(content string) []string {
	matches := tagRE.FindAllStringSubmatch(content, -1)
	seen := map[string]bool{}
	var tags []string
	for _, m := range matches {
		t := m[1]
		if !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}
	return tags
}

func withTags(m model.Memo) model.Memo {
	if len(m.Tags) == 0 {
		m.Tags = extractTags(m.Content)
	}
	return m
}

func (c *Client) ListMemos(filter, pageToken string, archived bool) ([]model.Memo, string, error) {
	params := url.Values{"pageSize": {"50"}}
	if filter != "" {
		params.Set("filter", filter)
	}
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	if archived {
		params.Set("state", "ARCHIVED")
	}
	resp, err := c.get("/memos", params)
	if err != nil {
		return nil, "", err
	}
	r, err := decode[model.ListMemosResponse](resp)
	if err != nil {
		return nil, "", err
	}
	memos := make([]model.Memo, len(r.Memos))
	for i, m := range r.Memos {
		memos[i] = withTags(m)
	}
	return memos, r.NextPageToken, nil
}

func (c *Client) GetMemo(name string) (model.Memo, error) {
	resp, err := c.get("/"+name, nil)
	if err != nil {
		return model.Memo{}, err
	}
	m, err := decode[model.Memo](resp)
	if err != nil {
		return model.Memo{}, err
	}
	return withTags(m), nil
}

func (c *Client) CreateMemo(content string, visibility model.Visibility) (model.Memo, error) {
	body := map[string]any{
		"content":    content,
		"visibility": string(visibility),
	}
	resp, err := c.post("/memos", body)
	if err != nil {
		return model.Memo{}, err
	}
	m, err := decode[model.Memo](resp)
	if err != nil {
		return model.Memo{}, err
	}
	return withTags(m), nil
}

func (c *Client) UpdateMemo(name, content string) (model.Memo, error) {
	body := map[string]any{
		"name":    name,
		"content": content,
	}
	resp, err := c.patch("/"+name+"?updateMask=content", body)
	if err != nil {
		return model.Memo{}, err
	}
	m, err := decode[model.Memo](resp)
	if err != nil {
		return model.Memo{}, err
	}
	return withTags(m), nil
}

func (c *Client) UpdateVisibility(name string, vis model.Visibility) (model.Memo, error) {
	body := map[string]any{
		"name":       name,
		"visibility": string(vis),
	}
	resp, err := c.patch("/"+name+"?updateMask=visibility", body)
	if err != nil {
		return model.Memo{}, err
	}
	m, err := decode[model.Memo](resp)
	if err != nil {
		return model.Memo{}, err
	}
	return withTags(m), nil
}

func (c *Client) UpdatePinned(name string, pinned bool) (model.Memo, error) {
	body := map[string]any{
		"name":   name,
		"pinned": pinned,
	}
	resp, err := c.patch("/"+name+"?updateMask=pinned", body)
	if err != nil {
		return model.Memo{}, err
	}
	m, err := decode[model.Memo](resp)
	if err != nil {
		return model.Memo{}, err
	}
	return withTags(m), nil
}

func (c *Client) UpdateState(name string, state model.MemoState) (model.Memo, error) {
	body := map[string]any{
		"name":  name,
		"state": string(state),
	}
	resp, err := c.patch("/"+name+"?updateMask=state", body)
	if err != nil {
		return model.Memo{}, err
	}
	m, err := decode[model.Memo](resp)
	if err != nil {
		return model.Memo{}, err
	}
	return withTags(m), nil
}

func (c *Client) DeleteMemo(name string) error {
	resp, err := c.delete("/" + name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ListShortcuts(userName string) ([]model.Shortcut, error) {
	resp, err := c.get("/"+userName+"/shortcuts", nil)
	if err != nil {
		return nil, err
	}
	r, err := decode[model.ListShortcutsResponse](resp)
	if err != nil {
		return nil, err
	}
	return r.Shortcuts, nil
}

func (c *Client) ListTags() ([]string, error) {
	seen := map[string]bool{}
	pageToken := ""
	for {
		memos, next, err := c.ListMemos("", pageToken, false)
		if err != nil {
			return nil, err
		}
		for _, m := range memos {
			for _, t := range m.Tags {
				seen[t] = true
			}
		}
		if next == "" {
			break
		}
		pageToken = next
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	return tags, nil
}
