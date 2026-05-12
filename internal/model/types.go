package model

import "time"

type MemoState string

const (
	MemoStateNormal   MemoState = "NORMAL"
	MemoStateArchived MemoState = "ARCHIVED"
)

type Visibility string

const (
	VisibilityPrivate   Visibility = "PRIVATE"
	VisibilityProtected Visibility = "PROTECTED"
	VisibilityPublic    Visibility = "PUBLIC"
)

func (v Visibility) Short() string {
	switch v {
	case VisibilityPublic, VisibilityProtected, VisibilityPrivate:
		return string(v)
	default:
		return "PRIVATE"
	}
}

func (v Visibility) Next() Visibility {
	switch v {
	case VisibilityPrivate:
		return VisibilityProtected
	case VisibilityProtected:
		return VisibilityPublic
	default:
		return VisibilityPrivate
	}
}

type Memo struct {
	Name        string     `json:"name"`        // "memos/{id}"
	Content     string     `json:"content"`
	Visibility  Visibility `json:"visibility"`
	Pinned      bool       `json:"pinned"`
	State       MemoState  `json:"state"`
	CreateTime  time.Time  `json:"createTime"`
	UpdateTime  time.Time  `json:"updateTime"`
	DisplayTime time.Time  `json:"displayTime"`
	Creator     string     `json:"creator"` // "users/{uid}"
	Tags        []string   `json:"tags"`    // returned by API; may be supplemented by caller
}

type User struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type ListMemosResponse struct {
	Memos         []Memo `json:"memos"`
	NextPageToken string `json:"nextPageToken"`
}

type ListTagsResponse struct {
	TagAmounts map[string]int32 `json:"tagAmounts"`
}

type Shortcut struct {
	Name   string `json:"name"`   // "users/{uid}/shortcuts/{id}"
	Title  string `json:"title"`
	Filter string `json:"filter"`
}

type ListShortcutsResponse struct {
	Shortcuts []Shortcut `json:"shortcuts"`
}
