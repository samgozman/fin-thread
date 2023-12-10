package journalist

import "fmt"

// ErrProvider is the error type for the NewsProvider
type ErrProvider struct {
	Err          string
	ProviderName string
}

func (e *ErrProvider) Error() string {
	return fmt.Sprintf("Provider %s error: %s", e.ProviderName, e.Err)
}

// newErrProvider creates a new ErrProvider instance with the given provider name and error message
func newErrProvider(providerName, err string) *ErrProvider {
	return &ErrProvider{
		Err:          err,
		ProviderName: providerName,
	}
}
