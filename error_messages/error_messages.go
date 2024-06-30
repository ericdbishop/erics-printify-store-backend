package error_messages

import "errors"

var (
	ErrDuplicate    = errors.New("record already exists")
	ErrNotExists    = errors.New("row not exists")
	ErrUpdateFailed = errors.New("update failed")
	ErrDeleteFailed = errors.New("delete failed")

	ErrInvalidItem = errors.New("invalid item")
	ErrInvalidName = errors.New("invalid customer name")
)
