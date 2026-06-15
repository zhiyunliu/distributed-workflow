package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
	"golang.org/x/crypto/bcrypt"
)

type initRepository interface {
	UserRepo() sysrepo.UserRepo
	RoleRepo() sysrepo.RoleRepo
	MenuRepo() sysrepo.MenuRepo
	DictRepo() sysrepo.DictRepo
}

// InitService 初始化服务
type InitService struct {
	repo initRepository
}

// NewInitService 创建初始化服务实例
func NewInitService(repo initRepository) *InitService {
	return &InitService{
		repo: repo,
	}
}

// InitializeSystem 初始化系统基础数据
func (s *InitService) InitializeSystem(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// 检查是否已有超级管理员角色
	roleRepo := s.repo.RoleRepo()
	role, err := roleRepo.GetByCode(ctx, "super_admin")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if role == nil {
		log.Println("创建超级管理员角色")
		adminRole := &sysmodel.SystemRole{
			RoleName:    "超级管理员",
			RoleCode:    "super_admin",
			Description: "系统超级管理员角色",
			Status:      1,
			DataScope:   0,
			CreateTime:  time.Now(),
		}
		roleID, err := roleRepo.Create(ctx, adminRole)
		if err != nil {
			return err
		}
		adminRole.ID = roleID
		log.Printf("超级管理员角色创建成功，ID: %d", roleID)
	} else {
		log.Printf("超级管理员角色已存在，ID: %d", role.ID)
	}

	// 检查是否已有超级管理员用户
	userRepo := s.repo.UserRepo()
	user, err := userRepo.GetByUsername(ctx, "admin")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if user == nil {
		initAdminPassword := os.Getenv("INIT_ADMIN_PASSWORD")
		if initAdminPassword == "" {
			return errors.New("missing INIT_ADMIN_PASSWORD, refuse to auto create admin")
		}
		if err := ValidatePassword(initAdminPassword, DefaultPasswordPolicy); err != nil {
			return fmt.Errorf("invalid INIT_ADMIN_PASSWORD: %w", err)
		}

		log.Println("创建超级管理员用户")
		hash, err := bcrypt.GenerateFromPassword([]byte(initAdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		adminUser := &sysmodel.SystemUser{
			Username:   "admin",
			Password:   string(hash),
			RealName:   "超级管理员",
			Status:     1,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		userID, err := userRepo.Create(ctx, adminUser)
		if err != nil {
			return err
		}
		adminUser.ID = userID
		log.Printf("超级管理员用户创建成功，ID: %d", userID)
	} else {
		log.Printf("超级管理员用户已存在，ID: %d", user.ID)
	}

	// 重新获取用户和角色信息
	user, err = userRepo.GetByUsername(ctx, "admin")
	if err != nil {
		return err
	}
	role, err = roleRepo.GetByCode(ctx, "super_admin")
	if err != nil {
		return err
	}

	// 绑定用户和角色关系
	// 先检查是否已有绑定关系
	assignedRoles, err := roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return err
	}
	hasRole := false
	for _, assignedRole := range assignedRoles {
		if assignedRole.ID == role.ID {
			hasRole = true
			break
		}
	}
	if !hasRole {
		log.Println("绑定用户和角色关系")
		err = roleRepo.AssignUserRoles(ctx, user.ID, []int64{role.ID})
		if err != nil {
			return err
		}
		log.Println("用户角色绑定成功")
	} else {
		log.Println("用户角色关系已存在")
	}

	// 检查并创建菜单
	menuRepo := s.repo.MenuRepo()
	allMenus, err := menuRepo.List(ctx, sysmodel.MenuFilter{})
	if err != nil {
		return err
	}

	if len(allMenus) == 0 {
		log.Println("创建默认菜单")
		defaultMenus := s.getDefaultMenus()
		for _, menu := range defaultMenus {
			_, err := menuRepo.Create(ctx, menu)
			if err != nil {
				log.Printf("创建菜单失败: %v", err)
				continue
			}
		}
		log.Println("默认菜单创建完成")
	} else {
		log.Printf("系统已有%d个菜单，跳过初始化", len(allMenus))
	}

	// 检查并创建数据字典
	dictRepo := s.repo.DictRepo()
	dictTypes, err := dictRepo.ListTypes(ctx)
	if err != nil {
		return err
	}

	if len(dictTypes) == 0 {
		log.Println("创建默认数据字典")
		defaultDictionaries := s.getDefaultDictionaries()
		for _, dictType := range defaultDictionaries {
			for _, item := range dictType {
				_, err := dictRepo.Create(ctx, item)
				if err != nil {
					log.Printf("创建数据字典失败: %v", err)
					continue
				}
			}
		}
		log.Println("默认数据字典创建完成")
	} else {
		log.Printf("系统已有%d种数据字典类型，跳过初始化", len(dictTypes))
	}

	return nil
}

func (s *InitService) getDefaultMenus() []*sysmodel.SystemMenu {
	return []*sysmodel.SystemMenu{
		{
			ParentID:  0,
			MenuName:  "系统管理",
			MenuType:  "M",
			Path:      "/system",
			Component: "",
			Perms:     "system",
			Icon:      "setting",
			Sort:      1,
			Visible:   1,
			IsFrame:   0,
		},
		{
			ParentID:  1,
			MenuName:  "用户管理",
			MenuType:  "C",
			Path:      "/system/user",
			Component: "system/user/index",
			Perms:     "system:user:list",
			Icon:      "user",
			Sort:      1,
			Visible:   1,
			IsFrame:   0,
		},
		{
			ParentID:  1,
			MenuName:  "角色管理",
			MenuType:  "C",
			Path:      "/system/role",
			Component: "system/role/index",
			Perms:     "system:role:list",
			Icon:      "team",
			Sort:      2,
			Visible:   1,
			IsFrame:   0,
		},
		{
			ParentID:  1,
			MenuName:  "菜单管理",
			MenuType:  "C",
			Path:      "/system/menu",
			Component: "system/menu/index",
			Perms:     "system:menu:list",
			Icon:      "unordered-list",
			Sort:      3,
			Visible:   1,
			IsFrame:   0,
		},
		{
			ParentID:  1,
			MenuName:  "字典管理",
			MenuType:  "C",
			Path:      "/system/dict",
			Component: "system/dict/index",
			Perms:     "system:dict:list",
			Icon:      "book",
			Sort:      4,
			Visible:   1,
			IsFrame:   0,
		},
	}
}

func (s *InitService) getDefaultDictionaries() map[string][]*sysmodel.DictionaryItem {
	return map[string][]*sysmodel.DictionaryItem{
		"user_status": {
			{
				DictType:   "user_status",
				DictName:   "启用",
				DictValue:  "1",
				DictGroup:  "user",
				Sort:       1,
				Status:     1,
				Remark:     "用户启用状态",
				CreateTime: time.Now(),
			},
			{
				DictType:   "user_status",
				DictName:   "禁用",
				DictValue:  "0",
				DictGroup:  "user",
				Sort:       2,
				Status:     1,
				Remark:     "用户禁用状态",
				CreateTime: time.Now(),
			},
		},
		"menu_type": {
			{
				DictType:   "menu_type",
				DictName:   "菜单",
				DictValue:  "C",
				DictGroup:  "menu",
				Sort:       1,
				Status:     1,
				Remark:     "菜单类型",
				CreateTime: time.Now(),
			},
			{
				DictType:   "menu_type",
				DictName:   "按钮",
				DictValue:  "F",
				DictGroup:  "menu",
				Sort:       2,
				Status:     1,
				Remark:     "按钮类型",
				CreateTime: time.Now(),
			},
		},
	}
}
