package scrapers

import (
	"os"
	"path"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

/*
仅供微调
*/

// 间隔时间
var (
	// Jump 跳过前面几章
	Jump int
	// Single 保留单章
	Single bool
	c      *C
)

func init() {
	initConfig()
	c = new(C)
	viper.Unmarshal(c)
}

// C 配置信息
type C struct { // Host 域名
	Host string
	// 以下为选择器
	//SBookName 书名
	BookName string
	//SContentList 目录
	ContentList string
	//SChapterName  章节名
	ChapterName string
	//SContent  正文
	Content string
}

func initConfig() {
	viper.SetConfigName("bqg")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path.Join("."))

	// 配置默认值
	viper.SetDefault("Host", "https://www.biquduo.com")
	viper.SetDefault("BookName", "#info>h1")
	viper.SetDefault("ContentList", "#list dd>a")
	viper.SetDefault("ChapterName", ".bookname h1")
	viper.SetDefault("Content", "#content")

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
