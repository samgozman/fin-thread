package models

import "errors"

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
)
