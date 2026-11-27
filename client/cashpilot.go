package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	// MaxResponseBytes caps one CashPilot response. The largest real payload,
	// the full service catalog, is a few hundred kilobytes.
	MaxResponseBytes = 4 << 20
	// maxErrorBytes caps how much of an error body is read to find its detail.
	maxErrorBytes = 8 << 10
	// maxDetailRunes caps the upstream detail quoted back to the model.
	maxDetailRunes = 300
	// maxConfirmRunes caps a confirm_deploy_targets message, which lists every
	// copy and may end with a warning about an old worker: kept whole.
	maxConfirmRunes = maxDetailRunes + maxReportedCopies*40
	// maxReportedCopies caps the containers kept from one refusal.
	maxReportedCopies = 32
	// invalidRequestText stands in for a detail that cannot be shown.
	invalidRequestText = "the request was rejected as invalid"
)

// slugPattern matches every CashPilot catalog slug (honeygain, sentinel-dvpn,
// proxybase-xyz, ...). It refuses "." and "..", slashes and anything else that
// could change the meaning of the URL path it is placed in.
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// ErrInvalidSlug is returned before any request is sent for a slug that is
// not a catalog slug.
var ErrInvalidSlug = errors.New("invalid service slug: use the lowercase slug from list_available_services or list_deployed_services, e.g. honeygain")

// ValidateSlug reports whether slug can be sent to CashPilot.
func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return ErrInvalidSlug
	}
	return nil
}

// Target picks which copy of a service a call acts on. Zero values are left
// out of the request, so CashPilot applies its own defaults: the only online
// worker, and the first instance.
type Target struct {
	WorkerID int
	Instance int
}

func (t Target) apply(q url.Values) {
	if t.WorkerID > 0 {
		q.Set("worker_id", strconv.Itoa(t.WorkerID))
	}
	if t.Instance > 0 {
		q.Set("instance", strconv.Itoa(t.Instance))
	}
}

// APIError is a non-2xx answer from CashPilot, reduced to what the model can
// act on. The raw body is never kept: it can be a proxy's HTML page.
type APIError struct {
	Status int
	Detail string
	// Code is the "error" field of a structured detail, e.g.
	// confirm_deploy_targets; empty for a plain-string detail.
	Code string
	// Containers is the "expected_containers" map of a confirm_deploy_targets
	// answer: copy key ("1", "2", "p3" for a proxied copy) to container id.
	Containers map[string]string
	// Current is the "current_container" of a deploy_target_changed answer:
	// the id of the container the worker found where the deploy expected none.
	Current string
}

// The deploy refusals CashPilot 3.0 answers when a deploy would replace a
// container that already exists on the worker.
const (
	// CodeConfirmDeployTargets: the dashboard asks the owner to confirm the
	// replacement (web app/main.py _ask_deploy_confirmation).
	CodeConfirmDeployTargets = "confirm_deploy_targets"
	// CodeDeployTargetChanged: the worker found a container where the deploy
	// expected none (web app/main.py _safe_worker_detail).
	CodeDeployTargetChanged = "deploy_target_changed"
)

func (e *APIError) Error() string {
	msg := fmt.Sprintf("CashPilot answered %d", e.Status)
	if e.Detail != "" {
		msg += ": " + e.Detail
		if e.Code != "" && e.Code != e.Detail {
			msg += " (" + e.Code + ")"
		}
	} else if t := http.StatusText(e.Status); t != "" {
		msg += " " + t
	}
	switch {
	case e.Status == http.StatusUnauthorized:
		msg += ". CASHPILOT_API_KEY is missing or wrong."
	case e.Status == http.StatusForbidden && e.Detail == "Read-only token":
		// app/deps.py _require_auth_api; the allowlist is _require_reader's callers.
		msg += ". The read-only key allows only get_earnings_summary, get_earnings_breakdown, get_fleet_summary, get_health_scores and list_deployed_services. " +
			"Use the admin key for this call, or set CASHPILOT_READ_ONLY=true to list only the tools this key allows."
	case e.Status == http.StatusForbidden && strings.Contains(e.Detail, "fleet key"):
		msg += ". Use the admin key or the read-only key, never the fleet key."
	case e.Status == http.StatusForbidden:
		msg += ". This API key's role does not allow this call."
	case e.Status == http.StatusBadRequest && strings.Contains(e.Detail, "worker_id"):
		msg += ". Pass worker_id; list_workers shows the ids."
	case e.Status == http.StatusNotFound && isWorkerNotFound(e.Detail):
		msg += ". Check worker_id with list_workers."
	case e.Status == http.StatusNotFound:
		msg += ". Check the slug with list_deployed_services or list_available_services."
	}
	return msg
}

// isWorkerNotFound matches CashPilot's worker 404s by their start: "Worker not
// found", "Unknown worker 7" and "No such worker, ..." (app/main.py). A service
// 404 such as "Service 'worker' not found" must fall through to the slug hint.
func isWorkerNotFound(detail string) bool {
	d := strings.ToLower(detail)
	return strings.HasPrefix(d, "worker not found") ||
		strings.HasPrefix(d, "unknown worker") ||
		strings.HasPrefix(d, "no such worker")
}

type Client struct {
	base   string
	hc     *http.Client
	slow   *http.Client
	apiKey string
}

func NewClient(base, apiKey string) *Client {
	return &Client{
		base: strings.TrimRight(base, "/"),
		hc:   &http.Client{Timeout: 30 * time.Second},
		// A deploy pulls the image on the worker before it answers.
		slow:   &http.Client{Timeout: 5 * time.Minute},
		apiKey: apiKey,
	}
}

func (c *Client) buildURL(path string, q url.Values) string {
	u := c.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

func (c *Client) do(hc *http.Client, req *http.Request) ([]byte, error) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, transportError(req.Context(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
		apiErr := &APIError{Status: resp.StatusCode}
		detailOf(b, apiErr)
		return nil, apiErr
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, transportError(req.Context(), err)
	}
	if len(b) > MaxResponseBytes {
		return nil, fmt.Errorf("CashPilot response is larger than %d MiB; refusing it", MaxResponseBytes>>20)
	}
	return b, nil
}

// transportError keeps the CashPilot URL out of what the model sees. The full
// error, which names the host, goes to the operator's log (stderr).
func transportError(ctx context.Context, err error) error {
	log.Printf("CashPilot request failed: %v", err)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var ue *url.Error
	if errors.As(err, &ue) && ue.Timeout() {
		return errors.New("CashPilot did not answer in time")
	}
	return errors.New("could not reach CashPilot: check CASHPILOT_URL and that the instance is running")
}

// codePattern and the two below accept only the shapes CashPilot sends, so
// nothing else from an error body reaches the model.
var (
	codePattern        = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	copyKeyPattern     = regexp.MustCompile(`^p?[0-9]{1,4}$`)
	containerIDPattern = regexp.MustCompile(`^(none|[0-9a-f]{12,64})$`)
)

// detailOf pulls FastAPI's detail out of an error body. A string detail is
// quoted. A dict detail, as CashPilot 3.0 sends for its structured refusals
// ({"error": "...", "message": "...", ...}), gives its message, its error code
// and the containers it is about: expected_containers for
// confirm_deploy_targets, current_container for deploy_target_changed. Anything
// else (HTML, validation arrays) is dropped rather than echoed.
func detailOf(body []byte, e *APIError) {
	var v struct {
		Detail json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(body, &v) != nil || len(v.Detail) == 0 {
		return
	}
	var s string
	if json.Unmarshal(v.Detail, &s) == nil {
		e.Detail = cleanDetail(s)
		return
	}
	var d struct {
		Error      json.RawMessage `json:"error"`
		Message    json.RawMessage `json:"message"`
		Containers json.RawMessage `json:"expected_containers"`
		Current    json.RawMessage `json:"current_container"`
	}
	if json.Unmarshal(v.Detail, &d) != nil {
		e.Detail = invalidRequestText
		return
	}
	var msg, code string
	_ = json.Unmarshal(d.Message, &msg) // a non-string message is dropped
	_ = json.Unmarshal(d.Error, &code)
	if !codePattern.MatchString(code) {
		code = ""
	}
	switch code {
	case CodeConfirmDeployTargets:
		e.Containers = containersOf(d.Containers)
	case CodeDeployTargetChanged:
		var id string
		if json.Unmarshal(d.Current, &id) == nil && containerIDPattern.MatchString(id) {
			e.Current = id
		}
	}
	e.Code = code
	limit := maxDetailRunes
	if code == CodeConfirmDeployTargets {
		limit = maxConfirmRunes
	}
	e.Detail = capDetail(msg, limit)
	if e.Detail == "" {
		e.Detail = code
	}
	if e.Detail == "" {
		e.Detail = invalidRequestText
	}
}

// ErrInvalidContainers is returned before any request is sent for a
// replacement map that is not one CashPilot sent.
var ErrInvalidContainers = fmt.Errorf("replace_containers must be the map from CashPilot's refusal, copied unchanged: "+
	"copy number (\"1\", or \"p3\" for a proxied copy) to container id or \"none\", at most %d entries", maxReportedCopies)

// ValidateContainers reports whether m has the shape of the
// expected_containers map CashPilot sends with confirm_deploy_targets, so it
// can be sent back as the owner's confirmation.
func ValidateContainers(m map[string]string) error {
	if len(m) == 0 || len(m) > maxReportedCopies {
		return ErrInvalidContainers
	}
	for k, id := range m {
		if !copyKeyPattern.MatchString(k) || !containerIDPattern.MatchString(id) {
			return ErrInvalidContainers
		}
	}
	return nil
}

// containersOf keeps the copy-to-container pairs of expected_containers that
// look like CashPilot's own, at most maxReportedCopies of them.
func containersOf(raw json.RawMessage) map[string]string {
	var in map[string]json.RawMessage
	if json.Unmarshal(raw, &in) != nil {
		return nil
	}
	out := map[string]string{}
	for k, rv := range in {
		var id string
		if json.Unmarshal(rv, &id) != nil || !copyKeyPattern.MatchString(k) || !containerIDPattern.MatchString(id) {
			continue
		}
		if len(out) == maxReportedCopies {
			break
		}
		out[k] = id
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// cleanDetail puts an upstream text on one line and caps its length.
func cleanDetail(s string) string { return capDetail(s, maxDetailRunes) }

// capDetail puts an upstream text on one line and caps it at limit runes.
func capDetail(s string, limit int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > limit {
		s = string(r[:limit]) + "…"
	}
	return s
}

func (c *Client) send(ctx context.Context, hc *http.Client, method, path string, q url.Values, body []byte) ([]byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, q), r)
	if err != nil {
		return nil, errors.New("CASHPILOT_URL is not a valid URL")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(hc, req)
}

func (c *Client) doGet(ctx context.Context, path string, q url.Values) ([]byte, error) {
	return c.send(ctx, c.hc, http.MethodGet, path, q, nil)
}

func (c *Client) servicePath(slug, format string) (string, error) {
	if err := ValidateSlug(slug); err != nil {
		return "", err
	}
	return fmt.Sprintf(format, url.PathEscape(slug)), nil
}

// ---------------------------------------------------------------------------
// Reads (GET)
// ---------------------------------------------------------------------------

func (c *Client) GetEarningsSummary(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/earnings/summary", nil)
}

func (c *Client) GetEarningsBreakdown(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/earnings/breakdown", nil)
}

func (c *Client) GetServicesDeployed(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/services/deployed", nil)
}

func (c *Client) GetServicesCatalog(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/services", nil)
}

// GetServicesAvailable is the catalog without the services CashPilot marks
// broken or gone, with deployed, has_collector and node_count added.
func (c *Client) GetServicesAvailable(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/services/available", nil)
}

func (c *Client) GetFleetSummary(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/fleet/summary", nil)
}

func (c *Client) GetWorkers(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/workers", nil)
}

// GetHealthScores reads the health scores over the last days (1-90); 0 keeps
// CashPilot's default of 7.
func (c *Client) GetHealthScores(ctx context.Context, days int) ([]byte, error) {
	q := url.Values{}
	if days > 0 {
		q.Set("days", strconv.Itoa(days))
	}
	return c.doGet(ctx, "/api/health/scores", q)
}

func (c *Client) GetCollectorAlerts(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/collector-alerts", nil)
}

func (c *Client) GetEarningsDaily(ctx context.Context, days int) ([]byte, error) {
	q := url.Values{}
	q.Set("days", strconv.Itoa(days))
	return c.doGet(ctx, "/api/earnings/daily", q)
}

func (c *Client) GetEarningsHistory(ctx context.Context, period string) ([]byte, error) {
	q := url.Values{}
	q.Set("period", period)
	return c.doGet(ctx, "/api/earnings/history", q)
}

func (c *Client) GetServiceLogs(ctx context.Context, slug string, lines int, t Target) ([]byte, error) {
	path, err := c.servicePath(slug, "/api/services/%s/logs")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("lines", strconv.Itoa(lines))
	t.apply(q)
	return c.doGet(ctx, path, q)
}

// GetCompose returns the docker-compose YAML for one service. arch is amd64,
// arm64 or arm; "" means the catalog's default image.
func (c *Client) GetCompose(ctx context.Context, slug, arch string) ([]byte, error) {
	path, err := c.servicePath(slug, "/api/compose/%s")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if arch != "" {
		q.Set("arch", arch)
	}
	return c.doGet(ctx, path, q)
}

// ---------------------------------------------------------------------------
// Writes (POST, DELETE)
// ---------------------------------------------------------------------------

func (c *Client) lifecycle(ctx context.Context, slug, action string, t Target) ([]byte, error) {
	path, err := c.servicePath(slug, "/api/services/%s/"+action)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	t.apply(q)
	return c.send(ctx, c.hc, http.MethodPost, path, q, nil)
}

func (c *Client) RestartService(ctx context.Context, slug string, t Target) ([]byte, error) {
	return c.lifecycle(ctx, slug, "restart", t)
}

func (c *Client) StopService(ctx context.Context, slug string, t Target) ([]byte, error) {
	return c.lifecycle(ctx, slug, "stop", t)
}

func (c *Client) StartService(ctx context.Context, slug string, t Target) ([]byte, error) {
	return c.lifecycle(ctx, slug, "start", t)
}

// DeployService posts a DeployRequest body. Only WorkerID of t is used: the
// number of copies is part of the body.
func (c *Client) DeployService(ctx context.Context, slug string, body []byte, t Target) ([]byte, error) {
	path, err := c.servicePath(slug, "/api/deploy/%s")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	Target{WorkerID: t.WorkerID}.apply(q)
	return c.send(ctx, c.slow, http.MethodPost, path, q, body)
}

// RemoveService removes a deployment. Volumes are kept: delete_volumes is
// never sent, so node identities and wallets survive a removal.
func (c *Client) RemoveService(ctx context.Context, slug string, t Target) ([]byte, error) {
	path, err := c.servicePath(slug, "/api/services/%s")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	Target{WorkerID: t.WorkerID}.apply(q)
	return c.send(ctx, c.hc, http.MethodDelete, path, q, nil)
}

func (c *Client) TriggerCollection(ctx context.Context) ([]byte, error) {
	return c.send(ctx, c.hc, http.MethodPost, "/api/collect", nil, nil)
}
