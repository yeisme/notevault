// Package rule 提供结构体和字段验证功能的封装，基于 go-playground/validator 实现.
package rule

import (
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	inst *validator.Validate
	once sync.Once
)

// initValidator 尝试复用 gin 的 validator 引擎；若不可用则新建并注册 tag name 函数.
func initValidator() {
	if engine := binding.Validator.Engine(); engine != nil {
		if v, ok := engine.(*validator.Validate); ok {
			inst = v
			inst.SetTagName("rule")

			return
		}
	}

	inst = validator.New()
	inst.SetTagName("rule")
}

// lazyInit 初始化全局 validator（幂等）.
func lazyInit() {
	once.Do(initValidator)
}

// Engine 返回全局 *validator.Validate，若未初始化则先初始化.
func Engine() *validator.Validate {
	lazyInit()

	return inst
}

// RegisterValidation 代理 RegisterValidation，确保已初始化.
func RegisterValidation(tag string, fn validator.Func, opts ...bool) error {
	lazyInit()

	return inst.RegisterValidation(tag, fn, opts...)
}

// ValidationErrors 是格式化后的验证错误字典，键为字段名（受 RegisterTagNameFunc 影响），值为可读错误信息.
type ValidationErrors map[string]string

// ValidateStruct 对结构体执行完整校验，返回原始 error（可用 Errors 解析）.
func ValidateStruct(s any) error {
	lazyInit()

	return inst.Struct(s)
}

// ValidateVar 按规则对单个变量校验，例如: ValidateVar("abc", "required,email").
func ValidateVar(field any, tag string) error {
	lazyInit()

	return inst.Var(field, tag)
}

// RegisterAlias 包装 RegisterAlias，便于注册别名规则.
func RegisterAlias(alias, rules string) {
	lazyInit()

	inst.RegisterAlias(alias, rules)
}
