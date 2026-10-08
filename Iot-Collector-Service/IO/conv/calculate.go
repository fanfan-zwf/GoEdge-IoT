/*
* 日期: 2026.4.16
* 作者: 范范zwf
* 作用: 采集后比例、倍数换算
 */

package conv

import (
	"errors"
	"fmt"
	"math"

	"github.com/expr-lang/expr"
)

// ExpressionCalc 正向换算：用 expr 引擎计算表达式
// conversion 为包含变量 x 的表达式，如 `x*2+10`、`x/100.0*3600`
// 传入 x 值，返回计算结果
func ExpressionCalc(x float64, conversion string) (float64, error) {
	program, err := expr.Compile(conversion, expr.Env(map[string]any{"x": float64(0)}))
	if err != nil {
		return 0, fmt.Errorf("换算表达式编译失败: %w", err)
	}
	result, err := expr.Run(program, map[string]any{"x": x})
	if err != nil {
		return 0, fmt.Errorf("换算表达式执行失败: %w", err)
	}
	// expr 返回 float64 或 int，统一转 float64
	switch v := result.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case float32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("换算表达式返回不支持的类型: %T", result)
	}
}

// ExpressionReverse 反向换算：从 y 反推 x
// 前提：conversion 必须是一元一次方程（线性表达式）
// 原理：两点法提取线性系数，f(0)=b, f(1)=k+b → k=f(1)-f(0)
// 反算：x = (y - b) / k
func ExpressionReverse(y float64, conversion string) (float64, error) {
	// 两点法求线性系数
	b, err := ExpressionCalc(0, conversion)
	if err != nil {
		return 0, err
	}
	f1, err := ExpressionCalc(1, conversion)
	if err != nil {
		return 0, err
	}
	k := f1 - b
	if math.Abs(k) < 1e-15 {
		return 0, errors.New("反向换算失败: 表达式不是有效的一元一次方程(k=0)")
	}
	return (y - b) / k, nil
}

// ConvertWithCalc 换算 + 类型转换一体化函数
// 传入：value — 原始采集值，fromType — 采集类型编码，toType — 目标输出类型编码，conversion — 换算表达式（可为空）
// conversion 为包含变量 x 的表达式，如 `x*2+10`、`x/100.0*3600`
// 返回：转换后的值，错误（nil 表示成功）
// 流程：conversion 非空时先执行表达式计算，再执行类型转换；conversion 为空时直接类型转换
func ConvertWithCalc(value any, fromType int, toType int, conversion string) (any, error) {
	if conversion != "" && conversion != "0" {
		fv, ok := ConvFloat64(value, fromType)
		if !ok {
			return nil, errors.New("转换为float64失败")
		}
		xv, err := ExpressionCalc(fv, conversion)
		if err != nil {
			return nil, err
		}
		fromType = TYPE_FLOAT64 // 换算后类型为 float64
		value = xv
	}

	result, ok := ConvertType(value, fromType, toType)
	if !ok {
		return nil, errors.New("类型转换失败")
	}
	return result, nil
}

// ConvertReverse 反向换算 + 类型转换一体化函数
// 传入：value — 目标值，fromType — 目标类型编码，toType — 原始采集类型编码，conversion — 换算表达式（可为空）
// conversion 为包含变量 x 的表达式，如 `x*2+10`（必须是一元一次方程）
// 返回：反向转换后的值，错误（nil 表示成功）
// 流程：conversion 非空时先执行表达式反向计算，再执行类型转换；conversion 为空时直接类型转换
func ConvertReverse(value any, fromType int, toType int, conversion string) (any, error) {
	if conversion != "" && conversion != "0" {
		fv, ok := ConvFloat64(value, fromType)
		if !ok {
			return nil, errors.New("转换为float64失败")
		}
		xv, err := ExpressionReverse(fv, conversion)
		if err != nil {
			return nil, err
		}
		fromType = TYPE_FLOAT64 // 反算后类型为 float64
		value = xv
	}

	result, ok := ConvertType(value, fromType, toType)
	if !ok {
		return nil, errors.New("类型转换失败")
	}
	return result, nil
}
