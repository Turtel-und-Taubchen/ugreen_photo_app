package config

import "testing"

func TestGeoDefaultsToPublicNominatimAndAllowsNoContact(t *testing.T) {
	g := GeoFromEnv(func(string) string { return "" })
	if g.BaseURL != "https://nominatim.openstreetmap.org" || g.Contact != "" {
		t.Errorf("got %+v", g)
	}
}

func TestGeoURLLosesTrailingSlash(t *testing.T) {
	env := map[string]string{"PC_NOMINATIM_URL": "http://mock:1/", "PC_NOMINATIM_CONTACT": "a@b.c"}
	g := GeoFromEnv(func(k string) string { return env[k] })
	if g.BaseURL != "http://mock:1" || g.Contact != "a@b.c" {
		t.Errorf("got %+v", g)
	}
}
