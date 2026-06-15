/*
 * @Date: 2026-06-15 17:53:11
 * @LastEditTime: 2026-06-15 17:54:38
 * @FilePath: /dark_pkg/pkg/gorm_cli/model_template.go
 * @Description:
 */
package gorm_cli

const ModelBaseTemplate = `package {{.Package}}

import (
	"github.com/ye-f-ying/dark_pkg/pkg/db"

	"gorm.io/gorm"
)

func GetModel() *gorm.DB {
	return db.GetGormDBMYSQL()
}
`

const ModelTemplate = `package {{.Package}}

import (
{{- if .NeedTimeImport}}
	"time"
{{- end}}
{{- if .NeedDecimalImport}}
	"{{.DecimalPackage}}"
{{- end}}
{{- if .NeedGormImport}}
	"gorm.io/gorm"
{{- end}}
)

// {{.StructName}} {{.TableDesc}}
type {{.StructName}} struct {
{{- range .Fields}}
	{{.StructFieldName}} {{.GoType}} {{$.Backtick}}gorm:"{{.GormTag}}" json:"{{.JsonTag}}"{{$.Backtick}} // {{.Comment}}
{{- end}}
}

func ({{.StructName}}) TableName() string {
	return "{{.TableName}}"
}

// GetById 根据主键ID查询
func (m *{{.StructName}}) GetById(id {{.PrimaryKeyType}}) (*{{.StructName}}, error) {
	var obj {{.StructName}}
	err := GetModel().First(&obj, id).Error
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

// Create 新增记录
func (m *{{.StructName}}) Create() error {
	return GetModel().Model(m).Create(m).Error
}

// Save 全量保存（主键存在则更新所有字段，不存在则新增）
func (m *{{.StructName}}) Save() error {
	return GetModel().Save(m).Error
}

// UpdateById 根据主键ID更新（仅更新非零值字段）
func (m *{{.StructName}}) UpdateById(id {{.PrimaryKeyType}}) error {
	return GetModel().Model(m).Where("{{.PrimaryKeyColumn}} = ?", id).Omit("{{.PrimaryKeyColumn}}").Updates(m).Error
}

// DeleteById 根据主键ID删除
{{- if .HasSoftDelete}}
// 软删除：自动更新 delete_time 字段
{{- else}}
// 硬删除：物理删除记录
{{- end}}
func (m *{{.StructName}}) DeleteById(id {{.PrimaryKeyType}}) error {
	return GetModel().Where("{{.PrimaryKeyColumn}} = ?", id).Delete(m).Error
}

// List 条件分页查询列表
func (m *{{.StructName}}) List(where map[string]interface{}, page, pageSize int) ([]{{.StructName}}, int64, error) {
	var list []{{.StructName}}
	var obj {{.StructName}}
	var total int64

	db := GetModel().Model(&obj)
	if len(where) > 0 {
		db = db.Where(where)
	}

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		db = db.Offset(offset).Limit(pageSize)
	}

	err = db.Find(&list).Error
	return list, total, err
}

{{- if .HasSoftDelete}}
// ListWithDeleted 条件分页查询列表（包含已软删除记录）
func (m *{{.StructName}}) ListWithDeleted(where map[string]interface{}, page, pageSize int) ([]{{.StructName}}, int64, error) {
	var list []{{.StructName}}
	var obj {{.StructName}}
	var total int64

	db := GetModel().Model(&obj).Unscoped()
	if len(where) > 0 {
		db = db.Where(where)
	}

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		db = db.Offset(offset).Limit(pageSize)
	}

	err = db.Find(&list).Error
	return list, total, err
}
{{- end}}
`
