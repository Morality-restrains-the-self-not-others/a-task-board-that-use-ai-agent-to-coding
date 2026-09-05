package main

import "testing"

func TestHumanizeStartVmErrorNotEnoughBalance(t *testing.T) {
	in := "SDKError:\n   StatusCode: 502\n   Code: InvalidAccountStatus.NotEnoughBalance\n   Message: Your account does not have enough balance"
	got := humanizeStartVmError(in)
	if got != startVmErrorNotEnoughBalance {
		t.Fatalf("got %q want %q", got, startVmErrorNotEnoughBalance)
	}
}

func TestHumanizeStartVmErrorInvalidAccountStatusNotEnoughBalance(t *testing.T) {
	in := "SDKError:\n   StatusCode: 403\n   Code: InvalidAccountStatus.NotEnoughBalance\n   Message: Your account does not have enough balance"
	got := humanizeStartVmError(in)
	if got != startVmErrorNotEnoughBalance {
		t.Fatalf("got %q want %q", got, startVmErrorNotEnoughBalance)
	}
}

func TestHumanizeStartVmErrorPassthroughUnknown(t *testing.T) {
	in := "调用镜像市场 API 失败: 镜像服务返回错误: 502"
	got := humanizeStartVmError(in)
	if got != in {
		t.Fatalf("got %q want original", got)
	}
}

func TestHumanizeStartVmErrorEmpty(t *testing.T) {
	if got := humanizeStartVmError("  "); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
