package cmd

import (
	"bqg/scrapers"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bqg <书籍主页链接>",
	Short: "下载笔趣阁的书籍",
	Long: `默认配置：https://www.biduoxs.com，

	其它支持网站可参考config中的配置文件。
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scrapers.Scrape(args[0])
	},
	Version: "1.6.3",
}

func init() {
	rootCmd.Flags().IntVarP(&scrapers.Jump, "jump", "j", 0, "跳过几章")
	rootCmd.Flags().IntVarP(&scrapers.Threads, "threads", "t", 16, "协程数量")
	rootCmd.Flags().IntVarP(&scrapers.Retry, "retry", "r", 5, "重试次数")
	rootCmd.Flags().BoolVar(&scrapers.Dry, "dry", false, "测试模式")
}

// Execute main call
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
