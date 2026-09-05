package mysqlmeta

import (
	"strings"
	"testing"
)

func TestStripMySQLClientMeta_RemovesDelimiterAndRewritesTerminator(t *testing.T) {
	in := "-- comment\nDELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND //\n\nDELIMITER ;\nCALL p();\n"
	got := StripMySQLClientMeta(in)
	if strings.Contains(got, "DELIMITER") {
		t.Fatalf("DELIMITER should be stripped, got:\n%s", got)
	}
	if strings.Contains(got, "//") {
		t.Fatalf("custom terminator // should be rewritten, got:\n%s", got)
	}
	if !strings.Contains(got, "END;") {
		t.Fatalf("expected END; terminator, got:\n%s", got)
	}
	if !strings.Contains(got, "CREATE PROCEDURE") || !strings.Contains(got, "CALL p()") {
		t.Fatalf("procedure body should remain, got:\n%s", got)
	}
}

func TestStripMySQLClientMeta_NoDelimiterUnchanged(t *testing.T) {
	in := "CREATE TABLE t (id INT);\n"
	if got := StripMySQLClientMeta(in); got != in {
		t.Fatalf("unexpected rewrite:\nwant %q\ngot  %q", in, got)
	}
}

func TestStripMySQLClientMeta_MultipleDelimiterBlocks(t *testing.T) {
	in := "DELIMITER //\nCREATE PROCEDURE a() BEGIN SELECT 1; END //\nDELIMITER ;\nCALL a();\nDELIMITER //\nCREATE PROCEDURE b() BEGIN SELECT 2; END //\nDELIMITER ;\nCALL b();\n"
	got := StripMySQLClientMeta(in)
	if strings.Contains(got, "DELIMITER") || strings.Contains(got, "//") {
		t.Fatalf("should strip all DELIMITER/terminators, got:\n%s", got)
	}
	if !strings.Contains(got, "CALL a()") || !strings.Contains(got, "CALL b()") {
		t.Fatalf("calls should remain, got:\n%s", got)
	}
}
