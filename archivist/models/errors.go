package models

import "errors"

var (
	ErrChannelIDTooLong     = errors.New("channel_id is too long")
	ErrHashTooLong          = errors.New("hash is too long")
	ErrPubIDTooLong         = errors.New("publication_id is too long")
	ErrProviderNameTooLong  = errors.New("provider_name is too long")
	ErrURLTooLong           = errors.New("url is too long")
	ErrOriginalTitleTooLong = errors.New("original_title is too long")
	ErrOriginalDescTooLong  = errors.New("original_desc is too long")
	ErrComposedTextTooLong  = errors.New("composed_text is too long")
	ErrOriginalDateEmpty    = errors.New("original_date is empty")
)
