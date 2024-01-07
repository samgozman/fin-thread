package journalist

import "fmt"

// ProviderError is the error type for the NewsProvider.
type ProviderError struct {
	Err          string
	ProviderName string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("Provider %s error: %s", e.ProviderName, e.Err)
}

// newErrProvider creates a new ProviderError instance with the given provider name and error message.
func newErrProvider(providerName, err string) *ProviderError {
	return &ProviderError{
		Err:          err,
		ProviderName: providerName,
	}
}
