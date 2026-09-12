package sanity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	http       *http.Client
	projectID  string
	dataset    string
	apiVersion string
	token      string
}

func New(token, projectID, dataset, apiVersion string) *Client {
	return &Client{
		http:       &http.Client{Timeout: 120 * time.Second},
		projectID:  projectID,
		dataset:    dataset,
		apiVersion: apiVersion,
		token:      token,
	}
}

func (c *Client) baseURL() string {
	return "https://" + c.projectID + ".api.sanity.io/" + c.apiVersion
}

func (c *Client) do(ctx context.Context, method, path, contentType string, body []byte) ([]byte, int, error) {
	delays := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}
	for attempt := 0; attempt <= len(delays); attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			if attempt < len(delays) {
				time.Sleep(delays[attempt])
				continue
			}
			return nil, 0, err
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < len(delays) {
				d := delays[attempt]
				if ra, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && ra > 0 {
					d = time.Duration(ra) * time.Second
				}
				time.Sleep(d)
				continue
			}
			return data, resp.StatusCode, fmt.Errorf("sanity api %d", resp.StatusCode)
		}
		return data, resp.StatusCode, nil
	}
	return nil, 0, errors.New("istek tamamlanamadı")
}

type QueryBody struct {
	Query  string         `json:"query"`
	Params map[string]any `json:"params,omitempty"`
}

func (c *Client) Query(ctx context.Context, groq string, params map[string]any, result any) error {
	body, err := json.Marshal(QueryBody{Query: groq, Params: params})
	if err != nil {
		return err
	}
	data, status, err := c.do(ctx, http.MethodPost, "/data/query/"+c.dataset+"?perspective=raw", "application/json", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("sorgu hatası (%d): %s", status, truncate(string(data), 600))
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	return json.Unmarshal(envelope.Result, result)
}

type MutateResult struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Rev  string `json:"revision"`
}

func (c *Client) Mutate(ctx context.Context, mutations []map[string]any, returnIDs bool) ([]MutateResult, error) {
	var results []MutateResult
	for start := 0; start < len(mutations); start += 20 {
		end := start + 20
		if end > len(mutations) {
			end = len(mutations)
		}
		batch := mutations[start:end]
		body, err := json.Marshal(map[string]any{"mutations": batch})
		if err != nil {
			return nil, err
		}
		path := "/data/mutate/" + c.dataset + "?returnIds=" + strconv.FormatBool(returnIDs) + "&visibility=sync"
		data, status, err := c.do(ctx, http.MethodPost, path, "application/json", body)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("mutasyon hatası (%d): %s", status, truncate(string(data), 600))
		}
		var envelope struct {
			Results []MutateResult `json:"results"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			return nil, fmt.Errorf("mutasyon yanıtı okunamadı: %w", err)
		}
		results = append(results, envelope.Results...)
	}
	return results, nil
}

func (c *Client) UploadAsset(ctx context.Context, content []byte, filename, contentType string) (string, error) {
	data, status, err := c.do(ctx, http.MethodPost, "/assets/images/"+c.dataset+"?filename="+url.QueryEscape(filename), contentType, content)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return "", fmt.Errorf("görsel yükleme hatası (%d): %s", status, truncate(string(data), 600))
	}
	var envelope struct {
		Document struct {
			ID string `json:"_id"`
		} `json:"document"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("görsel yanıtı okunamadı: %w", err)
	}
	if envelope.Document.ID == "" {
		return "", errors.New("görsel yüklendi ancak ID dönmedi")
	}
	return envelope.Document.ID, nil
}

func (c *Client) Publish(ctx context.Context, ids []string) error {
	var mutations []map[string]any
	for _, id := range ids {
		mutations = append(mutations, map[string]any{"publish": map[string]any{"id": id}})
	}
	_, err := c.Mutate(ctx, mutations, false)
	return err
}

func (c *Client) Delete(ctx context.Context, ids []string) error {
	var mutations []map[string]any
	for _, id := range ids {
		mutations = append(mutations, map[string]any{"delete": map[string]any{"id": id}})
	}
	_, err := c.Mutate(ctx, mutations, false)
	return err
}

func (c *Client) Revert(ctx context.Context, id, revision string) error {
	mutations := []map[string]any{
		{"revert": map[string]any{"id": id, "revision": revision}},
	}
	_, err := c.Mutate(ctx, mutations, false)
	return err
}

func (c *Client) CreateOrReplace(ctx context.Context, doc map[string]any) (MutateResult, error) {
	mutations := []map[string]any{{"createOrReplace": doc}}
	results, err := c.Mutate(ctx, mutations, true)
	if err != nil {
		return MutateResult{}, err
	}
	if len(results) == 0 {
		return MutateResult{}, errors.New("doküman oluşturulamadı")
	}
	return results[0], nil
}

func (c *Client) Patch(ctx context.Context, id string, patch map[string]any) error {
	mutations := []map[string]any{{"patch": map[string]any{"id": id, "set": patch}}}
	_, err := c.Mutate(ctx, mutations, false)
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
