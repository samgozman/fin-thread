package journalist

import "fmt"

// providerError is the error type for the NewsProvider.
type providerError struct {
	Err          error
	ProviderName string
}

func (e *providerError) Error() string {
	return fmt.Errorf("provider %s error: %w", e.ProviderName, e.Err).Error()
}

func (e *providerError) Unwrap() error {
	return e.Err
}

// newErrProvider creates a new providerError instance with the given provider name and error message.
func newErrProvider(providerName string, err error) *providerError {
	return &providerError{
		Err:          err,
		ProviderName: providerName,
	}
}
