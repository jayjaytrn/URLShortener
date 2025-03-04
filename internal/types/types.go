package types

// URLData represents the structure of a URL record, containing both the original URL
// and the shortened version, along with a flag indicating whether it has been deleted.
type URLData struct {
	UserID      string `json:"user_id,omitempty"`      // UserID is the identifier for the user who created the shortened URL
	ShortURL    string `json:"short_url,omitempty"`    // ShortURL is the shortened version of the original URL
	OriginalURL string `json:"original_url,omitempty"` // OriginalURL is the URL that was shortened
	DeletedFlag bool   `json:"is_deleted"`             // DeletedFlag indicates whether the URL has been deleted
}

// ShortenRequest represents the incoming request to shorten a URL.
type ShortenRequest struct {
	URL string `json:"url"` // URL is the original URL to be shortened
}

// ShortenResponse represents the response for a URL shortening request, containing the result.
type ShortenResponse struct {
	Result string `json:"result"` // Result is the shortened URL
}

// ShortenBatchRequest represents a batch request to shorten multiple URLs, where each request
// has a unique correlation ID to trace the batch.
type ShortenBatchRequest struct {
	CorrelationID string `json:"correlation_id"` // CorrelationID is a unique identifier for the batch request
	OriginalURL   string `json:"original_url"`   // OriginalURL is the URL to be shortened
}

// ShortenBatchResponse represents the response for a batch URL shortening request, containing
// the correlation ID for tracing and the shortened URL.
type ShortenBatchResponse struct {
	CorrelationID string `json:"correlation_id"` // CorrelationID is a unique identifier for the batch request
	ShortURL      string `json:"short_url"`      // ShortURL is the shortened version of the provided URL
}
