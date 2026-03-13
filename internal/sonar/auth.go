package sonar

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type validateAuthResponse struct {
	Valid bool `json:"valid"`
}

type organizationsSearchResponse struct {
	Organizations []struct {
		Key string `json:"key"`
	} `json:"organizations"`
}

func (c *Client) ValidateAuth(ctx context.Context) error {
	var response validateAuthResponse
	if err := c.do(ctx, http.MethodGet, "/api/authentication/validate", nil, &response); err != nil {
		return err
	}
	if !response.Valid {
		return &HTTPError{StatusCode: http.StatusUnauthorized, Message: "authentication failed"}
	}
	return nil
}

func (c *Client) CheckOrganizationAccess(ctx context.Context, org string) error {
	query := url.Values{}
	query.Set("organizations", org)

	var response organizationsSearchResponse
	if err := c.do(ctx, http.MethodGet, "/api/organizations/search", query, &response); err != nil {
		return err
	}
	if len(response.Organizations) == 0 {
		return &HTTPError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("organization %q was not found or is not accessible", org)}
	}
	return nil
}
