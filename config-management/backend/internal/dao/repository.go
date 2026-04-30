package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// User operations
func (r *Repository) GetUserByID(ctx context.Context, id int64) (*sysmodel.SysUser, error) {
	var user sysmodel.SysUser
	const q = `SELECT id, username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time 
	FROM workflow_sys_user WHERE id = @id AND deleted = 0`
	err := r.db.QueryRowContext(ctx, q, sql.Named("id", id)).Scan(
		&user.ID, &user.Username, &user.Password, &user.RealName, &user.Email, &user.Phone,
		&user.Avatar, &user.Status, &user.DeptID, &user.CreateTime, &user.UpdateTime,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*sysmodel.SysUser, error) {
	var user sysmodel.SysUser
	const q = `SELECT id, username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time 
	FROM workflow_sys_user WHERE username = @username AND deleted = 0`
	err := r.db.QueryRowContext(ctx, q, sql.Named("username", username)).Scan(
		&user.ID, &user.Username, &user.Password, &user.RealName, &user.Email, &user.Phone,
		&user.Avatar, &user.Status, &user.DeptID, &user.CreateTime, &user.UpdateTime,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) QueryUserCount(ctx context.Context, cond string, args ...interface{}) (int64, error) {
	var total int64
	countQ := fmt.Sprintf("SELECT COUNT(1) FROM workflow_sys_user WHERE %s", cond)
	err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *Repository) QueryUsers(ctx context.Context, cond string, offset, pageSize int, args ...interface{}) ([]*sysmodel.SysUser, error) {
	q := fmt.Sprintf(`SELECT id, username, real_name, email, phone, avatar, status, dept_id, create_time, update_time 
	FROM workflow_sys_user WHERE %s ORDER BY create_time DESC OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, cond, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SysUser
	for rows.Next() {
		user := &sysmodel.SysUser{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.RealName, &user.Email, &user.Phone,
			&user.Avatar, &user.Status, &user.DeptID, &user.CreateTime, &user.UpdateTime,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, user)
	}
	return result, nil
}

func (r *Repository) CreateUser(ctx context.Context, user *sysmodel.SysUser) error {
	const q = `INSERT INTO workflow_sys_user (username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time)
	VALUES (@username, @password, @realName, @email, @phone, @avatar, @status, @deptId, @createTime, @updateTime)`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("username", user.Username),
		sql.Named("password", user.Password),
		sql.Named("realName", user.RealName),
		sql.Named("email", user.Email),
		sql.Named("phone", user.Phone),
		sql.Named("avatar", user.Avatar),
		sql.Named("status", user.Status),
		sql.Named("deptId", user.DeptID),
		sql.Named("createTime", user.CreateTime),
		sql.Named("updateTime", user.UpdateTime))
	return err
}

func (r *Repository) UpdateUser(ctx context.Context, user *sysmodel.SysUser) error {
	const q = `UPDATE workflow_sys_user SET real_name=@realName, email=@email, phone=@phone,
	avatar=@avatar, status=@status, dept_id=@deptId, update_time=@updateTime WHERE id=@id`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("realName", user.RealName),
		sql.Named("email", user.Email),
		sql.Named("phone", user.Phone),
		sql.Named("avatar", user.Avatar),
		sql.Named("status", user.Status),
		sql.Named("deptId", user.DeptID),
		sql.Named("updateTime", user.UpdateTime),
		sql.Named("id", user.ID))
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, userID int64) error {
	const q = `UPDATE workflow_sys_user SET deleted=1, update_time=@now WHERE id=@id`
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, q, sql.Named("id", userID), sql.Named("now", now))
	return err
}

func (r *Repository) ChangePassword(ctx context.Context, userID int64, pwd string) error {
	const q = `UPDATE workflow_sys_user SET password=@pwd, update_time=@now WHERE id=@id AND deleted=0`
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, q, sql.Named("pwd", pwd), sql.Named("id", userID), sql.Named("now", now))
	return err
}

// Role operations
func (r *Repository) GetRoleByID(ctx context.Context, id int64) (*sysmodel.SysRole, error) {
	var role sysmodel.SysRole
	const q = `SELECT id, role_name, role_code, description, status, data_scope, create_time FROM workflow_sys_role WHERE id=@id AND deleted=0`
	err := r.db.QueryRowContext(ctx, q, sql.Named("id", id)).Scan(
		&role.ID, &role.RoleName, &role.RoleCode, &role.Description, &role.Status, &role.DataScope, &role.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) GetRoleByCode(ctx context.Context, code string) (*sysmodel.SysRole, error) {
	var role sysmodel.SysRole
	const q = `SELECT id, role_name, role_code, description, status, data_scope, create_time FROM workflow_sys_role WHERE role_code=@code AND deleted=0`
	err := r.db.QueryRowContext(ctx, q, sql.Named("code", code)).Scan(
		&role.ID, &role.RoleName, &role.RoleCode, &role.Description, &role.Status, &role.DataScope, &role.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) QueryRoleCount(ctx context.Context, cond string, args ...interface{}) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(1) FROM workflow_sys_role WHERE %s", cond), args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *Repository) QueryRoles(ctx context.Context, cond string, offset, pageSize int, args ...interface{}) ([]*sysmodel.SysRole, error) {
	q := fmt.Sprintf(`SELECT id, role_name, role_code, description, status, data_scope, create_time FROM workflow_sys_role WHERE %s ORDER BY create_time DESC OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, cond, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SysRole
	for rows.Next() {
		role := &sysmodel.SysRole{}
		err := rows.Scan(
			&role.ID, &role.RoleName, &role.RoleCode, &role.Description, &role.Status, &role.DataScope, &role.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, role)
	}
	return result, nil
}

func (r *Repository) CreateRole(ctx context.Context, role *sysmodel.SysRole) error {
	const q = `INSERT INTO workflow_sys_role (role_name, role_code, description, status, data_scope, create_time, update_time)
	VALUES (@roleName, @roleCode, @desc, @status, @dataScope, @createTime, @updateTime)`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("roleName", role.RoleName),
		sql.Named("roleCode", role.RoleCode),
		sql.Named("desc", role.Description),
		sql.Named("status", role.Status),
		sql.Named("dataScope", role.DataScope),
		sql.Named("createTime", role.CreateTime),
		sql.Named("updateTime", role.UpdateTime))
	return err
}

func (r *Repository) UpdateRole(ctx context.Context, role *sysmodel.SysRole) error {
	const q = `UPDATE workflow_sys_role SET role_name=@roleName, description=@desc, status=@status, data_scope=@dataScope, update_time=@now WHERE id=@id AND deleted=0`
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("roleName", role.RoleName),
		sql.Named("desc", role.Description),
		sql.Named("status", role.Status),
		sql.Named("dataScope", role.DataScope),
		sql.Named("now", now),
		sql.Named("id", role.ID))
	return err
}

func (r *Repository) DeleteRole(ctx context.Context, roleID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE workflow_sys_role SET deleted=1, update_time=@now WHERE id=@id`,
		sql.Named("id", roleID), sql.Named("now", time.Now().Format(time.RFC3339)))
	return err
}

func (r *Repository) GetUserRoles(ctx context.Context, userID int64) ([]*sysmodel.SysRole, error) {
	q := `SELECT r.id, r.role_name, r.role_code, r.description, r.status, r.data_scope, r.create_time
	FROM workflow_sys_role r INNER JOIN workflow_sys_user_role ur ON r.id = ur.role_id
	WHERE ur.user_id = @userID AND r.deleted=0`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("userID", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SysRole
	for rows.Next() {
		role := &sysmodel.SysRole{}
		err := rows.Scan(
			&role.ID, &role.RoleName, &role.RoleCode, &role.Description, &role.Status, &role.DataScope, &role.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, role)
	}
	return result, nil
}

func (r *Repository) UpdateUserRoleRelations(ctx context.Context, userID int64, roleIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx, `DELETE FROM workflow_sys_user_role WHERE user_id=@userID`, sql.Named("userID", userID)); err != nil {
		return err
	}

	for _, roleID := range roleIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_sys_user_role(user_id,role_id) VALUES(@userID,@roleID)`,
			sql.Named("userID", userID), sql.Named("roleID", roleID)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Menu operations
func (r *Repository) GetMenuByID(ctx context.Context, id int64) (*sysmodel.SysMenu, error) {
	var menu sysmodel.SysMenu
	const q = `SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame FROM workflow_sys_menu WHERE id=@id AND deleted=0`
	err := r.db.QueryRowContext(ctx, q, sql.Named("id", id)).Scan(
		&menu.ID, &menu.ParentID, &menu.MenuName, &menu.MenuType, &menu.Path, &menu.Component,
		&menu.Perms, &menu.Icon, &menu.Sort, &menu.Visible, &menu.IsFrame,
	)
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *Repository) QueryMenus(ctx context.Context, conditions map[string]interface{}) ([]*sysmodel.SysMenu, error) {
	where := []string{"deleted=0"} // 添加软删除条件
	args := []interface{}{}

	for k, v := range conditions {
		switch k {
		case "parent_id":
			where = append(where, fmt.Sprintf("parent_id = '%v'", v))
		case "menu_type":
			where = append(where, fmt.Sprintf("menu_type = '%v'", v))
		}
	}

	q := fmt.Sprintf("SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame FROM workflow_sys_menu WHERE %s ORDER BY sort ASC", strings.Join(where, " AND "))
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SysMenu
	for rows.Next() {
		menu := &sysmodel.SysMenu{}
		err := rows.Scan(
			&menu.ID, &menu.ParentID, &menu.MenuName, &menu.MenuType, &menu.Path, &menu.Component,
			&menu.Perms, &menu.Icon, &menu.Sort, &menu.Visible, &menu.IsFrame,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, menu)
	}
	return result, nil
}

func (r *Repository) GetUserMenus(ctx context.Context, userID int64) ([]*sysmodel.SysMenu, error) {
	q := `SELECT DISTINCT m.id, m.parent_id, m.menu_name, m.menu_type, m.path, m.component, m.perms, m.icon, m.sort, m.visible, m.is_frame
	FROM workflow_sys_menu m
	INNER JOIN workflow_sys_role_menu rm ON m.id = rm.menu_id
	INNER JOIN workflow_sys_user_role ur ON rm.role_id = ur.role_id
	WHERE ur.user_id = @userID AND m.deleted=0
	ORDER BY m.sort ASC`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("userID", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SysMenu
	for rows.Next() {
		menu := &sysmodel.SysMenu{}
		err := rows.Scan(
			&menu.ID, &menu.ParentID, &menu.MenuName, &menu.MenuType, &menu.Path, &menu.Component,
			&menu.Perms, &menu.Icon, &menu.Sort, &menu.Visible, &menu.IsFrame,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, menu)
	}
	return result, nil
}

func (r *Repository) CreateMenu(ctx context.Context, menu *sysmodel.SysMenu) error {
	const q = `INSERT INTO workflow_sys_menu (parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame, create_time, update_time)
	VALUES (@parentId, @menuName, @menuType, @path, @component, @perms, @icon, @sort, @visible, @isFrame, @createTime, @updateTime)`
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
		sql.Named("createTime", menu.CreateTime),
		sql.Named("updateTime", menu.UpdateTime))
	return err
}

func (r *Repository) UpdateMenu(ctx context.Context, menu *sysmodel.SysMenu) error {
	const q = `UPDATE workflow_sys_menu SET parent_id=@parentId, menu_name=@menuName, menu_type=@menuType,
	path=@path, component=@component, perms=@perms, icon=@icon, sort=@sort, visible=@visible, is_frame=@isFrame, update_time=@updateTime WHERE id=@id`
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
		sql.Named("updateTime", menu.UpdateTime),
		sql.Named("id", menu.ID))
	return err
}

func (r *Repository) DeleteMenu(ctx context.Context, menuID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE workflow_sys_menu SET deleted=1, update_time=@now WHERE id=@id`,
		sql.Named("id", menuID), sql.Named("now", time.Now().Format(time.RFC3339)))
	return err
}

// Dictionary operations
func (r *Repository) GetDictTypes(ctx context.Context) ([]string, error) {
	const q = `SELECT DISTINCT dict_type FROM workflow_sys_dictionary_info WHERE status=1 AND dict_type != 'distributedworkflow' ORDER BY dict_type`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var dictType string
		err := rows.Scan(&dictType)
		if err != nil {
			return nil, err
		}
		result = append(result, dictType)
	}
	return result, nil
}

func (r *Repository) GetDictByType(ctx context.Context, dictType string) ([]*sysmodel.SysDictionaryInfo, error) {
	const q = `SELECT dic_id, dict_type, dict_name, dict_value, dict_group, sort, status, ISNULL(remark,''), create_time FROM workflow_sys_dictionary_info WHERE dict_type=@dictType AND status=1 ORDER BY sort ASC`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("dictType", dictType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SysDictionaryInfo
	for rows.Next() {
		dict := &sysmodel.SysDictionaryInfo{}
		err := rows.Scan(
			&dict.DicID, &dict.DictType, &dict.DictName, &dict.DictValue, &dict.DictGroup,
			&dict.Sort, &dict.Status, &dict.Remark, &dict.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, dict)
	}
	return result, nil
}

func (r *Repository) QueryDictCount(ctx context.Context, cond string, args ...interface{}) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(1) FROM workflow_sys_dictionary_info WHERE %s", cond), args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *Repository) QueryDict(ctx context.Context, cond string, offset, pageSize int, args ...interface{}) ([]*sysmodel.SysDictionaryInfo, error) {
	q := fmt.Sprintf(`SELECT dic_id, dict_type, dict_name, dict_value, dict_group, sort, status, ISNULL(remark,''), create_time FROM workflow_sys_dictionary_info WHERE %s ORDER BY sort ASC OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, cond, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SysDictionaryInfo
	for rows.Next() {
		dict := &sysmodel.SysDictionaryInfo{}
		err := rows.Scan(
			&dict.DicID, &dict.DictType, &dict.DictName, &dict.DictValue, &dict.DictGroup,
			&dict.Sort, &dict.Status, &dict.Remark, &dict.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, dict)
	}
	return result, nil
}

func (r *Repository) CreateDict(ctx context.Context, dict *sysmodel.SysDictionaryInfo) error {
	const q = `INSERT INTO workflow_sys_dictionary_info (dict_type, dict_name, dict_value, dict_group, sort, status, remark, create_time, update_time) OUTPUT INSERTED.dic_id VALUES (@dictType, @dictName, @dictValue, @dictGroup, @sort, @status, @remark, @now, @now)`
	now := time.Now().Format(time.RFC3339)
	row := r.db.QueryRowContext(ctx, q,
		sql.Named("dictType", dict.DictType),
		sql.Named("dictName", dict.DictName),
		sql.Named("dictValue", dict.DictValue),
		sql.Named("dictGroup", dict.DictGroup),
		sql.Named("sort", dict.Sort),
		sql.Named("status", dict.Status),
		sql.Named("remark", dict.Remark),
		sql.Named("now", now))
	return row.Scan(&dict.DicID)
}

func (r *Repository) UpdateDict(ctx context.Context, dict *sysmodel.SysDictionaryInfo) error {
	const q = `UPDATE workflow_sys_dictionary_info SET dict_name=@dictName, dict_value=@dictValue, dict_group=@dictGroup, sort=@sort, status=@status, remark=@remark, update_time=@now WHERE dic_id=@dicId`
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("dictName", dict.DictName),
		sql.Named("dictValue", dict.DictValue),
		sql.Named("dictGroup", dict.DictGroup),
		sql.Named("sort", dict.Sort),
		sql.Named("status", dict.Status),
		sql.Named("remark", dict.Remark),
		sql.Named("now", now),
		sql.Named("dicId", dict.DicID))
	return err
}

func (r *Repository) DeleteDict(ctx context.Context, dicID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workflow_sys_dictionary_info WHERE dic_id=@dicId`, sql.Named("dicId", dicID))
	return err
}

// Role-Menu operations
func (r *Repository) GetRoleMenuIDs(ctx context.Context, roleID int64) ([]int64, error) {
	const q = `SELECT menu_id FROM workflow_sys_role_menu WHERE role_id = @roleID`
	rows, err := r.db.QueryContext(ctx, q, sql.Named("roleID", roleID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menuIDs []int64
	for rows.Next() {
		var menuID int64
		err := rows.Scan(&menuID)
		if err != nil {
			return nil, err
		}
		menuIDs = append(menuIDs, menuID)
	}
	return menuIDs, nil
}

func (r *Repository) UpdateRoleMenuRelations(ctx context.Context, roleID int64, menuIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx, `DELETE FROM workflow_sys_role_menu WHERE role_id=@roleID`, sql.Named("roleID", roleID)); err != nil {
		return err
	}

	for _, menuID := range menuIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_sys_role_menu(role_id,menu_id) VALUES(@roleID,@menuID)`,
			sql.Named("roleID", roleID), sql.Named("menuID", menuID)); err != nil {
			return err
		}
	}

	return tx.Commit()
}