package validator

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	gvalidator "github.com/go-playground/validator/v10"
)

func TestRegisterKeepsGinValidatorAvailable(t *testing.T) {
	Register()

	if _, ok := binding.Validator.Engine().(*gvalidator.Validate); !ok {
		t.Fatalf("binding validator engine = %T, want *validator.Validate", binding.Validator.Engine())
	}
}
