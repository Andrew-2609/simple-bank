package api

import (
	"github.com/Andrew-2609/simple-bank/util"
	"github.com/go-playground/validator/v10"
)

var validateCurrency validator.Func = func(fl validator.FieldLevel) bool {
	if currency, ok := fl.Field().Interface().(string); ok {
		return util.IsSupportedCurrency(currency)
	}

	return false
}
