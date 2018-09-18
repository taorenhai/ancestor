package retry

// Retryable is an interface for conditions which may be retried.
type Retryable interface {
	CanRetry() bool
}
