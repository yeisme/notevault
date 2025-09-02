package rule_test

import (
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/yeisme/notevault/pkg/rule"
)

// TestStruct 用于测试 ValidateStruct.
type TestStruct struct {
	Name string `rule:"required"`
	Age  int    `rule:"gte=18"`
}

// TestEngine 测试 Engine 函数返回非 nil 实例.
func TestEngine(t *testing.T) {
	engine := rule.Engine()
	if engine == nil {
		t.Error("Engine() returned nil")
	}
}

// TestValidateStruct 测试 ValidateStruct 对有效和无效结构体的验证.
func TestValidateStruct(t *testing.T) {
	// 有效结构体
	validStruct := TestStruct{Name: "John", Age: 25}

	err := rule.ValidateStruct(validStruct)
	if err != nil {
		t.Errorf("Expected no error for valid struct, got %v", err)
	}

	// 无效结构体：缺少 Name
	invalidStruct1 := TestStruct{Name: "", Age: 25}

	err = rule.ValidateStruct(invalidStruct1)
	if err == nil {
		t.Error("Expected error for invalid struct (missing name), got nil")
	}

	// 无效结构体：Age 小于 18
	invalidStruct2 := TestStruct{Name: "Jane", Age: 16}

	err = rule.ValidateStruct(invalidStruct2)
	if err == nil {
		t.Error("Expected error for invalid struct (age < 18), got nil")
	}
}

// TestValidateVar 测试 ValidateVar 对变量的验证.
func TestValidateVar(t *testing.T) {
	// 有效 email
	err := rule.ValidateVar("test@example.com", "required,email")
	if err != nil {
		t.Errorf("Expected no error for valid email, got %v", err)
	}

	// 无效 email
	err = rule.ValidateVar("invalid-email", "required,email")
	if err == nil {
		t.Error("Expected error for invalid email, got nil")
	}

	// 有效数字
	err = rule.ValidateVar(25, "gte=18")
	if err != nil {
		t.Errorf("Expected no error for valid number, got %v", err)
	}

	// 无效数字
	err = rule.ValidateVar(15, "gte=18")
	if err == nil {
		t.Error("Expected error for invalid number, got nil")
	}
}

// TestRegisterValidation 测试注册自定义验证.
func TestRegisterValidation(t *testing.T) {
	// 注册自定义验证：检查字符串长度是否为偶数
	err := rule.RegisterValidation("even_length", func(fl validator.FieldLevel) bool {
		str, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}

		return len(str)%2 == 0
	})
	if err != nil {
		t.Fatalf("Failed to register validation: %v", err)
	}

	// 测试有效字符串
	err = rule.ValidateVar("test", "even_length")
	if err != nil {
		t.Errorf("Expected no error for even length string, got %v", err)
	}

	// 测试无效字符串
	err = rule.ValidateVar("test1", "even_length")
	if err == nil {
		t.Error("Expected error for odd length string, got nil")
	}
}

// TestRegisterAlias 测试注册别名.
func TestRegisterAlias(t *testing.T) {
	rule.RegisterAlias("min_required", "required,min=3")

	// 测试有效字符串
	err := rule.ValidateVar("abc", "min_required")
	if err != nil {
		t.Errorf("Expected no error for valid string with alias, got %v", err)
	}

	// 测试无效字符串
	err = rule.ValidateVar("ab", "min_required")
	if err == nil {
		t.Error("Expected error for invalid string with alias, got nil")
	}
}
