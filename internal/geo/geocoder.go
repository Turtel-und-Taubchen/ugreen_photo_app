package geo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrDisabled means no contact address is configured; Nominatim's policy requires one.
	ErrDisabled = errors.New("geo: geocoding disabled, no contact configured")
	// ErrUpstream wraps every failure to get a usable answer from Nominatim.
	ErrUpstream = errors.New("geo: nominatim unavailable")
)

const (
	requestInterval = time.Second
	clientTimeout   = 10 * time.Second
	maxAnswerBytes  = 4 << 20
)

// Options configure a Geocoder. HTTP, Now and Sleep are replaceable for tests.
type Options struct {
	BaseURL string
	Contact string
	Version string
	DB      *sql.DB
	HTTP    *http.Client
	Now     func() time.Time
	Sleep   func(context.Context, time.Duration) error
}

// Geocoder is the app's only way to Nominatim: one request per second across all users,
// answers cached in app.db.
type Geocoder struct {
	baseURL   string
	contact   string
	userAgent string
	http      *http.Client
	cache     *cache
	limiter   *limiter
}

// New builds a Geocoder; without HTTP, Now and Sleep it uses a 10 s client and the real clock.
func New(o Options) *Geocoder {
	if o.HTTP == nil {
		o.HTTP = &http.Client{Timeout: clientTimeout}
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Sleep == nil {
		o.Sleep = sleepCtx
	}
	return &Geocoder{
		baseURL:   strings.TrimRight(o.BaseURL, "/"),
		contact:   o.Contact,
		userAgent: fmt.Sprintf("PhotoCurator/%s (%s)", o.Version, o.Contact),
		http:      o.HTTP,
		cache:     &cache{db: o.DB, now: o.Now},
		limiter:   newLimiter(requestInterval, o.Now, o.Sleep),
	}
}

// Enabled reports whether a contact is configured; without one nothing is sent.
func (g *Geocoder) Enabled() bool {
	return g.contact != ""
}

// Search looks up free text. An empty result is a valid, cached answer.
func (g *Geocoder) Search(ctx context.Context, q string) ([]Place, error) {
	params := url.Values{
		"q":               {strings.TrimSpace(q)},
		"format":          {"jsonv2"},
		"addressdetails":  {"1"},
		"namedetails":     {"1"},
		"accept-language": {"de"},
		"limit":           {strconv.Itoa(MaxResults)},
	}
	return lookup(ctx, g, searchKey(q), "/search", params, ParseSearch)
}

// Reverse finds the place at a coordinate; nil without error means Nominatim knows none there.
func (g *Geocoder) Reverse(ctx context.Context, lat, lon float64) (*Place, error) {
	// The request uses the cache key's precision so a stored answer is exactly the one for its key.
	params := url.Values{
		"lat":             {strconv.FormatFloat(lat, 'f', 5, 64)},
		"lon":             {strconv.FormatFloat(lon, 'f', 5, 64)},
		"format":          {"jsonv2"},
		"addressdetails":  {"1"},
		"namedetails":     {"1"},
		"accept-language": {"de"},
	}
	return lookup(ctx, g, reverseKey(lat, lon), "/reverse", params, ParseReverse)
}

// lookup serves key from the cache or, within the limiter, from Nominatim. The cache is read
// again after the wait: a caller queued behind an identical query gets that answer without
// using a slot.
func lookup[T any](ctx context.Context, g *Geocoder, key, path string, params url.Values, parse func([]byte) (T, error)) (T, error) {
	var zero T
	if !g.Enabled() {
		return zero, ErrDisabled
	}
	if body, ok, err := g.cache.get(ctx, key); err != nil || ok {
		return parseCached(body, err, parse)
	}
	t, err := g.limiter.Wait(ctx)
	if err != nil {
		return zero, err
	}
	defer t.Done()
	if body, ok, err := g.cache.get(ctx, key); err != nil || ok {
		return parseCached(body, err, parse)
	}
	t.Start()
	body, err := g.get(ctx, path, params)
	if err != nil {
		return zero, err
	}
	v, err := parse(body)
	if err != nil {
		return zero, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	if err := g.cache.put(ctx, key, body); err != nil {
		return zero, err
	}
	return v, nil
}

func parseCached[T any](body []byte, err error, parse func([]byte) (T, error)) (T, error) {
	if err != nil {
		var zero T
		return zero, err
	}
	return parse(body)
}

func (g *Geocoder) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	req.Header.Set("User-Agent", g.userAgent)
	resp, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswerBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	return body, nil
}
