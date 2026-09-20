package marketplace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

// HTTPClient implements pubmarket.Client over a network HTTP connection.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient creates a remote marketplace client targeting the specified baseURL.
func NewHTTPClient(baseURL string, client *http.Client) *HTTPClient {
	cleanURL := strings.TrimRight(baseURL, "/")
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPClient{
		baseURL:    cleanURL,
		httpClient: client,
	}
}

// Ensure HTTPClient implements pubmarket.Client.
var _ pubmarket.Client = (*HTTPClient)(nil)

func (c *HTTPClient) Search(ctx context.Context, query pubmarket.SearchQuery) (*pubmarket.SearchResult, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/search")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if query.Keyword != "" {
		q.Set("q", query.Keyword)
	}
	if query.Category != "" {
		q.Set("category", string(query.Category))
	}
	if query.Publisher != "" {
		q.Set("publisher", query.Publisher)
	}
	if query.Limit > 0 {
		q.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Offset > 0 {
		q.Set("offset", strconv.Itoa(query.Offset))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status: %s", resp.Status)
	}

	var res pubmarket.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *HTTPClient) GetPlugin(ctx context.Context, pluginID string) (*pubmarket.PluginDetail, error) {
	reqURL := fmt.Sprintf("%s/api/v1/plugins/%s", c.baseURL, url.PathEscape(pluginID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pubmarket.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get plugin failed with status: %s", resp.Status)
	}

	var detail pubmarket.PluginDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func (c *HTTPClient) GetVersions(ctx context.Context, pluginID string) ([]pubmarket.VersionInfo, error) {
	reqURL := fmt.Sprintf("%s/api/v1/plugins/%s/versions", c.baseURL, url.PathEscape(pluginID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pubmarket.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get versions failed with status: %s", resp.Status)
	}

	var vers []pubmarket.VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&vers); err != nil {
		return nil, err
	}
	return vers, nil
}

func (c *HTTPClient) GetFeatured(ctx context.Context, limit int) ([]pubmarket.PluginSummary, error) {
	reqURL := fmt.Sprintf("%s/api/v1/featured?limit=%d", c.baseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get featured failed with status: %s", resp.Status)
	}

	var items []pubmarket.PluginSummary
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *HTTPClient) GetTrending(ctx context.Context, limit int) ([]pubmarket.PluginSummary, error) {
	reqURL := fmt.Sprintf("%s/api/v1/trending?limit=%d", c.baseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get trending failed with status: %s", resp.Status)
	}

	var items []pubmarket.PluginSummary
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *HTTPClient) GetCategories(ctx context.Context) ([]pubmarket.Category, error) {
	reqURL := fmt.Sprintf("%s/api/v1/categories", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get categories failed with status: %s", resp.Status)
	}

	var cats []pubmarket.Category
	if err := json.NewDecoder(resp.Body).Decode(&cats); err != nil {
		return nil, err
	}
	return cats, nil
}

func (c *HTTPClient) GetPublisher(ctx context.Context, publisherID string) (*pubmarket.PublisherProfile, error) {
	reqURL := fmt.Sprintf("%s/api/v1/publishers/%s", c.baseURL, url.PathEscape(publisherID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pubmarket.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get publisher failed with status: %s", resp.Status)
	}

	var pub pubmarket.PublisherProfile
	if err := json.NewDecoder(resp.Body).Decode(&pub); err != nil {
		return nil, err
	}
	return &pub, nil
}

func (c *HTTPClient) CheckUpdates(ctx context.Context, installed []pubmarket.UpdateCheckItem) ([]pubmarket.UpdateAvailableInfo, error) {
	reqBody, err := json.Marshal(installed)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/api/v1/updates", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check updates failed with status: %s", resp.Status)
	}

	var updates []pubmarket.UpdateAvailableInfo
	if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (c *HTTPClient) DownloadArtifact(
	ctx context.Context,
	pluginID string,
	ver version.SemVer,
	destPath string,
) (*pubmarket.VersionInfo, error) {
	reqURL := fmt.Sprintf("%s/api/v1/plugins/%s/versions/%s/download",
		c.baseURL, url.PathEscape(pluginID), url.PathEscape(ver.String()))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pubmarket.ErrVersionNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %s", resp.Status)
	}

	expectedDigest := strings.Trim(resp.Header.Get("ETag"), "\"")

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	tmpFile := destPath + ".tmp"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return nil, fmt.Errorf("failed to create download temp file: %w", err)
	}

	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)

	written, copyErr := io.Copy(mw, resp.Body)
	_ = f.Close()

	if copyErr != nil {
		_ = os.Remove(tmpFile)
		return nil, fmt.Errorf("failed during artifact streaming: %w", copyErr)
	}

	actualDigest := hex.EncodeToString(hasher.Sum(nil))
	if expectedDigest != "" && actualDigest != expectedDigest {
		_ = os.Remove(tmpFile)
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrDigestMismatch, expectedDigest, actualDigest)
	}

	if err := os.Rename(tmpFile, destPath); err != nil {
		_ = os.Remove(tmpFile)
		return nil, fmt.Errorf("failed moving artifact to destination: %w", err)
	}

	return &pubmarket.VersionInfo{
		Version:        ver,
		ArtifactDigest: actualDigest,
		ArtifactSize:   written,
		DownloadURL:    reqURL,
		Status:         pubmarket.StatusPublished,
	}, nil
}

func (c *HTTPClient) SubmitReview(ctx context.Context, review pubmarket.Review) error {
	reqBody, err := json.Marshal(review)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s/api/v1/plugins/%s/reviews", c.baseURL, url.PathEscape(review.PluginID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("submit review failed with status: %s", resp.Status)
	}
	return nil
}

func (c *HTTPClient) SubmitRating(ctx context.Context, pluginID, authorID string, stars int) (*pubmarket.RatingSummary, error) {
	payload := map[string]any{
		"author_id": authorID,
		"stars":     stars,
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/api/v1/plugins/%s/ratings", c.baseURL, url.PathEscape(pluginID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("submit rating failed with status: %s", resp.Status)
	}

	var sum pubmarket.RatingSummary
	if err := json.NewDecoder(resp.Body).Decode(&sum); err != nil {
		return nil, err
	}
	return &sum, nil
}

func (c *HTTPClient) ResolveDependencies(ctx context.Context, pluginID string, ver version.SemVer) (*pubmarket.DependencyResolutionPlan, error) {
	reqURL := fmt.Sprintf("%s/api/v1/plugins/%s/versions/%s/dependencies",
		c.baseURL, url.PathEscape(pluginID), url.PathEscape(ver.String()))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pubmarket.ErrNotFound
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := errResp["error"]
		if strings.Contains(msg, "missing") {
			return nil, fmt.Errorf("%w: %s", pubmarket.ErrDependencyMissing, msg)
		}
		if strings.Contains(msg, "conflict") {
			return nil, fmt.Errorf("%w: %s", pubmarket.ErrDependencyConflict, msg)
		}
		if strings.Contains(msg, "circular") {
			return nil, fmt.Errorf("%w: %s", pubmarket.ErrCircularDependency, msg)
		}
		return nil, errors.New(msg)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resolve dependencies failed with status: %s", resp.Status)
	}

	var plan pubmarket.DependencyResolutionPlan
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		return nil, err
	}
	return &plan, nil
}
