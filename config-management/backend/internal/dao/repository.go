package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
)

// Repository 系统管理数据访问对象
type Repository struct {
	db *sql.DB
}

// NewDB 创建数据库访问对象
func NewDB(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// UserRepo 用户相关数据访问
func (r *Repository) UserRepo() sysrepo.UserRepo {
	return &UserRepository{db: r.db}
}

// RoleRepo 角色相关数据访问
func (r *Repository) RoleRepo() sysrepo.RoleRepo {
	return &RoleRepository{db: r.db}
}

// MenuRepo 菜单相关数据访问
func (r *Repository) MenuRepo() sysrepo.MenuRepo {
	return &MenuRepository{db: r.db}
}

// DictRepo 数据字典相关数据访问
func (r *Repository) DictRepo() sysrepo.DictRepo {
	return &DictRepository{db: r.db}
}

// UserRepository 用户数据访问对象
type UserRepository struct {
	db *sql.DB
}

// GetByID 根据ID获取用户
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*sysmodel.SystemUser, error) {
	var user sysmodel.SystemUser
	query := `SELECT id, username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time
	FROM workflow_sys_user WHERE id = @id`
	err := r.db.QueryRowContext(ctx, query, sql.Named("id", id)).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.RealName,
		&user.Email,
		&user.Phone,
		&user.Avatar,
		&user.Status,
		&user.DeptID,
		&user.CreateTime,
		&user.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*sysmodel.SystemUser, error) {
	var user sysmodel.SystemUser
	query := `SELECT id, username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time
	FROM workflow_sys_user WHERE username = @username`
	err := r.db.QueryRowContext(ctx, query, sql.Named("username", username)).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.RealName,
		&user.Email,
		&user.Phone,
		&user.Avatar,
		&user.Status,
		&user.DeptID,
		&user.CreateTime,
		&user.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// List 分页查询用户
func (r *UserRepository) List(ctx context.Context, filter sysmodel.UserFilter, page, pageSize int) ([]*sysmodel.SystemUser, int64, error) {
	where := "1=1"
	args := []interface{}{}

	if filter.Username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+filter.Username+"%")
	}
	if filter.RealName != "" {
		where += " AND real_name LIKE ?"
		args = append(args, "%"+filter.RealName+"%")
	}
	if filter.Status != nil {
		where += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if filter.DeptID != nil {
		where += " AND dept_id = ?"
		args = append(args, *filter.DeptID)
	}

	totalQuery := fmt.Sprintf("SELECT COUNT(1) FROM workflow_sys_user WHERE %s", where)
	var total int64
	err := r.db.QueryRowContext(ctx, totalQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := page * pageSize
	query := fmt.Sprintf(`SELECT id, username, real_name, email, phone, avatar, status, dept_id, create_time, update_time
	FROM workflow_sys_user WHERE %s
	ORDER BY create_time DESC
	OFFSET %d ROWS
	FETCH NEXT %d ROWS ONLY`, where, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*sysmodel.SystemUser
	for rows.Next() {
		user := &sysmodel.SystemUser{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.RealName,
			&user.Email,
			&user.Phone,
			&user.Avatar,
			&user.Status,
			&user.DeptID,
			&user.CreateTime,
			&user.UpdateTime,
		)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, user)
	}
	return result, total, nil
}

// Create 创建用户并返回ID
func (r *UserRepository) Create(ctx context.Context, user *sysmodel.SystemUser) (int64, error) {
	query := `INSERT INTO workflow_sys_user (username, password, real_name, email, phone, avatar, status, dept_id, create_time, update_time)
	OUTPUT INSERTED.ID
	VALUES (@username, @password, @realName, @email, @phone, @avatar, @status, @deptId, @createTime, @updateTime)`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("username", user.Username),
		sql.Named("password", user.Password),
		sql.Named("realName", user.RealName),
		sql.Named("email", user.Email),
		sql.Named("phone", user.Phone),
		sql.Named("avatar", user.Avatar),
		sql.Named("status", user.Status),
		sql.Named("deptId", user.DeptID),
		sql.Named("createTime", user.CreateTime),
		sql.Named("updateTime", user.UpdateTime)).Scan(&id)
	return id, err
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, user *sysmodel.SystemUser) error {
	query := `UPDATE workflow_sys_user
	SET username = @username, real_name = @realName, email = @email, phone = @phone, avatar = @avatar, 
		status = @status, dept_id = @deptId, update_time = @updateTime
	WHERE id = @id`
	_, err := r.db.ExecContext(ctx, query,
		sql.Named("username", user.Username),
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

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM workflow_sys_user WHERE id = @id`
	_, err := r.db.ExecContext(ctx, query, sql.Named("id", id))
	return err
}

// UpdatePassword 更新用户密码
func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	query := `UPDATE workflow_sys_user SET password = @password WHERE id = @id`
	_, err := r.db.ExecContext(ctx, query,
		sql.Named("password", passwordHash),
		sql.Named("id", id))
	return err
}

// RoleRepository 角色数据访问对象
type RoleRepository struct {
	db *sql.DB
}

// GetByID 根据ID获取角色
func (r *RoleRepository) GetByID(ctx context.Context, id int64) (*sysmodel.SystemRole, error) {
	var role sysmodel.SystemRole
	query := `SELECT id, role_name, role_code, description, status, data_scope, create_time
	FROM workflow_sys_role WHERE id = @id`
	err := r.db.QueryRowContext(ctx, query, sql.Named("id", id)).Scan(
		&role.ID,
		&role.RoleName,
		&role.RoleCode,
		&role.Description,
		&role.Status,
		&role.DataScope,
		&role.CreateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetByCode 根据角色代码获取角色
func (r *RoleRepository) GetByCode(ctx context.Context, code string) (*sysmodel.SystemRole, error) {
	var role sysmodel.SystemRole
	query := `SELECT id, role_name, role_code, description, status, data_scope, create_time
	FROM workflow_sys_role WHERE role_code = @code`
	err := r.db.QueryRowContext(ctx, query, sql.Named("code", code)).Scan(
		&role.ID,
		&role.RoleName,
		&role.RoleCode,
		&role.Description,
		&role.Status,
		&role.DataScope,
		&role.CreateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// List 分页查询角色
func (r *RoleRepository) List(ctx context.Context, filter sysmodel.RoleFilter, page, pageSize int) ([]*sysmodel.SystemRole, int64, error) {
	where := "1=1"
	args := []interface{}{}

	if filter.RoleName != "" {
		where += " AND role_name LIKE ?"
		args = append(args, "%"+filter.RoleName+"%")
	}
	if filter.Status != nil {
		where += " AND status = ?"
		args = append(args, *filter.Status)
	}

	totalQuery := fmt.Sprintf("SELECT COUNT(1) FROM workflow_sys_role WHERE %s", where)
	var total int64
	err := r.db.QueryRowContext(ctx, totalQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := page * pageSize
	query := fmt.Sprintf(`SELECT id, role_name, role_code, description, status, data_scope, create_time
	FROM workflow_sys_role WHERE %s
	ORDER BY create_time DESC
	OFFSET %d ROWS
	FETCH NEXT %d ROWS ONLY`, where, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*sysmodel.SystemRole
	for rows.Next() {
		role := &sysmodel.SystemRole{}
		err := rows.Scan(
			&role.ID,
			&role.RoleName,
			&role.RoleCode,
			&role.Description,
			&role.Status,
			&role.DataScope,
			&role.CreateTime,
		)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, role)
	}
	return result, total, nil
}

// Create 创建角色并返回ID
func (r *RoleRepository) Create(ctx context.Context, role *sysmodel.SystemRole) (int64, error) {
	query := `INSERT INTO workflow_sys_role (role_name, role_code, description, status, data_scope, create_time, update_time)
	OUTPUT INSERTED.ID
	VALUES (@roleName, @roleCode, @description, @status, @dataScope, @createTime, @updateTime)`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("roleName", role.RoleName),
		sql.Named("roleCode", role.RoleCode),
		sql.Named("description", role.Description),
		sql.Named("status", role.Status),
		sql.Named("dataScope", role.DataScope),
		sql.Named("createTime", role.CreateTime),
		sql.Named("updateTime", role.CreateTime)).Scan(&id) // 创建时间和更新时间一致
	return id, err
}

// Update 更新角色
func (r *RoleRepository) Update(ctx context.Context, role *sysmodel.SystemRole) error {
	query := `UPDATE workflow_sys_role
	SET role_name = @roleName, role_code = @roleCode, description = @description, 
		status = @status, data_scope = @dataScope, update_time = @updateTime
	WHERE id = @id`
	_, err := r.db.ExecContext(ctx, query,
		sql.Named("roleName", role.RoleName),
		sql.Named("roleCode", role.RoleCode),
		sql.Named("description", role.Description),
		sql.Named("status", role.Status),
		sql.Named("dataScope", role.DataScope),
		sql.Named("updateTime", role.CreateTime),
		sql.Named("id", role.ID))
	return err
}

// Delete 删除角色
func (r *RoleRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM workflow_sys_role WHERE id = @id`
	_, err := r.db.ExecContext(ctx, query, sql.Named("id", id))
	return err
}

// GetRolesByUserID 获取用户的角色列表
func (r *RoleRepository) GetRolesByUserID(ctx context.Context, userID int64) ([]*sysmodel.SystemRole, error) {
	query := `SELECT r.id, r.role_name, r.role_code, r.description, r.status, r.data_scope, r.create_time
	FROM workflow_sys_user_role ur
	INNER JOIN workflow_sys_role r ON ur.role_id = r.id
	WHERE ur.user_id = @userID`
	rows, err := r.db.QueryContext(ctx, query, sql.Named("userID", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SystemRole
	for rows.Next() {
		role := &sysmodel.SystemRole{}
		err := rows.Scan(
			&role.ID,
			&role.RoleName,
			&role.RoleCode,
			&role.Description,
			&role.Status,
			&role.DataScope,
			&role.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, role)
	}
	return result, nil
}

// GetMenuIDsByRoleID 获取角色的菜单ID列表
func (r *RoleRepository) GetMenuIDsByRoleID(ctx context.Context, roleID int64) ([]int64, error) {
	query := `SELECT menu_id FROM workflow_sys_role_menu WHERE role_id = @roleID`
	rows, err := r.db.QueryContext(ctx, query, sql.Named("roleID", roleID))
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

// AssignMenus 为角色分配菜单
func (r *RoleRepository) AssignMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 删除旧的菜单分配
	if _, err := tx.ExecContext(ctx, `DELETE FROM workflow_sys_role_menu WHERE role_id = @roleID`, sql.Named("roleID", roleID)); err != nil {
		return err
	}

	// 添加新的菜单分配
	for _, menuID := range menuIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO workflow_sys_role_menu (role_id, menu_id) VALUES (@roleID, @menuID)`,
			sql.Named("roleID", roleID), sql.Named("menuID", menuID)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// AssignUserRoles 为用户分配角色
func (r *RoleRepository) AssignUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 删除旧的关系
	if _, err := tx.ExecContext(ctx, `DELETE FROM workflow_sys_user_role WHERE user_id = @userID`, sql.Named("userID", userID)); err != nil {
		return err
	}

	// 添加新的关系
	for _, roleID := range roleIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO workflow_sys_user_role (user_id, role_id) VALUES (@userID, @roleID)`,
			sql.Named("userID", userID), sql.Named("roleID", roleID)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// MenuRepository 菜单数据访问对象
type MenuRepository struct {
	db *sql.DB
}

// GetByID 根据ID获取菜单
func (r *MenuRepository) GetByID(ctx context.Context, id int64) (*sysmodel.SystemMenu, error) {
	var menu sysmodel.SystemMenu
	query := `SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame
	FROM workflow_sys_menu WHERE id = @id`
	err := r.db.QueryRowContext(ctx, query, sql.Named("id", id)).Scan(
		&menu.ID,
		&menu.ParentID,
		&menu.MenuName,
		&menu.MenuType,
		&menu.Path,
		&menu.Component,
		&menu.Perms,
		&menu.Icon,
		&menu.Sort,
		&menu.Visible,
		&menu.IsFrame,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &menu, nil
}

// List 获取菜单列表
func (r *MenuRepository) List(ctx context.Context, filter sysmodel.MenuFilter) ([]*sysmodel.SystemMenu, error) {
	where := "1=1"
	args := []interface{}{}

	if filter.MenuName != "" {
		where += " AND menu_name LIKE ?"
		args = append(args, "%"+filter.MenuName+"%")
	}
	if filter.Visible != nil {
		where += " AND visible = ?"
		args = append(args, *filter.Visible)
	}

	query := fmt.Sprintf(`SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame
	FROM workflow_sys_menu WHERE %s
	ORDER BY sort ASC`, where)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SystemMenu
	for rows.Next() {
		menu := &sysmodel.SystemMenu{}
		err := rows.Scan(
			&menu.ID,
			&menu.ParentID,
			&menu.MenuName,
			&menu.MenuType,
			&menu.Path,
			&menu.Component,
			&menu.Perms,
			&menu.Icon,
			&menu.Sort,
			&menu.Visible,
			&menu.IsFrame,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, menu)
	}
	return result, nil
}

// GetByUserID 根据用户ID获取菜单
func (r *MenuRepository) GetByUserID(ctx context.Context, userID int64) ([]*sysmodel.SystemMenu, error) {
	query := `SELECT DISTINCT m.id, m.parent_id, m.menu_name, m.menu_type, m.path, m.component, m.perms, m.icon, m.sort, m.visible, m.is_frame
	FROM workflow_sys_menu m
	INNER JOIN workflow_sys_role_menu rm ON m.id = rm.menu_id
	INNER JOIN workflow_sys_user_role ur ON rm.role_id = ur.role_id
	INNER JOIN workflow_sys_role r ON ur.role_id = r.id
	WHERE ur.user_id = @userID AND r.status = 1 AND m.status = 1
	ORDER BY m.sort ASC`
	rows, err := r.db.QueryContext(ctx, query, sql.Named("userID", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.SystemMenu
	for rows.Next() {
		menu := &sysmodel.SystemMenu{}
		err := rows.Scan(
			&menu.ID,
			&menu.ParentID,
			&menu.MenuName,
			&menu.MenuType,
			&menu.Path,
			&menu.Component,
			&menu.Perms,
			&menu.Icon,
			&menu.Sort,
			&menu.Visible,
			&menu.IsFrame,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, menu)
	}
	return result, nil
}

// Create 创建菜单并返回ID
func (r *MenuRepository) Create(ctx context.Context, menu *sysmodel.SystemMenu) (int64, error) {
	query := `INSERT INTO workflow_sys_menu (parent_id, menu_name, menu_type, path, component, perms, icon, sort, visible, is_frame)
	OUTPUT INSERTED.ID
	VALUES (@parentId, @menuName, @menuType, @path, @component, @perms, @icon, @sort, @visible, @isFrame)`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("parentId", menu.ParentID),
		sql.Named("menuName", menu.MenuName),
		sql.Named("menuType", menu.MenuType),
		sql.Named("path", menu.Path),
		sql.Named("component", menu.Component),
		sql.Named("perms", menu.Perms),
		sql.Named("icon", menu.Icon),
		sql.Named("sort", menu.Sort),
		sql.Named("visible", menu.Visible),
		sql.Named("isFrame", menu.IsFrame)).Scan(&id)
	return id, err
}

// Update 更新菜单
func (r *MenuRepository) Update(ctx context.Context, menu *sysmodel.SystemMenu) error {
	query := `UPDATE workflow_sys_menu
	SET parent_id = @parentId, menu_name = @menuName, menu_type = @menuType, path = @path, 
		component = @component, perms = @perms, icon = @icon, sort = @sort, 
		visible = @visible, is_frame = @isFrame
	WHERE id = @id`
	_, err := r.db.ExecContext(ctx, query,
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
		sql.Named("id", menu.ID))
	return err
}

// Delete 删除菜单
func (r *MenuRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM workflow_sys_menu WHERE id = @id`
	_, err := r.db.ExecContext(ctx, query, sql.Named("id", id))
	return err
}

// DictRepository 数据字典数据访问对象
type DictRepository struct {
	db *sql.DB
}

// ListTypes 获取字典类型列表
func (r *DictRepository) ListTypes(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT dict_type FROM workflow_sys_dictionary_info ORDER BY dict_type ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		err := rows.Scan(&t)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, nil
}

// ListByType 根据类型获取字典项
func (r *DictRepository) ListByType(ctx context.Context, dictType string) ([]*sysmodel.DictionaryItem, error) {
	query := `SELECT dic_id, dict_type, dict_name, dict_value, dict_group, sort, status, remark, create_time
	FROM workflow_sys_dictionary_info WHERE dict_type = @dictType ORDER BY sort ASC`
	rows, err := r.db.QueryContext(ctx, query, sql.Named("dictType", dictType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*sysmodel.DictionaryItem
	for rows.Next() {
		dict := &sysmodel.DictionaryItem{}
		err := rows.Scan(
			&dict.DicID,
			&dict.DictType,
			&dict.DictName,
			&dict.DictValue,
			&dict.DictGroup,
			&dict.Sort,
			&dict.Status,
			&dict.Remark,
			&dict.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, dict)
	}
	return result, nil
}

// Page 分页查询字典
func (r *DictRepository) Page(ctx context.Context, filter sysmodel.DictFilter, page, pageSize int) ([]*sysmodel.DictionaryItem, int64, error) {
	where := "1=1"
	args := []interface{}{}

	if filter.DictType != "" {
		where += " AND dict_type = ?"
		args = append(args, filter.DictType)
	}
	if filter.DictGroup != "" {
		where += " AND dict_group = ?"
		args = append(args, filter.DictGroup)
	}
	if filter.Status != nil {
		where += " AND status = ?"
		args = append(args, *filter.Status)
	}

	totalQuery := fmt.Sprintf("SELECT COUNT(1) FROM workflow_sys_dictionary_info WHERE %s", where)
	var total int64
	err := r.db.QueryRowContext(ctx, totalQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := page * pageSize
	query := fmt.Sprintf(`SELECT dic_id, dict_type, dict_name, dict_value, dict_group, sort, status, remark, create_time
	FROM workflow_sys_dictionary_info WHERE %s
	ORDER BY sort ASC
	OFFSET %d ROWS
	FETCH NEXT %d ROWS ONLY`, where, offset, pageSize)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*sysmodel.DictionaryItem
	for rows.Next() {
		dict := &sysmodel.DictionaryItem{}
		err := rows.Scan(
			&dict.DicID,
			&dict.DictType,
			&dict.DictName,
			&dict.DictValue,
			&dict.DictGroup,
			&dict.Sort,
			&dict.Status,
			&dict.Remark,
			&dict.CreateTime,
		)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, dict)
	}
	return result, total, nil
}

// Create 创建字典项并返回ID
func (r *DictRepository) Create(ctx context.Context, item *sysmodel.DictionaryItem) (int64, error) {
	query := `INSERT INTO workflow_sys_dictionary_info (dict_type, dict_name, dict_value, dict_group, sort, status, remark)
	OUTPUT INSERTED.dic_id
	VALUES (@dictType, @dictName, @dictValue, @dictGroup, @sort, @status, @remark)`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("dictType", item.DictType),
		sql.Named("dictName", item.DictName),
		sql.Named("dictValue", item.DictValue),
		sql.Named("dictGroup", item.DictGroup),
		sql.Named("sort", item.Sort),
		sql.Named("status", item.Status),
		sql.Named("remark", item.Remark)).Scan(&id)
	return id, err
}

// Update 更新字典项
func (r *DictRepository) Update(ctx context.Context, item *sysmodel.DictionaryItem) error {
	query := `UPDATE workflow_sys_dictionary_info
	SET dict_type = @dictType, dict_name = @dictName, dict_value = @dictValue, 
		dict_group = @dictGroup, sort = @sort, status = @status, remark = @remark
	WHERE dic_id = @dicId`
	_, err := r.db.ExecContext(ctx, query,
		sql.Named("dictType", item.DictType),
		sql.Named("dictName", item.DictName),
		sql.Named("dictValue", item.DictValue),
		sql.Named("dictGroup", item.DictGroup),
		sql.Named("sort", item.Sort),
		sql.Named("status", item.Status),
		sql.Named("remark", item.Remark),
		sql.Named("dicId", item.DicID))
	return err
}

// Delete 删除字典项
func (r *DictRepository) Delete(ctx context.Context, dicID int64) error {
	query := `DELETE FROM workflow_sys_dictionary_info WHERE dic_id = @dicId`
	_, err := r.db.ExecContext(ctx, query, sql.Named("dicId", dicID))
	return err
}

// Close 关闭数据库连接
func (r *Repository) Close() error {
	return r.db.Close()
}