package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// 命令行参数定义
var (
	driverName = flag.String("driver", "sqlite3", "数据库驱动 (sqlite3, mysql, postgres)")
	dsn        = flag.String("dsn", "", "数据库连接字符串 (DSN)")
	tableName  = flag.String("table", "", "指定生成的表名，为空则生成所有表")
	pkgName    = flag.String("pkg", "models", "生成的 Go 代码包名")
	outDir     = flag.String("out", "./models", "代码输出目录")
	overwrite  = flag.Bool("overwrite", false, "如果文件已存在，是否覆盖")
)

// Model 模板定义
const modelTemplate = `package {{.Package}}

import (
	"time"
)

// {{.StructName}} 代表数据库表 {{.RawTableName}} 的模型
type {{.StructName}} struct {
{{- range .Fields}}
	{{.Name}} {{.Type}} ` + "`" + `jorm:"{{.Tag}}"` + "`" + `
{{- end}}
}
`

// Field 代表模型中的一个字段
type Field struct {
	Name string // Go 结构体字段名
	Type string // Go 类型
	Tag  string // jorm 标签
}

// ModelData 代表生成模板所需的数据
type ModelData struct {
	Package      string  // 包名
	StructName   string  // 结构体名
	RawTableName string  // 原始表名
	Fields       []Field // 字段列表
}

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	// 校验必要参数
	if *dsn == "" {
		fmt.Println("使用说明: jorm-gen -dsn <dsn> [其他选项]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// 连接数据库
	db, err := sql.Open(*driverName, *dsn)
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	defer db.Close()

	// 确定需要生成的表
	var tables []string
	if *tableName != "" {
		tables = append(tables, *tableName)
	} else {
		tables, err = fetchAllTables(db, *driverName)
		if err != nil {
			log.Fatalf("获取表列表失败: %v", err)
		}
	}

	// 确保输出目录存在
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	// 循环生成每个表的模型
	for _, table := range tables {
		if err := generateModel(db, table); err != nil {
			log.Printf("生成表 %s 的模型失败: %v", table, err)
		}
	}

	fmt.Println("生成完成！")
}

// generateModel 获取表结构并生成 Go 文件
func generateModel(db *sql.DB, table string) error {
	fields, err := fetchTableInfo(db, *driverName, table)
	if err != nil {
		return err
	}

	structName := snakeToCamel(table, true)
	fileName := filepath.Join(*outDir, strings.ToLower(table)+".go")

	// 检查文件是否存在
	if _, err := os.Stat(fileName); err == nil && !*overwrite {
		log.Printf("文件 %s 已存在，跳过 (使用 -overwrite 覆盖)", fileName)
		return nil
	}

	data := ModelData{
		Package:      *pkgName,
		StructName:   structName,
		RawTableName: table,
		Fields:       fields,
	}

	// 解析并执行模板
	tmpl, err := template.New("model").Parse(modelTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	log.Printf("已生成模型: %s -> %s", table, fileName)
	return nil
}

// fetchAllTables 获取数据库中所有的表名
func fetchAllTables(db *sql.DB, driver string) ([]string, error) {
	var query string
	switch driver {
	case "sqlite3":
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	case "mysql":
		query = "SHOW TABLES"
	case "postgres":
		query = "SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema'"
	default:
		return nil, fmt.Errorf("不支持的驱动: %s", driver)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

// fetchTableInfo 获取指定表的列信息
func fetchTableInfo(db *sql.DB, driver, table string) ([]Field, error) {
	var fields []Field

	switch driver {
	case "sqlite3":
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				cid       int
				name      string
				dataType  string
				notnull   int
				dfltValue sql.NullString
				pk        int
			)
			if err := rows.Scan(&cid, &name, &dataType, &notnull, &dfltValue, &pk); err != nil {
				return nil, err
			}
			fields = append(fields, Field{
				Name: snakeToCamel(name, true),
				Type: mapType(dataType),
				Tag:  generateTag(name, dataType, pk == 1, notnull == 1),
			})
		}
	case "mysql":
		rows, err := db.Query(fmt.Sprintf("DESCRIBE %s", table))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				field      string
				typ        string
				null       string
				key        string
				defaultVal sql.NullString
				extra      string
			)
			if err := rows.Scan(&field, &typ, &null, &key, &defaultVal, &extra); err != nil {
				return nil, err
			}
			isPK := key == "PRI"
			isNotNull := null == "NO"
			fields = append(fields, Field{
				Name: snakeToCamel(field, true),
				Type: mapType(typ),
				Tag:  generateTag(field, typ, isPK, isNotNull),
			})
		}
	case "postgres":
		rows, err := db.Query(`
			SELECT 
				c.column_name, 
				c.data_type, 
				c.is_nullable,
				CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN 'YES' ELSE 'NO' END as is_pk
			FROM information_schema.columns c
			LEFT JOIN information_schema.key_column_usage kcu 
				ON c.table_name = kcu.table_name 
				AND c.column_name = kcu.column_name 
				AND c.table_schema = kcu.table_schema
			LEFT JOIN information_schema.table_constraints tc 
				ON kcu.constraint_name = tc.constraint_name 
				AND kcu.table_schema = tc.table_schema
			WHERE c.table_name = $1 AND c.table_schema = 'public'
			ORDER BY c.ordinal_position`, table)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var name, dataType, isNullable, isPK string
			if err := rows.Scan(&name, &dataType, &isNullable, &isPK); err != nil {
				return nil, err
			}
			fields = append(fields, Field{
				Name: snakeToCamel(name, true),
				Type: mapType(dataType),
				Tag:  generateTag(name, dataType, isPK == "YES", isNullable == "NO"),
			})
		}
	}
	return fields, nil
}

// mapType 将数据库类型映射为 Go 类型
func mapType(dbType string) string {
	dbType = strings.ToUpper(dbType)
	switch {
	case strings.Contains(dbType, "INT"):
		return "int64"
	case strings.Contains(dbType, "CHAR"), strings.Contains(dbType, "TEXT"):
		return "string"
	case strings.Contains(dbType, "FLOAT"), strings.Contains(dbType, "DOUBLE"), strings.Contains(dbType, "DECIMAL"):
		return "float64"
	case strings.Contains(dbType, "BOOL"):
		return "bool"
	case strings.Contains(dbType, "TIME"), strings.Contains(dbType, "DATE"), strings.Contains(dbType, "TIMESTAMP"):
		return "time.Time"
	default:
		return "any"
	}
}

// generateTag 生成 jorm 标签内容
func generateTag(column, dbType string, isPK, isNotNull bool) string {
	var tags []string
	tags = append(tags, fmt.Sprintf("column:%s", column))
	if isPK {
		tags = append(tags, "pk")
		// 如果是整数类型，通常是自增主键
		if strings.Contains(strings.ToUpper(dbType), "INT") {
			tags = append(tags, "auto")
		}
	}
	if isNotNull {
		tags = append(tags, "notnull")
	}

	// 针对时间字段的特殊处理
	colLower := strings.ToLower(column)
	if colLower == "created_at" {
		tags = append(tags, "auto_time")
	} else if colLower == "updated_at" {
		tags = append(tags, "auto_update")
	}

	return strings.Join(tags, ";")
}

// snakeToCamel 将下划线命名转换为驼峰命名
func snakeToCamel(s string, upperFirst bool) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if i == 0 && !upperFirst {
			continue
		}
		// 处理 ID 的特殊情况
		if parts[i] == "id" {
			parts[i] = "ID"
		} else if len(parts[i]) > 0 {
			runes := []rune(parts[i])
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, "")
}
