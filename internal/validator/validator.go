package validator

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// slugRegex matches valid slugs: lowercase alphanumeric with hyphens, no leading/trailing/consecutive hyphens
var slugRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// validateSlug validates that a string is a valid slug
func validateSlug(fl validator.FieldLevel) bool {
	return slugRegex.MatchString(fl.Field().String())
}

// RegisterCustomValidators registers all custom validators with gin's validator
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("slug", validateSlug)
	}
}
