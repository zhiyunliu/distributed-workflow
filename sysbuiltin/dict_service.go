package builtin

import (
	"context"
	"errors"

	"github.com/zhiyunliu/distributed-workflow/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/sysrepo"
)

// ─── DictionaryService 内置实现 ───────────────────────────────────────────────

type dictService struct {
	dictRepo sysrepo.DictRepo
}

// NewDictionaryService 创建内置字典服务
func NewDictionaryService(dictRepo sysrepo.DictRepo) sysmanager.DictionaryService {
	return &dictService{dictRepo: dictRepo}
}

func (s *dictService) ListDictTypes(ctx context.Context) ([]string, error) {
	return s.dictRepo.ListTypes(ctx)
}

func (s *dictService) ListDictData(ctx context.Context, dictType string) ([]*sysmodel.DictionaryItem, error) {
	if dictType == "" {
		return nil, errors.New("字典类型不能为空")
	}
	return s.dictRepo.ListByType(ctx, dictType)
}

func (s *dictService) PageDictData(ctx context.Context, filter sysmodel.DictFilter, page, pageSize int) ([]*sysmodel.DictionaryItem, int64, error) {
	return s.dictRepo.Page(ctx, filter, page, pageSize)
}

func (s *dictService) CreateDictData(ctx context.Context, item *sysmodel.DictionaryItem) (int64, error) {
	if item.DictType == "" || item.DictName == "" || item.DictValue == "" {
		return 0, errors.New("字典类型、名称和值不能为空")
	}
	if item.DictGroup == "" {
		item.DictGroup = "*"
	}
	if item.Status == 0 {
		item.Status = 1
	}
	return s.dictRepo.Create(ctx, item)
}

func (s *dictService) UpdateDictData(ctx context.Context, item *sysmodel.DictionaryItem) error {
	if item.DicID <= 0 {
		return errors.New("字典ID不能为空")
	}
	return s.dictRepo.Update(ctx, item)
}

func (s *dictService) DeleteDictData(ctx context.Context, dicID int64) error {
	return s.dictRepo.Delete(ctx, dicID)
}
