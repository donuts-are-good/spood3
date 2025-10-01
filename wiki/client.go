package wiki

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client is a minimal MediaWiki (Fandom) API client supporting login and edits.
type Client struct {
	baseAPI   string
	username  string
	password  string
	http      *http.Client
	csrfToken string
	enabled   bool
}

func New() (*Client, error) {
	// Strictly use hard-coded values from config.go; no environment variables.
	enabled := DefaultEnabled
	c := &Client{enabled: enabled}
	if !enabled {
		return c, nil
	}
	base := DefaultBaseAPI
	if base == "" {
		return nil, errors.New("DefaultBaseAPI is required when wiki sync is enabled")
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid DefaultBaseAPI: %w", err)
	}
	// Ensure it's api.php endpoint
	if !strings.Contains(strings.ToLower(u.Path), "api.php") {
		if strings.HasSuffix(u.Path, "/") {
			u.Path = strings.TrimSuffix(u.Path, "/") + "/api.php"
		} else {
			u.Path = u.Path + "/api.php"
		}
	}
	jar, _ := cookiejar.New(nil)
	c.baseAPI = u.String()
	c.username = DefaultUsername
	c.password = DefaultPassword
	c.http = &http.Client{Timeout: 15 * time.Second, Jar: jar}
	return c, nil
}

func (c *Client) isEnabled() bool { return c != nil && c.enabled }

// EnsureLogin performs login and CSRF token fetch if needed.
func (c *Client) EnsureLogin() error {
	if !c.isEnabled() {
		return nil
	}
	if c.csrfToken != "" {
		return nil
	}
	// Step 1: fetch login token
	loginToken, err := c.fetchToken("login")
	if err != nil {
		return fmt.Errorf("fetch login token: %w", err)
	}
	// Step 2: login
	form := url.Values{
		"action":     {"login"},
		"lgname":     {c.username},
		"lgpassword": {c.password},
		"lgtoken":    {loginToken},
		"format":     {"json"},
	}
	resp, err := c.http.PostForm(c.baseAPI, form)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	var lr struct {
		Login struct {
			Result string `json:"result"`
		} `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return fmt.Errorf("login decode: %w", err)
	}
	if !strings.EqualFold(lr.Login.Result, "Success") {
		return fmt.Errorf("wiki login failed: %s", lr.Login.Result)
	}
	// Step 3: fetch csrf token
	csrf, err := c.fetchToken("csrf")
	if err != nil {
		return fmt.Errorf("fetch csrf token: %w", err)
	}
	c.csrfToken = csrf
	return nil
}

func (c *Client) fetchToken(tokenType string) (string, error) {
	q := url.Values{
		"action": {"query"},
		"meta":   {"tokens"},
		"format": {"json"},
		"type":   {tokenType},
	}
	resp, err := c.http.Get(c.baseAPI + "?" + q.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var tr struct {
		Query struct {
			Tokens map[string]string `json:"tokens"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	// token key names: csrftoken, logintoken
	for k, v := range tr.Query.Tokens {
		if strings.Contains(k, "token") {
			return v, nil
		}
	}
	return "", errors.New("token not found")
}

// AppendText appends text to a page (creates if missing) with an edit summary.
func (c *Client) AppendText(title, text, summary string) error {
	if !c.isEnabled() {
		return nil
	}
	if err := c.EnsureLogin(); err != nil {
		return err
	}
	form := url.Values{
		"action":     {"edit"},
		"title":      {title},
		"appendtext": {text},
		"summary":    {summary},
		"token":      {c.csrfToken},
		"format":     {"json"},
	}
	resp, err := c.http.PostForm(c.baseAPI, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("wiki edit failed: %s", resp.Status)
	}
	return nil
}

// FileExists checks if File:filename exists on the wiki.
func (c *Client) FileExists(filename string) (bool, error) {
	if !c.isEnabled() {
		return true, nil
	}
	if err := c.EnsureLogin(); err != nil {
		return false, err
	}
	q := url.Values{
		"action": {"query"},
		"titles": {"File:" + filename},
		"format": {"json"},
		"prop":   {"imageinfo"},
	}
	resp, err := c.http.Get(c.baseAPI + "?" + q.Encode())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var out struct {
		Query struct {
			Pages map[string]struct {
				Missing interface{} `json:"missing"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	for _, p := range out.Query.Pages {
		// If "missing" key is present, the file doesn't exist
		if p.Missing != nil {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// EnsureFileFromURL uploads if the given file does not already exist.
func (c *Client) EnsureFileFromURL(filename, srcURL, comment string) error {
	exists, err := c.FileExists(filename)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.UploadFromURL(filename, srcURL, comment)
}

// UploadFile uploads a file to the wiki (create or update) using the MediaWiki upload API.
func (c *Client) UploadFile(filename string, data []byte, comment string) error {
	if !c.isEnabled() {
		return nil
	}
	if err := c.EnsureLogin(); err != nil {
		return err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("action", "upload")
	_ = writer.WriteField("filename", filename)
	_ = writer.WriteField("ignorewarnings", "1")
	_ = writer.WriteField("comment", comment)
	_ = writer.WriteField("token", c.csrfToken)
	_ = writer.WriteField("format", "json")
	filePart, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(filePart, bytes.NewReader(data)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.baseAPI, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("wiki upload failed: %s", resp.Status)
	}
	return nil
}

// UploadFromURL downloads a file from a URL and uploads it to the wiki with the given filename.
func (c *Client) UploadFromURL(filename, srcURL, comment string) error {
	if !c.isEnabled() {
		return nil
	}
	if err := c.EnsureLogin(); err != nil {
		return err
	}
	resp, err := c.http.Get(srcURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return c.UploadFile(filename, data, comment)
}

// SetText replaces a page's content (creates if missing) with an edit summary.
func (c *Client) SetText(title, text, summary string) error {
	if !c.isEnabled() {
		return nil
	}
	if err := c.EnsureLogin(); err != nil {
		return err
	}
	form := url.Values{
		"action":  {"edit"},
		"title":   {title},
		"text":    {text},
		"summary": {summary},
		"token":   {c.csrfToken},
		"format":  {"json"},
	}
	resp, err := c.http.PostForm(c.baseAPI, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("wiki edit failed: %s", resp.Status)
	}
	return nil
}
