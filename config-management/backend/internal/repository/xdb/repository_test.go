package xdb

import (
	"database/sql"
	"testing"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
)

func TestNewDB_ProvidesAllSysRepos(t *testing.T) {
	repo := NewDB((*sql.DB)(nil))
	if repo == nil {
		t.Fatalf("NewDB should return non-nil repository")
	}

	var _ sysrepo.UserRepo = repo.UserRepo()
	var _ sysrepo.RoleRepo = repo.RoleRepo()
	var _ sysrepo.MenuRepo = repo.MenuRepo()
	var _ sysrepo.DictRepo = repo.DictRepo()
}
