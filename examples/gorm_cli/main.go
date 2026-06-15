/*
 * @Date: 2026-06-15 15:23:17
 * @LastEditTime: 2026-06-15 17:59:38
 * @FilePath: /dark_pkg/examples/gorm_cli/main.go
 * @Description:
 */
package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/ye-f-ying/dark_pkg/pkg/config"
	"github.com/ye-f-ying/dark_pkg/pkg/db"
	"github.com/ye-f-ying/dark_pkg/pkg/gorm_cli"
)

type Config struct {
	config.DefaultConfig `mapstructure:",squash"` // 需要扁平化
}

func main() {
	cfg, err := config.Init[*Config]()
	if err != nil {
		hlog.Errorf("init config error:%v ", err)
		os.Exit(1)
		return
	}
	conf, err := cfg.GetConfig()
	if err != nil {
		hlog.Errorf("get config error:%v ", err)
		os.Exit(1)
		return
	}

	err = db.InitGormMYSQL()
	if err != nil {
		hlog.Errorf("init grom db error:%v ", err)
		os.Exit(1)
		return
	}
	masterDB, err := db.GetGormDBMYSQL().DB()
	if err != nil {
		hlog.Errorf("init grom db error:%v ", err)
		os.Exit(1)
		return
	}
	dbName := conf.GetMysql().DBName
	tableFlag := flag.String("table", "", "指定生成的表名，不指定则生成当前库所有表")
	outFlag := flag.String("out", "./models", "模型文件输出目录，默认 ./models")
	pkgName := flag.String("pkg", "models", "包名，默认 models")
	flag.Parse()

	// 2. 创建输出目录（不存在则递归创建）
	outputDir := *outFlag
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		hlog.Errorf("❌ 创建输出目录失败: %v", err)
		os.Exit(1)
		return
	}

	if err := gorm_cli.GenerateBase(outputDir, *pkgName); err != nil {
		hlog.Errorf("❌ 创建基础模型失败: %v", err)
		os.Exit(1)
		return
	}

	var tables []string
	if *tableFlag != "" {
		tables = append(tables, *tableFlag)
		hlog.Infof("📌 指定生成单表: %s", *tableFlag)
	} else {
		tables, err = gorm_cli.GetAllTables(masterDB, dbName)
		if err != nil {
			hlog.Infof("❌ 获取库中所有表失败: %v", err)
			return
		}
		hlog.Infof("📌 检测到库中共有 %d 张表，开始批量生成\n", len(tables))
	}

	successCount := 0

	for _, tbl := range tables {
		meta, err := gorm_cli.GetTableMeta(masterDB, dbName, tbl)
		if err != nil {
			hlog.Errorf("❌ 读取表 [%s] 结构失败: %v\n", tbl, err)
			continue
		}

		// 文件名 = 表名.go
		outPath := filepath.Join(outputDir, tbl+".go")
		err = gorm_cli.GenerateModel(meta, gorm_cli.ModelTemplate, outPath, *pkgName)
		if err != nil {
			hlog.Errorf("❌ 生成表 [%s] 模型失败: %v\n", tbl, err)
			continue
		}

		hlog.Errorf("✅ 生成成功: %s\n", outPath)
		successCount++
	}

}
