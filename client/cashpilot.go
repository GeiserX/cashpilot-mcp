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
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("CashPilot answered %d", e.Status)
	if e.Detail != "" {
		msg += ": " + e.Detail
	} else if t := http.StatusText(e.Status); t != "" {
		msg += " " + t
	}
	switch {
	case e.Status == http.StatusUnauthorized:
		msg += ". CASHPILOT_API_KEY is missing or wrong."
	case e.Status == http.StatusForbidden:
		msg += ". The API key's role does not allow this call: a read-only key cannot change services or read logs."
	case e.Status == http.StatusBadRequest && strings.Contains(e.Detail, "worker_id"):
		msg += ". Pass worker_id; list_workers shows the ids."
	case e.Status == http.StatusNotFound:
		msg += ". Check the slug with list_deployed_services or list_available_services."
	}
	return msg
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
		return nil, &APIError{Status: resp.StatusCode, Detail: detailOf(b)}
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

// detailOf pulls FastAPI's {"detail": "..."} out of an error body. Anything
// else (HTML, validation arrays) is dropped rather than echoed.
func detailOf(body []byte) string {
	var v struct {
		Detail json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(body, &v) != nil || len(v.Detail) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(v.Detail, &s) != nil {
		return "the request was rejected as invalid"
	}
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	if r := []rune(s); len(r) > maxDetailRunes {
		s = string(r[:maxDetailRunes]) + "…"
	}
	return strings.TrimSpace(s)
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
