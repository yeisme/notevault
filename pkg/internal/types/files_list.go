package types

// ListFilesResponse is the response of listing user's files for a month.
// It reuses ObjectInfo to present file details.
type ListFilesResponse struct {
	Month string       `json:"month"` // e.g. 2025-09
	Files []ObjectInfo `json:"files"`
	Total int          `json:"total"`
}
