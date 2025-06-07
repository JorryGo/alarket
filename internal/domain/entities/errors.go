package entities

import "errors"

var (
	ErrInvalidSymbol   = errors.New("invalid symbol")
	ErrInvalidPrice    = errors.New("invalid price")
	ErrInvalidQuantity = errors.New("invalid quantity")
	ErrInvalidSpread   = errors.New("invalid spread: bid price cannot be higher than ask price")
	ErrInvalidAsset    = errors.New("invalid asset")
)
