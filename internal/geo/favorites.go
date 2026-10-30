package geo

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Favorite is a named place the user saved for reuse.
type Favorite struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Parts Parts   `json:"parts"`
}

var (
	ErrFavoriteExists   = errors.New("favorite place name already exists")
	ErrFavoriteNotFound = errors.New("favorite place not found")
)

const favoriteColumns = `id, name, lat, lon, sublocation, city, state, country, country_code`

// favoriteKey is the uniqueness key: case and whitespace differences do not make a new name.
func favoriteKey(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

type rowScanner interface{ Scan(dest ...any) error }

func scanFavorite(r rowScanner) (Favorite, error) {
	var f Favorite
	err := r.Scan(&f.ID, &f.Name, &f.Lat, &f.Lon, &f.Parts.Sublocation, &f.Parts.City,
		&f.Parts.State, &f.Parts.Country, &f.Parts.CountryCode)
	return f, err
}

// ListFavorites returns all favourites ordered by name; never nil.
func ListFavorites(ctx context.Context, db *sql.DB) ([]Favorite, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+favoriteColumns+` FROM favorite_places ORDER BY name_key, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Favorite{}
	for rows.Next() {
		f, err := scanFavorite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CreateFavorite validates and stores f (its ID is ignored). A name whose key is taken
// yields ErrFavoriteExists; invalid input yields *LocationInputError.
func CreateFavorite(ctx context.Context, db *sql.DB, f Favorite, now time.Time) (Favorite, error) {
	if err := ValidatePoint(f.Lat, f.Lon); err != nil {
		return Favorite{}, err
	}
	name, parts, err := ValidateLocationText(f.Name, f.Parts)
	if err != nil {
		return Favorite{}, err
	}
	res, err := db.ExecContext(ctx,
		`INSERT INTO favorite_places (name, name_key, lat, lon, sublocation, city, state, country, country_code, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name, favoriteKey(name), f.Lat, f.Lon, parts.Sublocation, parts.City, parts.State,
		parts.Country, parts.CountryCode, now.Unix(), now.Unix())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Favorite{}, ErrFavoriteExists
		}
		return Favorite{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Favorite{}, err
	}
	return Favorite{ID: id, Name: name, Lat: f.Lat, Lon: f.Lon, Parts: parts}, nil
}

// RenameFavorite changes only the name. Changing the spelling of an entry's own key is allowed.
func RenameFavorite(ctx context.Context, db *sql.DB, id int64, name string, now time.Time) (Favorite, error) {
	name, _, err := ValidateLocationText(name, Parts{})
	if err != nil {
		return Favorite{}, err
	}
	res, err := db.ExecContext(ctx,
		`UPDATE favorite_places SET name = ?, name_key = ?, updated_at = ? WHERE id = ?`,
		name, favoriteKey(name), now.Unix(), id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Favorite{}, ErrFavoriteExists
		}
		return Favorite{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Favorite{}, err
	}
	if n == 0 {
		return Favorite{}, ErrFavoriteNotFound
	}
	f, err := scanFavorite(db.QueryRowContext(ctx, `SELECT `+favoriteColumns+` FROM favorite_places WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Favorite{}, ErrFavoriteNotFound
	}
	return f, err
}

// DeleteFavorite removes one favourite.
func DeleteFavorite(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx, `DELETE FROM favorite_places WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrFavoriteNotFound
	}
	return nil
}
