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
	Long: `默认支持：
	- https://www.biquduo.com/`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scrapers.Scrape(args[0])
	},
	Version: "1.1.5",
}

func init() {
	rootCmd.Flags().IntVarP(&scrapers.Jump, "jump", "j", 0, "跳过几章")
	rootCmd.Flags().BoolVar(&scrapers.Single, "single", false, "保留单章")
	rootCmd.Flags().BoolVar(&scrapers.Limit, "limit", false, "保险模式")
}

// Execute main call
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
