// Package dao 系统管理数据层 SQL Server 实现（D4新增）
package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
)

// DB 包装 *sql.DB，用于系统管理存储层
type DB struct {
	db *sql.DB
}

// NewDB 使用已有的 *sql.DB 构建系统管理存储
func NewDB(db *sql.DB) *DB {
	return &DB{db: db}
}

// UserRepo 返回用户存储实现
func (d *DB) UserRepo() sysrepo.UserRepo { return &userRepo{db: d.db} }

// RoleRepo 返回角色存储实现
func (d *DB) RoleRepo() sysrepo.RoleRepo { return &roleRepo{db: d.db} }

// MenuRepo 返回菜单存储实现
func (d *DB) MenuRepo() sysrepo.MenuRepo { return &menuRepo{db: d.db} }

// DictRepo 返回字典存储实现
func (d *DB) DictRepo() sysrepo.DictRepo { return &dictRepo{db: d.db} }

// ─── userRepo ─────────────────────────────────────────────────────────────────

type userRepo struct{ db *sql.DB }

func (r *userRepo) GetByID(ctx context.Context, id int64) (*sysmodel.SystemUser, error) {
	const q = `
SELECT id, username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time
FROM sys_user WHERE id = @id AND deleted = 0`
	row := r.db.QueryRowContext(ctx, q, sql.Named("id", id))
	return scanUser(row)
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*sysmodel.SystemUser, error) {
	const q = `
SELECT id, username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time
FROM sys_user WHERE username = @username AND deleted = 0`
	row := r.db.QueryRowContext(ctx, q, sql.Named("username", username))
	return scanUser(row)
}

func (r *userRepo) List(ctx context.Context, filter sysmodel.UserFilter, page, pageSize int) ([]*sysmodel.SystemUser, int64, error) {
	args := []interface{}{}
	where := []string{"deleted = 0"}

	if filter.Username != "" {
		where = append(where, "username LIKE @username")
		args = append(args, sql.Named("username", "%"+filter.Username+"%"))
	}
	if filter.RealName != "" {
		where = append(where, "real_name LIKE @realName")
		args = append(args, sql.Named("realName", "%"+filter.RealName+"%"))
	}
	if filter.Status != nil {
		where = append(where, "status = @status")
		args = append(args, sql.Named("status", *filter.Status))
	}
	if filter.DeptID != nil {
		where = append(where, "dept_id = @deptId")
		args = append(args, sql.Named("deptId", *filter.DeptID))
	}

	cond := strings.Join(where, " AND ")
	countQ := fmt.Sprintf("SELECT COUNT(1) FROM sys_user WHERE %s", cond)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	offset := (page - 1) * pageSize
	listQ := fmt.Sprintf(`
SELECT id, username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time
FROM sys_user WHERE %s
ORDER BY create_time DESC
OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, cond, offset, pageSize)

	rows, err := r.db.QueryContext(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*sysmodel.SystemUser
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *userRepo) Create(ctx context.Context, user *sysmodel.SystemUser) (int64, error) {
	const q = `
INSERT INTO sys_user (username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time)
OUTPUT INSERTED.id
VALUES (@username, @password, @realName, @email, @phone, @avatar, @status, @deptId, @now, @now)`
	var id int64
	err := r.db.QueryRowContext(ctx, q,
		sql.Named("username", user.Username),
		sql.Named("password", user.Password),
		sql.Named("realName", user.RealName),
		sql.Named("email", user.Email),
		sql.Named("phone", user.Phone),
		sql.Named("avatar", user.Avatar),
		sql.Named("status", user.Status),
		sql.Named("deptId", user.DeptID),
		sql.Named("now", time.Now()),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	return id, nil
}

func (r *userRepo) Update(ctx context.Context, user *sysmodel.SystemUser) error {
	const q = `
UPDATE sys_user SET real_name=@realName, email=@email, phone=@phone,
    avatar=@avatar, status=@status, dept_id=@deptId, update_time=@now
WHERE id=@id AND deleted=0`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("realName", user.RealName),
		sql.Named("email", user.Email),
		sql.Named("phone", user.Phone),
		sql.Named("avatar", user.Avatar),
		sql.Named("status", user.Status),
		sql.Named("deptId", user.DeptID),
		sql.Named("now", time.Now()),
		sql.Named("id", user.ID),
	)
	return err
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	const q = `UPDATE sys_user SET deleted=1, update_time=@now WHERE id=@id`
	_, err := r.db.ExecContext(ctx, q, sql.Named("now", time.Now()), sql.Named("id", id))
	return err
}

func (r *userRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	const q = `UPDATE sys_user SET password=@pwd, update_time=@now WHERE id=@id AND deleted=0`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("pwd", hash),
		sql.Named("now", time.Now()),
		sql.Named("id", id),
	)
	return err
}

// ─── roleRepo ─────────────────────────────────────────────────────────────────

type roleRepo struct{ db *sql.DB }

func (r *roleRepo) GetByID(ctx context.Context, id int64) (*sysmodel.SystemRole, error) {
	const q = `SELECT id, role_name, role_code, description, status, data_scope, create_time FROM sys_role WHERE id=@id AND deleted=0`
	row := r.db.QueryRowContext(ctx, q, sql.Named("id", id))
	return scanRole(row)
}

func (r *roleRepo) GetByCode(ctx context.Context, code string) (*sysmodel.SystemRole, error) {
	const q = `SELECT id, role_name, role_code, description, status, data_scope, create_time FROM sys_role WHERE role_code=@code AND deleted=0`
	row := r.db.QueryRowContext(ctx, q, sql.Named("code", code))
	return scanRole(row)
}

func (r *roleRepo) List(ctx context.Context, filter sysmodel.RoleFilter, page, pageSize int) ([]*sysmodel.SystemRole, int64, error) {
	args := []interface{}{}
	where := []string{"deleted=0"}
	if filter.RoleName != "" {
		where = append(where, "role_name LIKE @roleName")
		args = append(args, sql.Named("roleName", "%"+filter.RoleName+"%"))
	}
	if filter.Status != nil {
		where = append(where, "status=@status")
		args = append(args, sql.Named("status", *filter.Status))
	}
	cond := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(1) FROM sys_role WHERE %s", cond), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}
	offset := (page - 1) * pageSize
	q := fmt.Sprintf(`SELECT id, role_name, role_code, description, status, data_scope, create_time FROM sys_role WHERE %s ORDER BY create_time DESC OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, cond, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var roles []*sysmodel.SystemRole
	for rows.Next() {
		role, err := scanRoleRow(rows)
		if err != nil {
			return nil, 0, err
		}
		roles = append(roles, role)
	}
	return roles, total, rows.Err()
}

func (r *roleRepo) Create(ctx context.Context, role *sysmodel.SystemRole) (int64, error) {
	const q = `
INSERT INTO sys_role (role_name, role_code, description, status, data_scope, create_time, update_time)
OUTPUT INSERTED.id VALUES (@roleName, @roleCode, @desc, @status, @dataScope, @now, @now)`
	var id int64
	err := r.db.QueryRowContext(ctx, q,
		sql.Named("roleName", role.RoleName),
		sql.Named("roleCode", role.RoleCode),
		sql.Named("desc", role.Description),
		sql.Named("status", role.Status),
		sql.Named("dataScope", role.DataScope),
		sql.Named("now", time.Now()),
	).Scan(&id)
	return id, err
}

func (r *roleRepo) Update(ctx context.Context, role *sysmodel.SystemRole) error {
	const q = `UPDATE sys_role SET role_name=@roleName, description=@desc, status=@status, data_scope=@dataScope, update_time=@now WHERE id=@id AND deleted=0`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("roleName", role.RoleName),
		sql.Named("desc", role.Description),
		sql.Named("status", role.Status),
		sql.Named("dataScope", role.DataScope),
		sql.Named("now", time.Now()),
		sql.Named("id", role.ID),
	)
	return err
}

func (r *roleRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sys_role SET deleted=1, update_time=@now WHERE id=@id`,
		sql.Named("now", time.Now()), sql.Named("id", id))
	return err
}

func (r *roleRepo) GetRolesByUserID(ctx context.Context, userID int64) ([]*sysmodel.SystemRole, error) {
	const q = `
SELECT r.id, r.role_name, r.role_code, r.description, r.status, r.data_scope, r.create_time
FROM sys_role r INNER JOIN sys_user_role ur ON r.id = ur.role_id
WHERE ur.user_id = @userID AND r.deleted = 0 AND r.status = 1`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("userID", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []*sysmodel.SystemRole
	for rows.Next() {
		role, err := scanRoleRow(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *roleRepo) GetMenuIDsByRoleID(ctx context.Context, roleID int64) ([]int64, error) {
	const q = `SELECT menu_id FROM sys_role_menu WHERE role_id = @roleID`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("roleID", roleID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *roleRepo) AssignMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM sys_role_menu WHERE role_id=@roleID`, sql.Named("roleID", roleID)); err != nil {
		return err
	}
	for _, mid := range menuIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO sys_role_menu(role_id,menu_id) VALUES(@roleID,@menuID)`,
			sql.Named("roleID", roleID), sql.Named("menuID", mid)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *roleRepo) AssignUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM sys_user_role WHERE user_id=@userID`, sql.Named("userID", userID)); err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO sys_user_role(user_id,role_id) VALUES(@userID,@roleID)`,
			sql.Named("userID", userID), sql.Named("roleID", rid)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ─── menuRepo ─────────────────────────────────────────────────────────────────

type menuRepo struct{ db *sql.DB }

func (r *menuRepo) GetByID(ctx context.Context, id int64) (*sysmodel.SystemMenu, error) {
	const q = `SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame FROM sys_menu WHERE id=@id AND deleted=0`
	row := r.db.QueryRowContext(ctx, q, sql.Named("id", id))
	return scanMenu(row)
}

func (r *menuRepo) List(ctx context.Context, filter sysmodel.MenuFilter) ([]*sysmodel.SystemMenu, error) {
	args := []interface{}{}
	where := []string{"deleted=0"}
	if filter.MenuName != "" {
		where = append(where, "menu_name LIKE @menuName")
		args = append(args, sql.Named("menuName", "%"+filter.MenuName+"%"))
	}
	if filter.Visible != nil {
		where = append(where, "visible=@visible")
		args = append(args, sql.Named("visible", *filter.Visible))
	}
	q := fmt.Sprintf("SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame FROM sys_menu WHERE %s ORDER BY sort ASC", strings.Join(where, " AND "))
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMenuRows(rows)
}

func (r *menuRepo) GetByUserID(ctx context.Context, userID int64) ([]*sysmodel.SystemMenu, error) {
	const q = `
SELECT DISTINCT m.id, m.parent_id, m.menu_name, m.menu_type, m.path, m.component, m.perms, m.icon, m.sort, m.visible, m.is_frame
FROM sys_menu m
INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
WHERE ur.user_id = @userID AND m.deleted = 0 AND m.visible = 1
ORDER BY m.sort ASC`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("userID", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMenuRows(rows)
}

func (r *menuRepo) Create(ctx context.Context, menu *sysmodel.SystemMenu) (int64, error) {
	const q = `
INSERT INTO sys_menu (parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame, create_time, update_time)
OUTPUT INSERTED.id
VALUES (@parentId, @menuName, @menuType, @path, @component, @perms, @icon, @sort, @visible, @isFrame, @now, @now)`
	var id int64
	visibleVal := 1
	if menu.Visible == 0 {
		visibleVal = 0
	}
	err := r.db.QueryRowContext(ctx, q,
		sql.Named("parentId", menu.ParentID),
		sql.Named("menuName", menu.MenuName),
		sql.Named("menuType", menu.MenuType),
		sql.Named("path", menu.Path),
		sql.Named("component", menu.Component),
		sql.Named("perms", menu.Perms),
		sql.Named("icon", menu.Icon),
		sql.Named("sort", menu.Sort),
		sql.Named("visible", visibleVal),
		sql.Named("isFrame", menu.IsFrame),
		sql.Named("now", time.Now()),
	).Scan(&id)
	return id, err
}

func (r *menuRepo) Update(ctx context.Context, menu *sysmodel.SystemMenu) error {
	const q = `
UPDATE sys_menu SET parent_id=@parentId, menu_name=@menuName, menu_type=@menuType,
    path=@path, component=@component, perms=@perms, icon=@icon,
    sort=@sort, visible=@visible, is_frame=@isFrame, update_time=@now
WHERE id=@id AND deleted=0`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("parentId", menu.ParentID),
		sql.Named("menuName", menu.MenuName),
		sql.Named("menuType", menu.MenuType),
		sql.Named("path", menu.Path),
		sql.Named("component", menu.Component),
		sql.Named("perms", menu.Perms),
		sql.Named("icon", menu.Icon),
		sql.Named("sort", menu.Sort),
		sql.Named("visible", menu.Visible),
		sql.Named("isFrame", menu.IsFrame),
		sql.Named("now", time.Now()),
		sql.Named("id", menu.ID),
	)
	return err
}

func (r *menuRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sys_menu SET deleted=1, update_time=@now WHERE id=@id`,
		sql.Named("now", time.Now()), sql.Named("id", id))
	return err
}

// ─── dictRepo ─────────────────────────────────────────────────────────────────

type dictRepo struct{ db *sql.DB }

func (r *dictRepo) ListTypes(ctx context.Context) ([]string, error) {
	const q = `SELECT DISTINCT dict_type FROM sys_dictionary_info WHERE status=1 AND dict_type != 'distributedworkflow' ORDER BY dict_type`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var types []string
	for rows.Next() {
		var t string
		if err = rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

func (r *dictRepo) ListByType(ctx context.Context, dictType string) ([]*sysmodel.DictionaryItem, error) {
	const q = `SELECT dic_id, dict_type, dict_name, dict_value, dict_group, sort, status, ISNULL(remark,''), create_time FROM sys_dictionary_info WHERE dict_type=@dictType AND status=1 ORDER BY sort ASC`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("dictType", dictType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDictRows(rows)
}

func (r *dictRepo) Page(ctx context.Context, filter sysmodel.DictFilter, page, pageSize int) ([]*sysmodel.DictionaryItem, int64, error) {
	args := []interface{}{}
	where := []string{"1=1"}
	if filter.DictType != "" {
		where = append(where, "dict_type=@dictType")
		args = append(args, sql.Named("dictType", filter.DictType))
	}
	if filter.DictGroup != "" {
		where = append(where, "dict_group=@dictGroup")
		args = append(args, sql.Named("dictGroup", filter.DictGroup))
	}
	if filter.Status != nil {
		where = append(where, "status=@status")
		args = append(args, sql.Named("status", *filter.Status))
	}
	cond := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(1) FROM sys_dictionary_info WHERE %s", cond), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}
	offset := (page - 1) * pageSize
	q := fmt.Sprintf(`SELECT dic_id, dict_type, dict_name, dict_value, dict_group, sort, status, ISNULL(remark,''), create_time FROM sys_dictionary_info WHERE %s ORDER BY sort ASC OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, cond, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanDictRows(rows)
	return items, total, err
}

func (r *dictRepo) Create(ctx context.Context, item *sysmodel.DictionaryItem) (int64, error) {
	const q = `INSERT INTO sys_dictionary_info (dict_type, dict_name, dict_value, dict_group, sort, status, remark, create_time, update_time) OUTPUT INSERTED.dic_id VALUES (@dictType, @dictName, @dictValue, @dictGroup, @sort, @status, @remark, @now, @now)`
	var id int64
	err := r.db.QueryRowContext(ctx, q,
		sql.Named("dictType", item.DictType),
		sql.Named("dictName", item.DictName),
		sql.Named("dictValue", item.DictValue),
		sql.Named("dictGroup", item.DictGroup),
		sql.Named("sort", item.Sort),
		sql.Named("status", item.Status),
		sql.Named("remark", item.Remark),
		sql.Named("now", time.Now()),
	).Scan(&id)
	return id, err
}

func (r *dictRepo) Update(ctx context.Context, item *sysmodel.DictionaryItem) error {
	const q = `UPDATE sys_dictionary_info SET dict_name=@dictName, dict_value=@dictValue, dict_group=@dictGroup, sort=@sort, status=@status, remark=@remark, update_time=@now WHERE dic_id=@dicId`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("dictName", item.DictName),
		sql.Named("dictValue", item.DictValue),
		sql.Named("dictGroup", item.DictGroup),
		sql.Named("sort", item.Sort),
		sql.Named("status", item.Status),
		sql.Named("remark", item.Remark),
		sql.Named("now", time.Now()),
		sql.Named("dicId", item.DicID),
	)
	return err
}

func (r *dictRepo) Delete(ctx context.Context, dicID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sys_dictionary_info WHERE dic_id=@dicId`, sql.Named("dicId", dicID))
	return err
}

// ─── 辅助扫描函数 ─────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(s scanner) (*sysmodel.SystemUser, error) {
	u := &sysmodel.SystemUser{}
	var email, phone, avatar sql.NullString
	var deptID sql.NullInt64
	err := s.Scan(&u.ID, &u.Username, &u.Password, &u.RealName, &email, &phone, &avatar, &u.Status, &deptID, &u.CreateTime, &u.UpdateTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Email = email.String
	u.Phone = phone.String
	u.Avatar = avatar.String
	if deptID.Valid {
		u.DeptID = deptID.Int64
	}
	return u, nil
}

func scanUserRow(rows *sql.Rows) (*sysmodel.SystemUser, error) {
	u := &sysmodel.SystemUser{}
	var email, phone, avatar sql.NullString
	var deptID sql.NullInt64
	err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.RealName, &email, &phone, &avatar, &u.Status, &deptID, &u.CreateTime, &u.UpdateTime)
	if err != nil {
		return nil, fmt.Errorf("scan user row: %w", err)
	}
	u.Email = email.String
	u.Phone = phone.String
	u.Avatar = avatar.String
	if deptID.Valid {
		u.DeptID = deptID.Int64
	}
	return u, nil
}

func scanRole(s scanner) (*sysmodel.SystemRole, error) {
	r := &sysmodel.SystemRole{}
	var desc sql.NullString
	err := s.Scan(&r.ID, &r.RoleName, &r.RoleCode, &desc, &r.Status, &r.DataScope, &r.CreateTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan role: %w", err)
	}
	r.Description = desc.String
	return r, nil
}

func scanRoleRow(rows *sql.Rows) (*sysmodel.SystemRole, error) {
	r := &sysmodel.SystemRole{}
	var desc sql.NullString
	err := rows.Scan(&r.ID, &r.RoleName, &r.RoleCode, &desc, &r.Status, &r.DataScope, &r.CreateTime)
	if err != nil {
		return nil, fmt.Errorf("scan role row: %w", err)
	}
	r.Description = desc.String
	return r, nil
}

func scanMenu(s scanner) (*sysmodel.SystemMenu, error) {
	m := &sysmodel.SystemMenu{}
	var path, component, perms, icon sql.NullString
	err := s.Scan(&m.ID, &m.ParentID, &m.MenuName, &m.MenuType, &path, &component, &perms, &icon, &m.Sort, &m.Visible, &m.IsFrame)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan menu: %w", err)
	}
	m.Path = path.String
	m.Component = component.String
	m.Perms = perms.String
	m.Icon = icon.String
	return m, nil
}

func scanMenuRows(rows *sql.Rows) ([]*sysmodel.SystemMenu, error) {
	var menus []*sysmodel.SystemMenu
	for rows.Next() {
		m := &sysmodel.SystemMenu{}
		var path, component, perms, icon sql.NullString
		if err := rows.Scan(&m.ID, &m.ParentID, &m.MenuName, &m.MenuType, &path, &component, &perms, &icon, &m.Sort, &m.Visible, &m.IsFrame); err != nil {
			return nil, fmt.Errorf("scan menu row: %w", err)
		}
		m.Path = path.String
		m.Component = component.String
		m.Perms = perms.String
		m.Icon = icon.String
		menus = append(menus, m)
	}
	return menus, rows.Err()
}

func scanDictRows(rows *sql.Rows) ([]*sysmodel.DictionaryItem, error) {
	var items []*sysmodel.DictionaryItem
	for rows.Next() {
		item := &sysmodel.DictionaryItem{}
		if err := rows.Scan(&item.DicID, &item.DictType, &item.DictName, &item.DictValue, &item.DictGroup, &item.Sort, &item.Status, &item.Remark, &item.CreateTime); err != nil {
			return nil, fmt.Errorf("scan dict row: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
