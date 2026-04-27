package builtin

import (
	"context"
	"errors"

	"github.com/zhiyunliu/distributed-workflow/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/sysrepo"
)

// ─── MenuService 内置实现 ─────────────────────────────────────────────────────

type menuService struct {
	menuRepo sysrepo.MenuRepo
}

// NewMenuService 创建内置菜单服务
func NewMenuService(menuRepo sysrepo.MenuRepo) sysmanager.MenuService {
	return &menuService{menuRepo: menuRepo}
}

func (s *menuService) GetMenuByID(ctx context.Context, menuID int64) (*sysmodel.SystemMenu, error) {
	return s.menuRepo.GetByID(ctx, menuID)
}

func (s *menuService) ListMenus(ctx context.Context, filter sysmodel.MenuFilter) ([]*sysmodel.SystemMenu, error) {
	return s.menuRepo.List(ctx, filter)
}

func (s *menuService) GetMenuTree(ctx context.Context, filter sysmodel.MenuFilter) ([]*sysmodel.SystemMenu, error) {
	list, err := s.menuRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return buildTree(list, 0), nil
}

func (s *menuService) GetUserMenuTree(ctx context.Context, userID int64) ([]*sysmodel.SystemMenu, error) {
	list, err := s.menuRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return buildTree(list, 0), nil
}

func (s *menuService) CreateMenu(ctx context.Context, menu *sysmodel.SystemMenu) (int64, error) {
	if menu.MenuName == "" {
		return 0, errors.New("菜单名称不能为空")
	}
	if menu.MenuType == "" {
		return 0, errors.New("菜单类型不能为空")
	}
	if menu.Visible == 0 {
		menu.Visible = 1
	}
	return s.menuRepo.Create(ctx, menu)
}

func (s *menuService) UpdateMenu(ctx context.Context, menu *sysmodel.SystemMenu) error {
	if menu.ID <= 0 {
		return errors.New("菜单ID不能为空")
	}
	return s.menuRepo.Update(ctx, menu)
}

func (s *menuService) DeleteMenu(ctx context.Context, menuID int64) error {
	return s.menuRepo.Delete(ctx, menuID)
}

// buildTree 将平铺菜单列表转换为树形结构
func buildTree(list []*sysmodel.SystemMenu, parentID int64) []*sysmodel.SystemMenu {
	var children []*sysmodel.SystemMenu
	for _, m := range list {
		if m.ParentID == parentID {
			m.Children = buildTree(list, m.ID)
			children = append(children, m)
		}
	}
	return children
}
