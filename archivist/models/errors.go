package models

import (
	"errors"
	"github.com/samgozman/fin-thread/pkg/errlvl"
)

var (
	errChannelIDTooLong     = errors.New("channel_id is too long")
	errHashTooLong          = errors.New("hash is too long")
	errPubIDTooLong         = errors.New("publication_id is too long")
	errProviderNameTooLong  = errors.New("provider_name is too long")
	errURLTooLong           = errors.New("url is too long")
	errOriginalTitleTooLong = errors.New("original_title is too long")
	errOriginalDescTooLong  = errors.New("original_desc is too long")
	errComposedTextTooLong  = errors.New("composed_text is too long")
	errOriginalDateEmpty    = errors.New("original_date is empty")
	errTitleTooLong         = errors.New("title is too long")
	errURLEmpty             = errors.New("url is empty")
	errEventValidation      = errors.New("event validation failed")
	errEventCreation        = errors.New("event creation failed")
	errEventUpdate          = errors.New("event update failed")
	errFindRecentEvents     = errors.New("failed to find recent events")
	errFindUntilEvents      = errors.New("failed to find events until the given date")
	errNewsValidation       = errors.New("news validation failed")
	errNewsCreation         = errors.New("news creation failed")
	errNewsUpdate           = errors.New("news update failed")
	errNewsFindAllByHash    = errors.New("failed to find news by hash")
	errNewsFindAllByUrls    = errors.New("failed to find news by urls")
	errNewsFindUntil        = errors.New("failed to find news until the given date")
)

type Error struct {
	// severity level of the error
	level errlvl.Lvl
	// errors stack (preferably generic error + the real error)
	errs []error
}

func (e *Error) Error() string {
	return errlvl.Wrap(errors.Join(e.errs...), e.level).Error()
}

// newError creates a new Error instance with the given errors.
func newError(lvl errlvl.Lvl, errs ...error) *Error {
	return &Error{
		level: lvl,
		errs:  errs,
	}
}
