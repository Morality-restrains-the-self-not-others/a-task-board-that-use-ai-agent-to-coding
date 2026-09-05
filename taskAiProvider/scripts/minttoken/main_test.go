package main

import (
	"strings"
	"testing"
)

// OPT-20260824-066: minttoken staff 分支曾查询/插入表前缀迁移前的旧表名
// marketplace_platformstaff，实际表为 ai_provider_platformstaff，导致
// `go run ./scripts/minttoken staff` 报错。回归断言 SQL 指向当前表名。
func TestStaffSQLReferencesCurrentTableName(t *testing.T) {
	for name, sql := range map[string]string{
		"lookup": staffLookupSQL(),
		"insert": staffInsertSQL(),
	} {
		if strings.Contains(sql, "marketplace_platformstaff") {
			t.Errorf("%s SQL 引用表前缀迁移前的旧表名 marketplace_platformstaff: %s", name, sql)
		}
		if !strings.Contains(sql, platformStaffTable) {
			t.Errorf("%s SQL 未引用当前表名 %s: %s", name, platformStaffTable, sql)
		}
	}
}

func TestVendorSQLReferencesCurrentTableName(t *testing.T) {
	for name, sql := range map[string]string{
		"lookup": vendorLookupSQL(),
		"insert": vendorInsertSQL(),
	} {
		if strings.Contains(sql, "marketplace_vendor") {
			t.Errorf("%s SQL 引用表前缀迁移前的旧表名 marketplace_vendor: %s", name, sql)
		}
		if !strings.Contains(sql, vendorTable) {
			t.Errorf("%s SQL 未引用当前表名 %s: %s", name, vendorTable, sql)
		}
	}
}

func TestMySQLConnectDSNDoesNotAppendSQLitePragma(t *testing.T) {
	in := "user:pass@tcp(127.0.0.1:3306)/task_ai_provider"
	got := mysqlConnectDSN(in)
	if got != in {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "_pragma") {
		t.Fatalf("dsn=%s", got)
	}
}
