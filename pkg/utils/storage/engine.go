package storage

import (
	"context"
	"errors"
	"fmt"

	ext "github.com/fize/go-ext/config"
	"github.com/fize/go-ext/log"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const paramError = "<id> and <name> can't be empty at the same time"

// 存储引擎
type Engine struct {
	conn *gorm.DB
}

// 初始化新存储引擎
func NewEngine(address, database, user, passwd string) *Engine {
	var db *gorm.DB
	var err error
	if config.Read().DB.Type == ext.Mysql {
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			user, passwd, address, database)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Error(err)
		}
	} else {
		db, err = gorm.Open(sqlite.Open(database), &gorm.Config{})
		if err != nil {
			log.Error(err)
		}
	}
	return &Engine{
		conn: db,
	}
}

// Client 获取engine的connection
func (s *Engine) Client() interface{} {
	return s.conn
}

// IsExist 根据 key 判断是否存在
func (s *Engine) IsExist(ctx context.Context, id uint, name string, v interface{}) bool {
	var result *gorm.DB
	if id != 0 {

		result = s.conn.WithContext(ctx).Where("id = ?", id).First(v)
	}
	if name != "" {
		result = s.conn.WithContext(ctx).Where("name = ?", name).First(v)
	}
	if id == 0 && name == "" {
		return false
	}
	if result.Error != nil {
		return !errors.Is(result.Error, gorm.ErrRecordNotFound)
	}
	return true
}

// Get 获取记录详情
func (s *Engine) Get(ctx context.Context, id uint, name string, del bool, v interface{}) error {
	if id != 0 {
		if del {
			return s.conn.WithContext(ctx).Unscoped().Where("id = ?", id).First(v).Error
		}
		return s.conn.WithContext(ctx).Where("id = ?", id).First(v).Error
	}
	if name != "" {
		if del {
			return s.conn.WithContext(ctx).Unscoped().Where("name = ?", name).First(v).Error
		}
		return s.conn.WithContext(ctx).Where("name = ?", name).First(v).Error
	}
	return errors.New(paramError)
}

// Create 创建一条新纪录，如果已经存在，则返回错误
func (s *Engine) Create(ctx context.Context, v interface{}) error {
	return s.conn.WithContext(ctx).Create(v).Error
}

// Update 更新一条记录，只会更新有变更的字段
func (s *Engine) Update(ctx context.Context, id uint, name string, has interface{}, v interface{}) error {
	if id != 0 {
		return s.conn.WithContext(ctx).Model(has).Where("id = ?", id).Updates(v).Error
	}
	if name != "" {
		return s.conn.WithContext(ctx).Model(has).Where("name = ?", name).Updates(v).Error
	}
	return errors.New(paramError)
}

// ForceUpdate 强制更新一条记录的所有字段
func (s *Engine) ForceUpdate(ctx context.Context, id uint, name string, has interface{}, v interface{}) error {
	if id != 0 {
		return s.conn.WithContext(ctx).Model(has).Where("id = ?", id).Save(v).Error
	}
	if name != "" {
		return s.conn.WithContext(ctx).Model(has).Where("name = ?", name).Save(v).Error
	}
	return errors.New(paramError)
}

// Delete 将记录的 Delete 时间设置为当前时间
func (s *Engine) Delete(ctx context.Context, id uint, name string, v interface{}) error {
	if id != 0 {
		return s.conn.WithContext(ctx).Where("id = ?", id).Delete(v).Error
	}
	if name != "" {
		return s.conn.WithContext(ctx).Where("name = ?", name).Delete(v).Error
	}
	return errors.New(paramError)
}

// ForceDelete 强制删除一条记录，真正的将记录从数据库中删除
func (s *Engine) ForceDelete(ctx context.Context, id uint, name string, v interface{}) error {
	if id != 0 {
		return s.conn.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(v).Error
	}
	if name != "" {
		return s.conn.WithContext(ctx).Unscoped().Where("name = ?", name).Delete(v).Error
	}
	return errors.New(paramError)
}

// List 根据给定的数据返回一个数据列表
// TODO: condition有sql注入风险
func (s *Engine) List(ctx context.Context, size, current int, condition string, v interface{}) (int64, error) {
	count := s.Count(condition, v)
	countInt := int(count)
	limit, offset := s.handleList(size, current, countInt)
	var result *gorm.DB
	if len(condition) > 0 {
		if limit < 0 { // 当 limit < 0 时，表示不限制 limit
			result = s.conn.WithContext(ctx).Order("id desc").Where(condition).Find(v)
		} else {
			result = s.conn.WithContext(ctx).Order("id desc").Where(condition).Offset(offset).Limit(limit).Find(v)
		}
	} else {
		if limit < 0 {
			result = s.conn.WithContext(ctx).Order("id desc").Find(v)
		} else {
			result = s.conn.WithContext(ctx).Order("id desc").Offset(offset).Limit(limit).Find(v)
		}
	}
	return count, result.Error
}

// 处理记录数量
func (s *Engine) Count(condition string, v interface{}) int64 {
	var count int64
	var err error
	switch {
	case len(condition) > 0:
		err = s.conn.Where(condition).Find(v).Count(&count).Error
	default:
		err = s.conn.Find(v).Count(&count).Error
	}
	if err != nil {
		return 0
	}
	return count
}

// 处理列表数据的分页 limit 和 offset
func (s *Engine) handleList(size, current, count int) (limit, offset int) {
	current = s.compareCurrentPage(size, current, count)
	return size, (current - 1) * size
}

// 处理当前页
func (s *Engine) compareCurrentPage(size, current, count int) int {
	var realPage int
	if count%size > 0 {
		realPage = count/size + 1
	} else {
		realPage = count / size
	}
	if current > realPage {
		current = realPage
	}
	return current
}

// GetAssociation 根据给定的条件关系查询关联数据
func (s *Engine) GetAssociation(ctx context.Context, association string, objA, objB interface{}) error {
	return s.conn.WithContext(ctx).Model(objA).Association(association).Find(objB)
}
