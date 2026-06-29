package validator

import (
	"github.com/gin-gonic/gin/binding"
	gvalidator "github.com/go-playground/validator/v10"
)

// Register 注册自定义校验规则。
// 在 server 启动时调用一次。
func Register() {
	if v, ok := binding.Validator.Engine().(*gvalidator.Validate); ok {
		_ = v // 在此注册自定义 tag，例如：v.RegisterValidation("notblank", notBlank)
	}
}
