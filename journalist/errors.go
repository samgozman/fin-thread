package journalist

import "fmt"

// providerError is the error type for the NewsProvider.
type providerError struct {
	Err          string
	ProviderName string
}

func (e *providerError) Error() string {
	return fmt.Sprintf("Provider %s error: %s", e.ProviderName, e.Err)
}

// newErrProvider creates a new providerError instance with the given provider name and error message.
func newErrProvider(providerName, err string) *providerError {
	return &providerError{
		Err:          err,
		ProviderName: providerName,
	}
}
