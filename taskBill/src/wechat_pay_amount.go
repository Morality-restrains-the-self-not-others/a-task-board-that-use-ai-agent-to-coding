package main

import "fmt"

// wechatNativeMaxFen Native 下单金额上限（分），与历史 4999 元上限对齐。
const wechatNativeMaxFen int64 = 4999 * 100

// wechatNativeAmountFen 将订单金额（分）转为微信支付 Native `amount.total`。
// 官方文档：单位为分、整型、必须大于 0；1 元应填写 100。
// 禁止先取整到「整数元」再乘 100，否则 0.55 元会变成 1.00 元。
func wechatNativeAmountFen(orderCents int64) (int64, error) {
	if orderCents < 1 || orderCents > wechatNativeMaxFen {
		return 0, fmt.Errorf("amount out of range")
	}
	return orderCents, nil
}

// kycAmountYuanCeil 将订单分金额向上取整到元，供 taskAuth KYC 整数元限额接口使用。
// 与微信下单金额无关：0.55 元仍按 1 元做限额校验，但向微信传 55 分。
func kycAmountYuanCeil(orderCents int64) int64 {
	if orderCents <= 0 {
		return 0
	}
	return (orderCents + 99) / 100
}
