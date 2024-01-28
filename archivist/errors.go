package archivist

import (
	"errors"
	"github.com/samgozman/fin-thread/pkg/errlvl"
)

// archivistError is a service-level error type.
type archivistError error

var (
	errChannelIDTooLong     archivistError = errors.New("channel_id is too long")
	errHashTooLong          archivistError = errors.New("hash is too long")
	errPubIDTooLong         archivistError = errors.New("publication_id is too long")
	errProviderNameTooLong  archivistError = errors.New("provider_name is too long")
	errURLTooLong           archivistError = errors.New("url is too long")
	errOriginalTitleTooLong archivistError = errors.New("original_title is too long")
	errOriginalDescTooLong  archivistError = errors.New("original_desc is too long")
	errComposedTextTooLong  archivistError = errors.New("composed_text is too long")
	errOriginalDateEmpty    archivistError = errors.New("original_date is empty")
	errTitleTooLong         archivistError = errors.New("title is too long")
	errURLEmpty             archivistError = errors.New("url is empty")
	errEventValidation      archivistError = errors.New("event validation failed")
	errEventCreation        archivistError = errors.New("event creation failed")
	errEventUpdate          archivistError = errors.New("event update failed")
	errFindRecentEvents     archivistError = errors.New("failed to find recent events")
	errFindUntilEvents      archivistError = errors.New("failed to find events until the given date")
	errNewsValidation       archivistError = errors.New("news validation failed")
	errNewsCreation         archivistError = errors.New("news creation failed")
	errNewsUpdate           archivistError = errors.New("news update failed")
	errNewsFindAllByHash    archivistError = errors.New("failed to find news by hash")
	errNewsFindAllByUrls    archivistError = errors.New("failed to find news by urls")
	errNewsFindUntil        archivistError = errors.New("failed to find news until the given date")
	errFailedMigration      archivistError = errors.New("failed to migrate schema")
	errFailedConnection     archivistError = errors.New("failed to connect to database")
)

// newError creates a wrapped error instance with the given errors.
func newError(lvl errlvl.Lvl, genericErr archivistError, err error) error {
	var wrappedErr error
	if err != nil {
		wrappedErr = errlvl.Wrap(errors.Join(genericErr, err), lvl)
	} else {
		wrappedErr = errlvl.Wrap(genericErr, lvl)
	}

	return wrappedErr
}
