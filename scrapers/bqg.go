package scrapers

import (
	"os"
	"path"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

/*
仅供微调
*/

const (
	retryDelay = time.Second * 5
	timeout    = time.Second * 30
)

// 间隔时间
var (
	// Extend 扩展基准url
	Extend bool
	// Threads 线程数
	Threads int
	// Retry 重试次数
	Retry int
	// Jump 跳过前面几章
	Jump int
	// Single 保留单章
	Single bool
	// 配置文件的配置
	c *C
)

func init() {
	initConfig()
	c = new(C)
	viper.Unmarshal(c)
}

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

func initConfig() {
	viper.SetConfigName("bqg")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path.Join("."))

	// 配置默认值
	viper.SetDefault("Book.Name", "#info>h1")
	viper.SetDefault("Book.ContentList", "#list dd>a")
	viper.SetDefault("Chapter.ChapterName", ".bookname h1")
	viper.SetDefault("Chapter.Content", "#content")

	if err := viper.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			err := viper.WriteConfigAs(path.Join("bqg.yml"))
			if err != nil {
				color.Red("配置文件写入失败!")
				os.Exit(2)
			}
			color.Yellow("配置文件已生成")
		default:
			color.Red("无法读取配置文件")
			os.Exit(2)
		}
	}
}
