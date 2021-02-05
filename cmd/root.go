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
	Long: `默认配置：https://www.biquduo.com，

	其它支持网站可参考config中的配置文件。
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scrapers.Scrape(args[0])
	},
	Version: "1.4.0",
}

func init() {
	rootCmd.Flags().IntVarP(&scrapers.Jump, "jump", "j", 0, "跳过几章")
	rootCmd.Flags().IntVarP(&scrapers.Threads, "threads", "t", 16, "协程数量")
	rootCmd.Flags().IntVarP(&scrapers.Retry, "retry", "r", 8, "重试次数")
	rootCmd.Flags().BoolVar(&scrapers.Single, "single", true, "单个文件")
	rootCmd.Flags().BoolVar(&scrapers.Extend, "ex", false, `扩展host以支持特定网站，比如书籍链接不是
	完整的path的`)
}

// Execute main call
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
