package health

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// Checker checks for healthiness of elastic search.
type Checker interface {
	http.Handler
	IsHealthy() bool
}

type elasticHealthCheck struct {
	client         *http.Client
	elasticAddress string
}

// NewElasticHealthChecker returns a new HTTP handler that returns a status 200
// if elastic search is healthy.
func NewElasticHealthChecker(elasticAddress string) Checker {
	return &elasticHealthCheck{
		client: &http.Client{
			Timeout: time.Second * 2,
		},
		elasticAddress: elasticAddress,
	}
}

func (h *elasticHealthCheck) IsHealthy() bool {
	u, err := url.Parse(h.elasticAddress)
	if err != nil {
		return false
	}
	u.Path = "_cluster/health"
	resp, err := h.client.Get(u.String())
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	type healthResponse struct {
		Status string `json:"status"`
	}
	var hr healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return false
	}
	return hr.Status == "green" || hr.Status == "yellow"
}

func (h *elasticHealthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.IsHealthy() {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}
