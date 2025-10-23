package apierror

// compile-time proof of error interface implementation
var _ error = (*APIError)(nil)

// APIError represents api error
type APIError struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// Error returns api error's error message
func (apiErr *APIError) Error() string {
	return apiErr.Message
}
