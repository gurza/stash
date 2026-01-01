package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/umputun/stash/app/enum"
	"github.com/umputun/stash/app/server/internal/cookie"
	"github.com/umputun/stash/app/store"
)

//go:generate moq -out mocks/auditstore.go -pkg mocks -skip-ensure -fmt goimports . AuditStore

// AuditStore defines the interface for audit log queries.
type AuditStore interface {
	QueryAudit(ctx context.Context, q store.AuditQuery) ([]store.AuditEntry, int, error)
}

// AuditHandler handles audit log web UI requests.
type AuditHandler struct {
	store    AuditStore
	auth     AuthProvider
	tmpl     *templateManager
	baseURL  string
	pageSize int
}

// templateManager wraps template execution for audit handler.
type templateManager struct {
	handler *Handler
}

func (tm *templateManager) execute(w http.ResponseWriter, name string, data any) error {
	if err := tm.handler.tmpl.ExecuteTemplate(w, name, data); err != nil {
		return fmt.Errorf("execute template %s: %w", name, err)
	}
	return nil
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(auditStore AuditStore, auth AuthProvider, h *Handler) *AuditHandler {
	return &AuditHandler{
		store:    auditStore,
		auth:     auth,
		tmpl:     &templateManager{handler: h},
		baseURL:  h.baseURL,
		pageSize: 100,
	}
}

// auditTemplateData holds data passed to audit templates.
type auditTemplateData struct {
	Entries []store.AuditEntry
	Total   int
	Page    int
	Limit   int

	// filter values for form state
	Key       string
	Actor     string
	Action    string
	Result    string
	ActorType string
	From      string
	To        string

	// pagination
	TotalPages int
	HasPrev    bool
	HasNext    bool

	// UI state
	Theme       enum.Theme
	AuthEnabled bool
	BaseURL     string
	Error       string
}

// HandleAuditPage handles GET /audit - renders full audit page.
func (h *AuditHandler) HandleAuditPage(w http.ResponseWriter, r *http.Request) {
	username := h.getCurrentUser(r)
	if username == "" {
		http.Redirect(w, r, h.baseURL+"/login?return="+h.baseURL+"/audit", http.StatusFound)
		return
	}

	if !h.auth.IsAdmin(username) {
		w.WriteHeader(http.StatusForbidden)
		_ = h.tmpl.execute(w, "error", map[string]any{
			"Error":   "Admin access required",
			"BaseURL": h.baseURL,
		})
		return
	}

	data := h.buildAuditData(r)
	if err := h.tmpl.execute(w, "audit.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// HandleAuditTable handles GET /web/audit - returns audit table partial for HTMX.
func (h *AuditHandler) HandleAuditTable(w http.ResponseWriter, r *http.Request) {
	username := h.getCurrentUser(r)
	if username == "" || !h.auth.IsAdmin(username) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	data := h.buildAuditData(r)
	if err := h.tmpl.execute(w, "audit-table", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// buildAuditData builds template data from request parameters.
func (h *AuditHandler) buildAuditData(r *http.Request) auditTemplateData {
	q := r.URL.Query()

	// parse filter params
	key := q.Get("key")
	actor := q.Get("actor")
	action := q.Get("action")
	result := q.Get("result")
	actorType := q.Get("actor_type")
	from := q.Get("from")
	to := q.Get("to")
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	// build query
	query := store.AuditQuery{
		Key:    key,
		Actor:  actor,
		Limit:  h.pageSize,
		Offset: (page - 1) * h.pageSize,
	}

	// parse enums
	if action != "" && action != "all" {
		if a, err := enum.ParseAuditAction(action); err == nil {
			query.Action = a
		}
	}
	if result != "" && result != "all" {
		if r, err := enum.ParseAuditResult(result); err == nil {
			query.Result = r
		}
	}
	if actorType != "" && actorType != "all" {
		if at, err := enum.ParseActorType(actorType); err == nil {
			query.ActorType = at
		}
	}

	// parse timestamps
	if from != "" {
		if t, err := time.Parse("2006-01-02T15:04", from); err == nil {
			query.From = t
		}
	}
	if to != "" {
		if t, err := time.Parse("2006-01-02T15:04", to); err == nil {
			query.To = t
		}
	}

	// execute query
	entries, total, err := h.store.QueryAudit(r.Context(), query)
	if err != nil {
		return auditTemplateData{
			Error:       "Failed to query audit log",
			Theme:       h.getTheme(r),
			AuthEnabled: h.auth.Enabled(),
			BaseURL:     h.baseURL,
		}
	}

	if entries == nil {
		entries = []store.AuditEntry{}
	}

	// pagination
	totalPages := (total + h.pageSize - 1) / h.pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	return auditTemplateData{
		Entries:     entries,
		Total:       total,
		Page:        page,
		Limit:       h.pageSize,
		Key:         key,
		Actor:       actor,
		Action:      action,
		Result:      result,
		ActorType:   actorType,
		From:        from,
		To:          to,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		Theme:       h.getTheme(r),
		AuthEnabled: h.auth.Enabled(),
		BaseURL:     h.baseURL,
	}
}

// getCurrentUser returns the username from the session cookie.
func (h *AuditHandler) getCurrentUser(r *http.Request) string {
	for _, cookieName := range cookie.SessionCookieNames {
		if c, err := r.Cookie(cookieName); err == nil {
			if username, ok := h.auth.GetSessionUser(r.Context(), c.Value); ok {
				return username
			}
		}
	}
	return ""
}

// getTheme returns the current theme from cookie.
func (h *AuditHandler) getTheme(r *http.Request) enum.Theme {
	if c, err := r.Cookie("theme"); err == nil {
		if theme, err := enum.ParseTheme(c.Value); err == nil {
			return theme
		}
	}
	return enum.ThemeSystem
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return strconv.Itoa(mins) + "m ago"
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return strconv.Itoa(hours) + "h ago"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return strconv.Itoa(days) + "d ago"
	default:
		return t.Format("Jan 2")
	}
}

// actionClass returns CSS class for action badge.
func actionClass(action enum.AuditAction) string {
	switch action {
	case enum.AuditActionCreate:
		return "action-create"
	case enum.AuditActionRead:
		return "action-read"
	case enum.AuditActionUpdate:
		return "action-update"
	case enum.AuditActionDelete:
		return "action-delete"
	default:
		return ""
	}
}

// resultClass returns CSS class for result badge.
func resultClass(result enum.AuditResult) string {
	switch result {
	case enum.AuditResultSuccess:
		return "result-success"
	case enum.AuditResultDenied:
		return "result-denied"
	case enum.AuditResultNotFound:
		return "result-not_found"
	default:
		return ""
	}
}

// AuditTemplateFuncs returns template functions for audit templates.
func AuditTemplateFuncs() map[string]any {
	return map[string]any{
		"relativeTime": relativeTime,
		"actionClass":  actionClass,
		"resultClass":  resultClass,
		"upper":        strings.ToUpper,
	}
}
