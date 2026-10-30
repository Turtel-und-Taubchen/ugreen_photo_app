package geo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CacheTTL is how long a stored Nominatim answer is served without asking again.
const CacheTTL = 90 * 24 * time.Hour

// cache keeps raw Nominatim answers in app.db's geocode_cache; parsing happens on every read
// so a mapping change applies to cached answers too.
type cache struct {
	db  *sql.DB
	now func() time.Time
}

func searchKey(q string) string {
	return "search:de:" + strings.ToLower(strings.Join(strings.Fields(q), " "))
}

func reverseKey(lat, lon float64) string {
	return fmt.Sprintf("reverse:de:%.5f,%.5f", lat, lon)
}

// get returns a fresh answer for key; an expired or missing entry is a miss.
func (c *cache) get(ctx context.Context, key string) ([]byte, bool, error) {
	var response string
	var fetchedAt int64
	err := c.db.QueryRowContext(ctx, `SELECT response, fetched_at FROM geocode_cache WHERE key = ?`, key).
		Scan(&response, &fetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("geo: read geocode cache: %w", err)
	}
	if c.now().Sub(time.Unix(fetchedAt, 0)) >= CacheTTL {
		return nil, false, nil
	}
	return []byte(response), true, nil
}

func (c *cache) put(ctx context.Context, key string, response []byte) error {
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO geocode_cache (key, response, fetched_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET response = excluded.response, fetched_at = excluded.fetched_at`,
		key, string(response), c.now().Unix())
	if err != nil {
		return fmt.Errorf("geo: write geocode cache: %w", err)
	}
	return nil
}
