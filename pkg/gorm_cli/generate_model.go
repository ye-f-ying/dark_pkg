/*
 * @Date: 2026-06-15 17:52:55
 * @LastEditTime: 2026-06-15 17:55:40
 * @FilePath: /dark_pkg/pkg/gorm_cli/generate_model.go
 * @Description:
 */
package gorm_cli

import (
	"bytes"
	"database/sql"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/ye-f-ying/dark_pkg/pkg/utils"
)

const (
	decimalPackage = "github.com/shopspring/decimal"
)

// 行
type Column struct {
	Field      string // 字段
	Type       string // 类型
	Null       string // 是否为空
	Key        string // 键
	DefaultVal string // 默认值
	Comment    string // 说明
	Extra      string // 扩展

}

// 索引
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// 表
type TableMeta struct {
	TableName string
	TableDesc string
	Columns   []Column
	Indexes   []Index
}

// 模板数据
type TplData struct {
	Package           string
	TableName         string
	StructName        string
	TableDesc         string
	Fields            []FieldInfo
	Backtick          string // 反引号占位，解决模板内反引号语法问题
	NeedDecimalImport bool   // 是否需要导入decimal包
	DecimalPackage    string
	NeedGormImport    bool   // 是否需要导入 gorm 包
	HasSoftDelete     bool   // 是否包含软删除字段
	PrimaryKeyColumn  string // 主键字段名
	PrimaryKeyType    string // 主键Go类型
	NeedTimeImport    bool   // 是否需要导入时间
}

// 字段信息
type FieldInfo struct {
	StructFieldName string // 结构体字段名称
	GoType          string // go 类型
	GormTag         string // grom 标签
	JsonTag         string // json 标签
	Comment         string // 评论
}

func mysql2GoType(mysqlType, null string) string {
	lowType := strings.ToLower(mysqlType)
	switch {
	case strings.Contains(lowType, "decimal") || strings.Contains(lowType, "numeric"):
		if null == "YES" {
			return "*decimal.Decimal"
		}
		return "decimal.Decimal"
	case strings.Contains(lowType, "bigint"):
		if null == "YES" {
			return "*uint64"
		}
		return "uint64"
	case strings.Contains(lowType, "tinyint"):
		if null == "YES" {
			return "*int8"
		}
		return "int8"
	case strings.Contains(lowType, "int"):
		if null == "YES" {
			return "*int32"
		}
		return "int32"
	case strings.Contains(lowType, "varchar"), strings.Contains(lowType, "char"), strings.Contains(lowType, "text"):
		if null == "YES" {
			return "*string"
		}
		return "string"
	case strings.Contains(lowType, "datetime"), strings.Contains(lowType, "timestamp"):
		if null == "YES" {
			return "*time.Time"
		}
		return "time.Time"
	case strings.Contains(lowType, "double"), strings.Contains(lowType, "float"):
		if null == "YES" {
			return "*float64"
		}
		return "float64"
	default:
		return "string"
	}
}

// ===================== MySQL 元数据读取 =====================
/**
 * @description: 获取库中所有表名
 * @param {*sql.DB} db
 * @param {string} dbName
 * @return {*}
 */
func GetAllTables(db *sql.DB, dbName string) ([]string, error) {
	rows, err := db.Query(
		`SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?`,
		dbName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}
func parseIndexes(db *sql.DB, dbName, table string) ([]Index, error) {
	rows, err := db.Query(`
		SELECT INDEX_NAME, COLUMN_NAME, NON_UNIQUE 
		FROM INFORMATION_SCHEMA.STATISTICS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`, dbName, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*Index)
	for rows.Next() {
		var (
			indexName string
			colName   string
			nonUnique int
		)
		if err := rows.Scan(&indexName, &colName, &nonUnique); err != nil {
			return nil, err
		}

		unique := nonUnique == 0
		if idx, ok := indexMap[indexName]; ok {
			idx.Columns = append(idx.Columns, colName)
		} else {
			indexMap[indexName] = &Index{
				Name:    indexName,
				Columns: []string{colName},
				Unique:  unique,
			}
		}
	}

	var indexes []Index
	for _, v := range indexMap {
		// 跳过主键（主键已在字段上单独标记 primaryKey）
		if v.Name == "PRIMARY" {
			continue
		}
		indexes = append(indexes, *v)
	}
	return indexes, nil
}

func GetTableMeta(db *sql.DB, dbName, table string) (*TableMeta, error) {
	var tableComment string
	err := db.QueryRow(
		`SELECT TABLE_COMMENT FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`,
		dbName, table,
	).Scan(&tableComment)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`", table)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		var collation, priv sql.NullString // 可能为 NULL 的字段用 NullString
		var defaultVal sql.NullString      // 默认值也可能为 NULL
		err := rows.Scan(
			&col.Field,
			&col.Type,
			&collation, // Collation 可能为 NULL（非字符串类型）
			&col.Null,
			&col.Key,
			&defaultVal, // Default 可能为 NULL
			&col.Extra,
			&priv, // Privileges
			&col.Comment,
		)
		if err != nil {
			return nil, err
		}

		// NULL 默认值转空字符串
		if defaultVal.Valid {
			col.DefaultVal = defaultVal.String
		} else {
			col.DefaultVal = ""
		}

		columns = append(columns, col)
	}

	indexes, err := parseIndexes(db, dbName, table)
	if err != nil {
		return nil, err
	}

	return &TableMeta{
		TableName: table,
		TableDesc: tableComment,
		Columns:   columns,
		Indexes:   indexes,
	}, nil
}

// ===================== 构建模板字段数据 =====================
func buildFieldList(meta *TableMeta) ([]FieldInfo, bool, string, string, bool) {
	var fields []FieldInfo
	var hasSoftDelete bool
	var primaryKeyColumn string
	var primaryKeyType string
	var needTimeImport bool

	indexMap := make(map[string][]Index)
	for _, idx := range meta.Indexes {
		for _, col := range idx.Columns {
			indexMap[col] = append(indexMap[col], idx)
		}
	}

	//dtReg := regexp.MustCompile(`(?i)current_timestamp`)
	dtReg := regexp.MustCompile(`(?i)(current_timestamp|now)\s*(\(\d*\))?`)

	for _, col := range meta.Columns {
		var fi FieldInfo
		fi.StructFieldName = utils.ToCamelCase(col.Field)
		fi.Comment = strings.TrimSpace(col.Comment)

		isDeleteAt := strings.ToLower(col.Field) == "delete_time"
		if isDeleteAt {
			fi.GoType = "gorm.DeletedAt"
			hasSoftDelete = true
		} else {
			fi.GoType = mysql2GoType(col.Type, col.Null)
			if strings.Contains(fi.GoType, "time.Time") {
				needTimeImport = true
			}
		}

		// 记录主键信息
		if col.Key == "PRI" {
			primaryKeyColumn = col.Field
			primaryKeyType = mysql2GoType(col.Type, col.Null)
			// 主键去掉指针类型
			primaryKeyType = strings.TrimPrefix(primaryKeyType, "*")
		}

		var gormParts []string
		gormParts = append(gormParts, fmt.Sprintf("type:%s", col.Type))

		if col.Null == "NO" {
			if !(strings.Contains(col.Type, "datetime") || strings.Contains(col.Type, "timestamp")) {
				gormParts = append(gormParts, "not null")
			}
		}

		def := strings.TrimSpace(col.DefaultVal)
		if def != "" {
			if dtReg.MatchString(def) {
				if strings.Contains(strings.ToLower(col.Field), "create_time") {
					gormParts = append(gormParts, "autoCreateTime")
				} else if strings.Contains(strings.ToLower(col.Field), "update_time") {
					gormParts = append(gormParts, "autoUpdateTime")
				} else {
					gormParts = append(gormParts, "autoCreateTime")
				}
			} else {
				gormParts = append(gormParts, fmt.Sprintf("default:'%s'", def))
			}
		}

		if col.Key == "PRI" {
			gormParts = append(gormParts, "primaryKey")
			if strings.Contains(strings.ToLower(col.Extra), "auto_increment") {
				gormParts = append(gormParts, "autoIncrement")
			}
		}

		if idxs, ok := indexMap[col.Field]; ok {
			for _, idx := range idxs {
				if idx.Unique {
					gormParts = append(gormParts, fmt.Sprintf("uniqueIndex:%s", idx.Name))
				} else {
					gormParts = append(gormParts, fmt.Sprintf("index:%s", idx.Name))
				}
			}
		}

		gormParts = append(gormParts, fmt.Sprintf("column:%s", col.Field))
		fi.GormTag = strings.Join(gormParts, ";")

		if isDeleteAt {
			fi.JsonTag = "-"
		} else {
			jsonName := utils.ToLowerCamel(col.Field)
			if strings.Contains(strings.ToLower(col.Type), "bigint") {
				fi.JsonTag = fmt.Sprintf("%s,string", jsonName)
			} else {
				fi.JsonTag = jsonName
			}
		}

		fields = append(fields, fi)
	}

	// 兜底：没检测到主键时默认 id uint64
	if primaryKeyColumn == "" {
		primaryKeyColumn = "id"
		primaryKeyType = "uint64"
	}

	return fields, hasSoftDelete, primaryKeyColumn, primaryKeyType, needTimeImport
}

func checkNeedDecimal(fields []FieldInfo) bool {
	for _, f := range fields {
		if strings.Contains(f.GoType, "decimal") {
			return true
		}
	}
	return false
}

// 检查是否需要导入 gorm 包（仅用到 gorm.DeletedAt 时才需要）
func checkNeedGorm(fields []FieldInfo) bool {
	for _, f := range fields {
		if strings.Contains(f.GoType, "gorm.DeletedAt") {
			return true
		}
	}
	return false
}

// ===================== 模板渲染 =====================
// RenderTemplate 渲染模板并自动执行 go fmt 格式化后写入文件
func RenderTemplate(tplStr string, data TplData, outPath string) error {
	tpl, err := template.New("gorm_model").Parse(tplStr)
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	// 1. 先渲染到内存缓冲区
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	// 2. 执行 go fmt 标准格式化
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// 格式化失败兜底：保留原始文件方便排查问题
		_ = os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("代码格式化失败（已保留原始文件）: %w", err)
	}

	// 3. 写入格式化后的最终文件
	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// GenerateBase 生成公共 base.go 文件，已存在则跳过
func GenerateBase(outDir, packageName string) error {
	outPath := filepath.Join(outDir, "model.go")
	if _, err := os.Stat(outPath); err == nil {
		return nil
	}

	data := struct {
		Package string
	}{
		Package: packageName,
	}

	tpl, err := template.New("base").Parse(ModelBaseTemplate)
	if err != nil {
		return fmt.Errorf("解析 base 模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("渲染 base 模板失败: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		_ = os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("base 代码格式化失败: %w", err)
	}

	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("写入 base.go 失败: %w", err)
	}
	return nil
}

func GenerateModel(meta *TableMeta, tplStr string, outPath, packageName string) error {
	fieldList, hasSoftDelete, pkColumn, pkType, needTime := buildFieldList(meta)
	needDecimal := checkNeedDecimal(fieldList)
	needGorm := checkNeedGorm(fieldList)
	tplData := TplData{
		Package:           packageName,
		TableName:         meta.TableName,
		StructName:        utils.ToCamelCase(meta.TableName),
		TableDesc:         meta.TableDesc,
		Fields:            fieldList,
		Backtick:          "`",
		NeedDecimalImport: needDecimal,
		NeedGormImport:    needGorm,
		DecimalPackage:    decimalPackage,
		HasSoftDelete:     hasSoftDelete,
		PrimaryKeyColumn:  pkColumn,
		PrimaryKeyType:    pkType,
		NeedTimeImport:    needTime,
	}

	return RenderTemplate(tplStr, tplData, outPath)
}
