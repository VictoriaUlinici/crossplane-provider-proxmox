package proxmoxclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	proxmoxv1alpha1 "provider-proxmox/api/v1alpha1"
)

type ProxmoxClient struct {
	Endpoint   string
	Ticket     string
	CSRFToken  string
	HTTPClient *http.Client
}

const (
	StatusRunning  = "running"
	StatusCreating = "creating"
	StatusDeleting = "deleting"
)

func NewClientWithCredentials(endpoint, username, password string) (*ProxmoxClient, error) {
	authURL := fmt.Sprintf("%s/api2/json/access/ticket", endpoint)
	authPayload := fmt.Sprintf("username=%s&password=%s", username, password)

	req, err := http.NewRequest("POST", authURL, bytes.NewBufferString(authPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Proxmox API: %w", err)
	}
	defer resp.Body.Close()

	var authResponse struct {
		Data struct {
			Ticket              string `json:"ticket"`
			CSRFPreventionToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return nil, fmt.Errorf("failed to parse authentication response: %w", err)
	}

	return &ProxmoxClient{
		Endpoint:   endpoint,
		Ticket:     authResponse.Data.Ticket,
		CSRFToken:  authResponse.Data.CSRFPreventionToken,
		HTTPClient: client,
	}, nil
}

func (c *ProxmoxClient) HealthCheck() error {
	url := fmt.Sprintf("%s/api2/json/nodes", c.Endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	req.Header.Set("Cookie", "PVEAuthCookie="+c.Ticket)
	req.Header.Set("CSRFPreventionToken", c.CSRFToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *ProxmoxClient) Request(method, urlPath string, payload interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.Endpoint, urlPath)
	var body io.Reader

	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "PVEAuthCookie="+c.Ticket)
	req.Header.Set("CSRFPreventionToken", c.CSRFToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to Proxmox API failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *ProxmoxClient) GetVMStatus(ctx context.Context, vmid int) (*VMStatus, error) {
	resp, err := c.Request("GET", fmt.Sprintf("/api2/json/nodes/pve/qemu/%d/status/current", vmid), nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("VM with ID %d not found", vmid)
		}
		return nil, err
	}
	defer resp.Body.Close()

	var statusResponse struct {
		Data VMStatus `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statusResponse); err != nil {
		return nil, fmt.Errorf("failed to parse VM status response: %w", err)
	}

	return &statusResponse.Data, nil
}

type VMStatus struct {
	Status   string `json:"status"`
	Hostname string `json:"name"`
	ID       int    `json:"vmid"`
}

func (vs *VMStatus) ConfigurationMatches(spec proxmoxv1alpha1.VirtualMachineSpec) bool {
	return vs.Hostname == spec.Name &&
		vs.Status == spec.OSType // Assicurarsi che il campo `OSType` sia corretto
}

func (c *ProxmoxClient) Create(payload map[string]interface{}) error {
	resp, err := c.Request("POST", "/api2/json/nodes/pve/qemu", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *ProxmoxClient) Update(vmid int, payload map[string]interface{}) error {
	resp, err := c.Request("PUT", fmt.Sprintf("/api2/json/nodes/pve/qemu/%d/config", vmid), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *ProxmoxClient) Delete(vmid int) error {
	resp, err := c.Request("DELETE", fmt.Sprintf("/api2/json/nodes/pve/qemu/%d", vmid), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func IsNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}
