// repository 包包含相关应用代码。
package repository

import (
	"errors"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestBuildListWhere_Public(t *testing.T) {
	where, args := buildListWhere(" 手机 ", false, true)
	if where != "WHERE deleted_at IS NULL AND status = 1 AND name LIKE ?" {
		t.Fatalf("public where mismatch: %s", where)
	}
	if len(args) != 1 || args[0] != "%手机%" {
		t.Fatalf("public args mismatch: %#v", args)
	}
}

func TestBuildListWhere_AdminIncludeDeleted(t *testing.T) {
	where, args := buildListWhere("", true, false)
	if where != "" {
		t.Fatalf("admin include deleted where mismatch: %s", where)
	}
	if len(args) != 0 {
		t.Fatalf("admin include deleted args mismatch: %#v", args)
	}
}

func TestBuildListWhere_AdminExcludeDeleted(t *testing.T) {
	where, args := buildListWhere("abc", false, false)
	if where != "WHERE deleted_at IS NULL AND name LIKE ?" {
		t.Fatalf("admin exclude deleted where mismatch: %s", where)
	}
	if len(args) != 1 || args[0] != "%abc%" {
		t.Fatalf("admin exclude deleted args mismatch: %#v", args)
	}
}

func TestIsDuplicateEntry(t *testing.T) {
	if !IsDuplicateEntry(&mysqlDriver.MySQLError{Number: 1062}) {
		t.Fatal("expect mysql duplicate entry to be true")
	}
	if IsDuplicateEntry(&mysqlDriver.MySQLError{Number: 1213}) {
		t.Fatal("expect non-duplicate mysql error to be false")
	}
	if IsDuplicateEntry(errors.New("duplicate")) {
		t.Fatal("expect generic error to be false")
	}
	if IsDuplicateEntry(nil) {
		t.Fatal("expect nil error to be false")
	}
}
