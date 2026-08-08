package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mypaas/internal/config"
)

type CloudflareClient struct {
	cfg *config.Config
}

func NewCloudflareClient(cfg *config.Config) *CloudflareClient {
	return &CloudflareClient{cfg: cfg}
}

type MetricsData struct {
	TotalRequests int `json:"total_requests"`
	Bandwidth     int `json:"bandwidth"`
	Errors        int `json:"errors"`
}

func (c *CloudflareClient) GetProjectMetrics(ctx context.Context, subdomain string) (*MetricsData, error) {
	if c.cfg.CloudflareAPIToken == "" || c.cfg.CloudflareZoneID == "" {
		return nil, fmt.Errorf("cloudflare not configured")
	}

	query := `
		query GetZoneAnalytics($zoneTag: string, $host: string, $datetime: string) {
			viewer {
				zones(filter: {zoneTag: $zoneTag}) {
					httpRequestsAdaptiveGroups(
						limit: 1,
						filter: {clientRequestHTTPHost: $host, datetime_geq: $datetime}
					) {
						count
						sum {
							edgeResponseBytes
						}
					}
					errors: httpRequestsAdaptiveGroups(
						limit: 1,
						filter: {
							clientRequestHTTPHost: $host,
							datetime_geq: $datetime,
							edgeResponseStatus_gt: 399
						}
					) {
						count
					}
				}
			}
		}
	`

	reqBody := map[string]interface{}{
		"query": query,
		"variables": map[string]string{
			"zoneTag":  c.cfg.CloudflareZoneID,
			"host":     subdomain,
			"datetime": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.cloudflare.com/client/v4/graphql", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.CloudflareAPIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cloudflare api returned %d", resp.StatusCode)
	}

	var payload struct {
		Data struct {
			Viewer struct {
				Zones []struct {
					HttpRequestsAdaptiveGroups []struct {
						Count int `json:"count"`
						Sum   struct {
							EdgeResponseBytes int `json:"edgeResponseBytes"`
						} `json:"sum"`
					} `json:"httpRequestsAdaptiveGroups"`
					Errors []struct {
						Count int `json:"count"`
					} `json:"errors"`
				} `json:"zones"`
			} `json:"viewer"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if len(payload.Errors) > 0 {
		return nil, fmt.Errorf("graphql errors: %v", payload.Errors)
	}

	var data MetricsData
	if len(payload.Data.Viewer.Zones) > 0 {
		zone := payload.Data.Viewer.Zones[0]
		for _, g := range zone.HttpRequestsAdaptiveGroups {
			data.TotalRequests += g.Count
			data.Bandwidth += g.Sum.EdgeResponseBytes
		}
		for _, g := range zone.Errors {
			data.Errors += g.Count
		}
	}

	return &data, nil
}
