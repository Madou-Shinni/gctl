package main

import (
	"bufio"
	"fmt"
	"github.com/Madou-Shinni/gin-quickstart/pkg/tools/str"
	"github.com/urfave/cli/v2"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

var (
	defaultInitRoutersDir = "initialize/router.go"
	defaultInitDataDir    = "initialize/data.go"
	version               = "1.2.0"
)

type Temp struct {
	Module             string // 模块名
	ModuleLower        string
	ModuleCamelToSnake string
}

func main() {
	app := &cli.App{
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "module",
				Aliases:  []string{"m"},
				Usage:    "生成模块的名称",
				Required: true,
			},
		},
		Action: gen,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// 生成代码
func gen(c *cli.Context) error {
	var err error
	var wg sync.WaitGroup
	var tempSlice []string

	s := c.String("module")

	// 定义变量
	//k v
	data := Temp{
		Module:             s,
		ModuleLower:        strings.ToLower(s[:1]) + s[1:],
		ModuleCamelToSnake: str.CamelToSnake(s),
	}

	templateDir := "cmd/template"

	// 遍历模板文件
	dir, err := os.ReadDir(templateDir)
	if err != nil {
		log.Fatalf("read dir failed: %s", err.Error())
		return err
	}

	for _, entry := range dir {
		if strings.Contains(entry.Name(), "txt") {
			// 将模板文件名加入列表
			tempSlice = append(tempSlice, entry.Name())
		}
	}

	err = checkFile(s)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	wg.Add(len(tempSlice))
	for i := 0; i < len(tempSlice); i++ {
		// 启动5个goroutine生成不同的模板文件
		go func(i int) {
			defer wg.Done()

			var t *template.Template
			var f *os.File

			// 解析模板文件
			t, err = template.ParseFiles(templateDir + "/" + tempSlice[i])
			if err != nil {
				return
			}

			// 写出文件
			err = writeOutput(s, tempSlice[i], data, f, t)
			if err != nil {
				return
			}

		}(i)
	}

	wg.Wait()

	if err != nil {
		return err
	}

	err = insertRouterRegister(defaultInitRoutersDir, s)
	if err != nil {
		return err
	}

	err = insertDataAutoMigrate(defaultInitDataDir, s)

	log.Println("gen code success")

	return nil
}

// 检查文件在目录下是否存在
func checkFile(s string) error {
	return filepath.Walk("internal", func(path string, d fs.FileInfo, err error) error {
		if d.IsDir() {
			return nil
		}

		if d.Name() == fmt.Sprint(str.CamelToSnake(s), ".go") {
			return fmt.Errorf("\033[31m file %s exists \033[0m", fmt.Sprint(path))
		}
		return nil
	})
}

// 写出文件
func writeOutput(module string, sliceItem string, data Temp, f *os.File, t *template.Template) error {
	dirname := strings.Split(sliceItem, "_")[0]
	outputDir := "."

	switch dirname {
	case "data":
		outputDir = "internal/data"
	case "domain":
		outputDir = "internal/domain"
	case "service":
		outputDir = "internal/service"
	case "handle":
		outputDir = "api/handle"
	case "route":
		outputDir = "api/routers"
	default:
		outputDir = "./gen_code"
	}

	// 创建文件夹If path is already a directory, MkdirAll does nothing and returns nil.
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	// 将驼峰式字符串转换为下划线式字符串
	module = str.CamelToSnake(module)

	// 渲染模板并将结果写入文件
	f, err = os.Create(fmt.Sprintf("%s/%s.go", outputDir, module))
	if err != nil {
		return err
	}
	defer f.Close()

	err = t.Execute(f, data)
	if err != nil {
		return err
	}

	return nil
}

// insertRouterRegister 在注册路由的位置插入新的路由注册代码
func insertRouterRegister(targetFile, moduleName string) error {
	// 读取目标文件内容
	fileContent, err := os.ReadFile(targetFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(fileContent)))
	var output strings.Builder

	foundRegistrationSection := false
	lastRouterRegisterLine := -1
	lines := []string{}

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		if strings.Contains(line, "// 注册路由") {
			foundRegistrationSection = true
		}

		if foundRegistrationSection && strings.Contains(line, "routers.") && strings.Contains(line, "RouterRegister") {
			lastRouterRegisterLine = len(lines) - 1
		}
	}

	if lastRouterRegisterLine != -1 {
		for i, line := range lines {
			output.WriteString(line + "\n")
			if i == lastRouterRegisterLine {
				output.WriteString(fmt.Sprintf("\trouters.%sRouterRegister(public)\n", moduleName))
			}
		}
	} else {
		// 如果没有找到注册路由部分，则在 // 注册路由 下添加
		for _, line := range lines {
			output.WriteString(line + "\n")
			if strings.Contains(line, "// 注册路由") {
				output.WriteString(fmt.Sprintf("\trouters.%sRouterRegister(public)\n", moduleName))
			}
		}
	}

	// 写回文件
	err = os.WriteFile(targetFile, []byte(output.String()), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// insertDataAutoMigrate 在自动迁移的位置插入新的自动迁移代码
func insertDataAutoMigrate(targetFile, moduleName string) error {
	// 读取目标文件内容
	fileContent, err := os.ReadFile(targetFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(fileContent)))
	var output strings.Builder

	foundRegistrationSection := false
	lastAuthMigrateLine := -1
	lines := []string{}

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		if strings.Contains(line, "// 自动迁移") {
			foundRegistrationSection = true
		}

		if foundRegistrationSection && strings.Contains(line, "domain.") && strings.Contains(line, "{}") {
			lastAuthMigrateLine = len(lines) - 1
		}
	}

	if lastAuthMigrateLine != -1 {
		for i, line := range lines {
			output.WriteString(line + "\n")
			if i == lastAuthMigrateLine {
				output.WriteString(fmt.Sprintf("\t\tdomain.%s{},\n", moduleName))
			}
		}
	} else {
		// 如果没有找到注册路由部分，则在 // 注册路由 下添加
		for _, line := range lines {
			output.WriteString(line + "\n")
			if strings.Contains(line, "// 自动迁移") {
				output.WriteString(fmt.Sprintf("\tdb.AutoMigrate(\n\t\t// 表\n\t\tdomain.%s{},\n\t)\n", moduleName))
			}
		}
	}

	// 写回文件
	err = os.WriteFile(targetFile, []byte(output.String()), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
