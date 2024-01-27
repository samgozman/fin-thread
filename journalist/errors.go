package journalist

import (
	"errors"
	"fmt"
	"github.com/samgozman/fin-thread/pkg/errlvl"
)

var (
	errFetchingNews      = errors.New("failed to fetch news")
	errMarshalNewsList   = errors.New("failed to marshal NewsList")
	errMarshalSimpleNews = errors.New("failed to marshal simpleNews")
)

// Error is the error type for the Journalist.
type Error struct {
	level        errlvl.Lvl // severity level of the error
	errs         []error
	providerName string
}

func (e *Error) Error() string {
	err := errors.Join(e.errs...)

	if e.providerName != "" {
		return errlvl.Wrap(fmt.Errorf("provider %s: %w", e.providerName, err), e.level).Error()
	}

	return errlvl.Wrap(err, e.level).Error()
}

func (e *Error) WithProvider(providerName string) *Error {
	e.providerName = providerName
	return e
}

// newError creates a new Error instance.
func newError(lvl errlvl.Lvl, errs ...error) *Error {
	return &Error{
		level: lvl,
		errs:  errs,
	}
}
