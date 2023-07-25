package config

import "errors"

var (
	ErrNonPointerArgument        = errors.New("must be a pointer")
	MasterProfileConfNotSetError = errors.New("master profile config not set")
)
