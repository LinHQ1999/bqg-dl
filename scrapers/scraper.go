package scrapers

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	str "strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// Scrape 启动
func Scrape(meta string) {
	// 环境配置
	tmp := filepath.Join("chunk")
	os.MkdirAll(tmp, os.ModeDir)
	start := time.Now()
	var total int32 = 0

	// 获取主页信息
	page, err := http.Get(meta)
	wg := sync.WaitGroup{}
	if err != nil || page.StatusCode == 302 {
		color.Red("无法获取目录！")
		os.Exit(2)
	}
	defer page.Body.Close()
	// GBK转UTF-8
	rd := simplifiedchinese.GBK.NewDecoder().Reader(page.Body)
	doc, _ := goquery.NewDocumentFromReader(rd)
	name := doc.Find(sBookName).First().Text()

	// 遍历目录, 下载书籍
	all := doc.Find(sContentList)
	all.Each(func(i int, s *goquery.Selection) {
		if DebugMode {
			log.Println(i, s.Text)
		}
		wg.Add(1)
		go func(id int, url string) {
			defer func() {
				wg.Done()
				// 进度条处理
				now := atomic.AddInt32(&total, 1)
				fmt.Printf("已完成: %.2f%%\r", float32(now)/float32(len(all.Nodes)*100))
			}()
			spage, err := http.Get(host + url)
			if err != nil || page.StatusCode == 302 {
				color.Red("本章下载失败！")
				return
			}
			rd := simplifiedchinese.GBK.NewDecoder().Reader(spage.Body)
			defer spage.Body.Close()
			doc, _ := goquery.NewDocumentFromReader(rd)

			title := doc.Find(sChapterName).First().Text()
			txt, _ := doc.Find(sContent).First().Html()
			content := str.ReplaceAll(str.ReplaceAll(txt, "&nbsp", " "), "<br/><br/>", "\n")
			f, err := os.Create(filepath.Join(tmp, fmt.Sprintf("%d.txt", id)))
			if err != nil {
				color.Red("无法创建文件")
				return
			}
			f.WriteString(fmt.Sprintf("\n%s\n\n", title))
			f.WriteString(content)
		}(i, s.AttrOr("href", ""))
		time.Sleep(time.Millisecond)
	})
	wg.Wait()

color.Green("\n下载完毕，正在生成 ...")
	// 创建目标文件
	f, err :=os.Create(path.Join(name + ".txt"))
	defer f.Close()
	if err != nil {
		color.Red("无法创文件")
		os.Exit(2)
	}
	bf := bufio.NewWriter(f)
	// 读取每一章内容
	chunks, err := ioutil.ReadDir(tmp)
	if err != nil {
		color.Red("无法获块信息")
		os.Exit(2)
	}
	for _, v := range chunks {
		ct, err := ioutil.ReadFile(path.Join(tmp, v.Name()))
		if err != nil {
			color.Red("无法获块")
			os.Exit(2)
		}
		_, err = bf.Write(ct)
		if err != nil {
			color.Red("写入失")
			os.Exit(2)
		}
	}
	bf.Flush()
	err = os.RemoveAll(tmp)
	if err != nil {
		color.Red("清理任失败，跳过")
	}
	color.HiGreen("生成完毕! 用时: %v", time.Since(start))
}
