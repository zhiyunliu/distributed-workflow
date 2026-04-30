package service

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/dao"
	"github.com/zhiyunliu\distributed-workflow/config-management/backend/internal/sysmodel"
	"golang.org/x/crypto/bcrypt"
)

// InitService 初始化服务
type InitService struct {
	repo *dao.Repository
}

// NewInitService 创建初始化服务实例
func NewInitService(repo *dao.Repository) *InitService {
	return &InitService{repo: repo}
}

// InitializeSystem 初始化系统基础数据
func (s *InitService) InitializeSystem(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	
	log.Println("开始初始化系统基础数据...")

	// 初始化管理员角色
	if err := s.initAdminRole(ctx); err != nil {
		log.Printf("初始化管理员角色失败: %v", err)
		return err
	}

	// 初始化管理员用户
	if err := s.initAdminUser(ctx); err != nil {
		log.Printf("初始化管理员用户失败: %v", err)
		return err
	}

	// 初始化系统菜单
	if err := s.initSystemMenus(ctx); err != nil {
		log.Printf("初始化系统菜单失败: %v", err)
		return err
	}

	// 初始化数据字典
	if err := s.initDictionary(ctx); err != nil {
		log.Printf("初始化数据字典失败: %v", err)
		return err
	}

	log.Println("系统基础数据初始化完成")
	return nil
}

// initAdminRole 初始化管理员角色
func (s *InitService) initAdminRole(ctx context.Context) error {
	// 检查是否已存在管理员角色
	role, err := s.repo.GetRoleByCode(ctx, "super_admin")
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if role != nil {
		log.Println("管理员角色已存在，跳过初始化")
		return nil
	}

	// 创建管理员角色
	adminRole := &sysmodel.SysRole{
		RoleName:    "超级管理员",
		RoleCode:    "super_admin",
		Description: "拥有所有权限",
		Status:      1,
		DataScope:   1, // 全量数据权限
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	err = s.repo.CreateRole(ctx, adminRole)
	if err != nil {
		return err
	}

	log.Println("管理员角色初始化成功")
	return nil
}

// initAdminUser 初始化管理员用户
func (s *InitService) initAdminUser(ctx context.Context) error {
	// 检查是否已存在管理员用户
	user, err := s.repo.GetUserByUsername(ctx, "admin")
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if user != nil {
		log.Println("管理员用户已存在，跳过初始化")
		return nil
	}

	// 加密密码 "admin123"
	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 创建管理员用户
	adminUser := &sysmodel.SysUser{
		Username:   "admin",
		Password:   string(hashedPassword),
		RealName:   "系统管理员",
		Status:     1,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	err = s.repo.CreateUser(ctx, adminUser)
	if err != nil {
		return err
	}

	// 获取刚刚创建的用户ID
	user, err = s.repo.GetUserByUsername(ctx, "admin")
	if err != nil {
		return err
	}

	// 获取管理员角色ID
	role, err := s.repo.GetRoleByCode(ctx, "super_admin")
	if err != nil {
		return err
	}

	// 将管理员用户与管理员角色关联
	err = s.repo.UpdateUserRoleRelations(ctx, user.ID, []int64{role.ID})
	if err != nil {
		return err
	}

	log.Println("管理员用户初始化成功")
	return nil
}

// initSystemMenus 初始化系统菜单
func (s *InitService) initSystemMenus(ctx context.Context) error {
	// 检查是否已存在系统菜单
	menus, err := s.repo.QueryMenus(ctx, map[string]interface{}{})
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if len(menus) > 0 {
		log.Println("系统菜单已存在，跳过初始化")
		return nil
	}

	// 定义系统菜单结构
	menusData := []*sysmodel.SysMenu{
		{
			ParentID:  0,
			MenuName:  "系统管理",
			MenuType:  "M", // 目录
			Path:      "/system",
			Component: "",
			Perms:     "",
			Icon:      "setting",
			Sort:      1,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			ParentID:  1, // 系统管理子菜单
			MenuName:  "用户管理",
			MenuType:  "C", // 菜单
			Path:      "/system/user",
			Component: "/system/user/index",
			Perms:     "system:user:list",
			Icon:      "user",
			Sort:      1,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			ParentID:  1, // 系统管理子菜单
			MenuName:  "角色管理",
			MenuType:  "C", // 菜单
			Path:      "/system/role",
			Component: "/system/role/index",
			Perms:     "system:role:list",
			Icon:      "team",
			Sort:      2,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			ParentID:  1, // 系统管理子菜单
			MenuName:  "菜单管理",
			MenuType:  "C", // 菜单
			Path:      "/system/menu",
			Component: "/system/menu/index",
			Perms:     "system:menu:list",
			Icon:      "menu",
			Sort:      3,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			ParentID:  2, // 用户管理子按钮
			MenuName:  "用户查询",
			MenuType:  "F", // 按钮
			Path:      "",
			Component: "",
			Perms:     "system:user:query",
			Icon:      "",
			Sort:      1,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			ParentID:  2, // 用户管理子按钮
			MenuName:  "用户新增",
			MenuType:  "F", // 按钮
			Path:      "",
			Component: "",
			Perms:     "system:user:add",
			Icon:      "",
			Sort:      2,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			ParentID:  2, // 用户管理子按钮
			MenuName:  "用户修改",
			MenuType:  "F", // 按钮
			Path:      "",
			Component: "",
			Perms:     "system:user:edit",
			Icon:      "",
			Sort:      3,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			ParentID:  2, // 用户管理子按钮
			MenuName:  "用户删除",
			MenuType:  "F", // 按钮
			Path:      "",
			Component: "",
			Perms:     "system:user:remove",
			Icon:      "",
			Sort:      4,
			Visible:   1,
			IsFrame:   0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
	}

	// 插入菜单数据
	for i, menu := range menusData {
		// 设置正确的父ID（菜单ID是自增的，从1开始）
		if menu.ParentID > 0 {
			// 因为我们按顺序插入，所以直接使用数组索引+1来计算预期的ID
			// 但我们需要根据实际插入后的ID重新分配
			menu.ParentID = getMenuParentID(i, menusData)
		}
		
		err = s.repo.CreateMenu(ctx, menu)
		if err != nil {
			log.Printf("创建菜单 %s 失败: %v", menu.MenuName, err)
			continue
		}
	}

	log.Println("系统菜单初始化成功")
	return nil
}

// getMenuParentID 根据菜单索引获取父菜单的实际ID
func getMenuParentID(index int, menus []*sysmodel.SysMenu) int64 {
	// 这个辅助函数是为了确保父子关系正确
	// 由于ID是自动生成的，我们先简单地按顺序处理
	// 在真实场景中，可能需要先插入父菜单，获取其ID后再插入子菜单
	switch index {
	case 1: // 用户管理
		return 1 // 父菜单ID为系统管理
	case 2: // 角色管理
		return 1 // 父菜单ID为系统管理
	case 3: // 菜单管理
		return 1 // 父菜单ID为系统管理
	case 4, 5, 6, 7: // 用户管理的子按钮
		return 2 // 父菜单ID为用户管理
	default:
		return 0 // 根菜单
	}
}

// initDictionary 初始化数据字典
func (s *InitService) initDictionary(ctx context.Context) error {
	// 检查是否已存在数据字典
	types, err := s.repo.GetDictTypes(ctx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if len(types) > 0 {
		log.Println("数据字典已存在，跳过初始化")
		return nil
	}

	// 定义初始数据字典
	dictItems := []*sysmodel.SysDictionaryInfo{
		{
			DictType:   "sys_normal_disable",
			DictName:   "系统开关",
			DictValue:  "0",
			DictGroup:  "*",
			Sort:       1,
			Status:     1,
			Remark:     "系统开关-正常",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			DictName:   "系统开关",
			DictValue:  "1",
			DictGroup:  "*",
			Sort:       2,
			Status:     1,
			Remark:     "系统开关-停用",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			DictType:   "sys_user_sex",
			DictName:   "性别",
			DictValue:  "0",
			DictGroup:  "*",
			Sort:       1,
			Status:     1,
			Remark:     "性别-男",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			DictName:   "性别",
			DictValue:  "1",
			DictGroup:  "*",
			Sort:       2,
			Status:     1,
			Remark:     "性别-女",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			DictType:   "sys_yes_no",
			DictName:   "系统默认",
			DictValue:  "Y",
			DictGroup:  "*",
			Sort:       1,
			Status:     1,
			Remark:     "系统默认-是",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		{
			DictName:   "系统默认",
			DictValue:  "N",
			DictGroup:  "*",
			Sort:       2,
			Status:     1,
			Remark:     "系统默认-否",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
	}

	// 修复每个字典项的类型
	currentType := ""
	for i := range dictItems {
		if dictItems[i].DictType == "" {
			dictItems[i].DictType = currentType
		} else {
			currentType = dictItems[i].DictType
		}
	}

	// 插入数据字典
	for _, item := range dictItems {
		err = s.repo.CreateDict(ctx, item)
		if err != nil {
			log.Printf("创建字典项 %s 失败: %v", item.DictName, err)
			continue
		}
	}

	log.Println("数据字典初始化成功")
	return nil
}