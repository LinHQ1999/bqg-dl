package scrapers

import (
	_ "embed"
	"os"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// 配置参数
var (
	// Threads 线程数
	Threads int
	// Retry 重试次数
	Retry int
	// Jump 跳过前面几章
	Jump int
	// Referrer 不要探测 referer
	NoReferrer bool
	// Dry 保留单章
	Dry bool
	// 配置文件
	c *C
)

// C 配置信息
type C struct {
	Book
	Chapter
}

// Book 主页设置
type Book struct {
	// Name 书名
	Name string
	// ContentList 目录
	ContentList string
}

// Chapter 章节及内容设置
type Chapter struct {
	// ChapterName  章节名
	ChapterName string
	// Content  正文
	Content string
}

//go:embed bqg.52.yml
var cfg []byte

// ReadConfig 从配置文件读取配置
func ReadConfig() {
	if _, err := os.Stat("bqg.yml"); os.IsNotExist(err) {
		color.Green("配置文件已创建!")
		os.WriteFile("bqg.yml", cfg, os.ModePerm)
	}
	data, _ := os.ReadFile("bqg.yml")
	c = new(C)
	yaml.Unmarshal(data, &c)
}
