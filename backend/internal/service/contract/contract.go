// Package contract defines the transport contracts for the CMS implementation.
package contract

import "peptide-site/internal/pkg/money"

// Error describes a stable machine-readable failure without backend internals.
type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// Page describes a paginated response with a non-nil Items slice.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// Price exposes decimal strings already converted by the backend.
type Price = money.Price

// RichText contains editor HTML with controlled media identifier references.
// Media IDs are untrusted input and must be verified against HTML and storage.
type RichText struct {
	HTML     string   `json:"html"`
	MediaIDs []string `json:"mediaIds"`
}

// VersionedWrite prevents silently overwriting a concurrent update.
type VersionedWrite struct {
	ExpectedVersion int64 `json:"expectedVersion"`
}
