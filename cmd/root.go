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
	Long: `目前仅支持：
	- https://www.biquduo.com/`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scrapers.Scrape(args[0])
	},
}

// Execute main call
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().BoolVar(&scrapers.DebugMode, "debug", false, "调试模式")
}
