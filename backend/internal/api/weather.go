package api

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// WeatherHandler proxies weather requests to Open-Meteo with a 10-minute
// in-memory cache. Public endpoint (no auth) so the home-page widget can
// render without login. Rate-limited at registration site (see main.go).
//
// Why a backend proxy at all rather than letting frontend hit Open-Meteo
// directly:
//   - Single cache shared across all panel viewers (only 1 upstream call per
//     city per 10 min, regardless of how many users view the home page)
//   - User's NAS is whitelisted to reach Open-Meteo; downstream client (e.g.
//     a phone on a metered connection) might not be
//   - Future option: switch upstream provider without rebuilding frontend
//
// Why round lat/lon to 2 decimals before caching: weather doesn't vary on
// the ~1km scale, but a malicious client could spam unrounded lat values to
// blow past the cache and amplify load on Open-Meteo. Rounding collapses
// near-identical requests.
type WeatherHandler struct {
	cache    sync.Map // key "lat,lon" → *weatherCacheEntry
	upstream string   // override for testing; empty → Open-Meteo
}

type weatherCacheEntry struct {
	body      []byte
	fetchedAt time.Time
}

const (
	weatherCacheTTL  = 10 * time.Minute
	weatherUpstream  = "https://api.open-meteo.com/v1/forecast"
	weatherTimeout   = 10 * time.Second
	weatherMaxBytes  = 1 << 20 // 1 MiB upstream response cap
	weatherCoordPrec = 100     // 2 decimals (1/100 degree ≈ 1 km)
)

// GetHandler returns the Gin handler for GET /api/public/weather. Exported so
// main.go can wrap it with rate-limit middleware at registration site.
func (h *WeatherHandler) GetHandler() gin.HandlerFunc {
	return h.get
}

// get accepts ?lat=X&lon=Y and proxies to Open-Meteo's current-weather endpoint.
// On success returns the upstream JSON verbatim. Caches at (rounded lat, rounded lon).
func (h *WeatherHandler) get(c *gin.Context) {
	lat, ok := parseCoord(c.Query("lat"), -90, 90, "lat")
	if !ok {
		Fail(c, http.StatusBadRequest, 400, "invalid lat (must be a number in [-90, 90])")
		return
	}
	lon, ok := parseCoord(c.Query("lon"), -180, 180, "lon")
	if !ok {
		Fail(c, http.StatusBadRequest, 400, "invalid lon (must be a number in [-180, 180])")
		return
	}

	rLat := math.Round(lat*weatherCoordPrec) / weatherCoordPrec
	rLon := math.Round(lon*weatherCoordPrec) / weatherCoordPrec
	key := fmt.Sprintf("%.2f,%.2f", rLat, rLon)

	if cached, ok := h.lookup(key); ok {
		c.Data(http.StatusOK, "application/json", cached)
		return
	}

	upstream := h.upstream
	if upstream == "" {
		upstream = weatherUpstream
	}
	url := fmt.Sprintf("%s?latitude=%v&longitude=%v&current=temperature_2m,weather_code,is_day&timezone=auto",
		upstream, rLat, rLon)

	client := &http.Client{Timeout: weatherTimeout}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "moon-panel/weather")
	resp, err := client.Do(req)
	if err != nil {
		Fail(c, http.StatusBadGateway, 502, fmt.Sprintf("upstream error: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, weatherMaxBytes))
	if err != nil {
		Fail(c, http.StatusBadGateway, 502, "read upstream failed")
		return
	}
	if resp.StatusCode != http.StatusOK {
		Fail(c, http.StatusBadGateway, 502, fmt.Sprintf("upstream returned HTTP %d", resp.StatusCode))
		return
	}

	h.store(key, body)
	c.Data(http.StatusOK, "application/json", body)
}

func (h *WeatherHandler) lookup(key string) ([]byte, bool) {
	v, ok := h.cache.Load(key)
	if !ok {
		return nil, false
	}
	e := v.(*weatherCacheEntry)
	if time.Since(e.fetchedAt) > weatherCacheTTL {
		h.cache.Delete(key)
		return nil, false
	}
	return e.body, true
}

func (h *WeatherHandler) store(key string, body []byte) {
	h.cache.Store(key, &weatherCacheEntry{body: body, fetchedAt: time.Now()})
}

func parseCoord(s string, min, max float64, _ string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if math.IsNaN(v) || math.IsInf(v, 0) || v < min || v > max {
		return 0, false
	}
	return v, true
}
