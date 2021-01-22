package scrapers

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
		wg.Add(1)
		go func(id int, url string) {
			defer func() {
				wg.Done()
				// 进度条处理
				now := atomic.AddInt32(&total, 1)
				fmt.Printf("已下载: %.2f%%\r", float32(now)/float32(len(all.Nodes))*100)
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
			f, err := os.Create(filepath.Join(tmp, fmt.Sprintf("%d.txt", id+1)))
			if err != nil {
				color.Red("无法创建文件")
				return
			}
			f.WriteString(fmt.Sprintf("\n%s\n\n", title))
			f.WriteString(content)
		}(i, s.AttrOr("href", ""))
		time.Sleep(time.Millisecond * 150)
	})
	// 等待所有的协程完成
	wg.Wait()

	color.Green("\n下载完毕，正在生成 ...")

	// 创建目标文件
	f, err := os.Create(path.Join(name + ".txt"))
	defer f.Close()
	if err != nil {
		color.Red("无法创建文件")
		os.Exit(2)
	}
	bf := bufio.NewWriter(f)

	// 按顺序读取章列表
	dir, err := os.Open(tmp)
	if err != nil {
		color.Red("无法打开临时目录")
		os.Exit(2)
	}

	chunks, err := dir.Readdirnames(-1)
	if err != nil {
		color.Red("临时目录信息无法获取")
		os.Exit(2)
	}
	// 对文件名称进行排序
	sort.Slice(chunks, func(i, j int) bool {
		a, _ := strconv.Atoi(chunks[i][:strings.LastIndex(chunks[i], ".")])
		b, _ := strconv.Atoi(chunks[j][:strings.LastIndex(chunks[j], ".")])
		return a < b
	})
	for _, v := range chunks {
		fmt.Println(v)
		ct, err := ioutil.ReadFile(path.Join(tmp, v))
		if err != nil {
			color.Red("无法获取块")
			os.Exit(2)
		}
		_, err = bf.Write(ct)
		if err != nil {
			color.Red("写入失败")
			os.Exit(2)
		}
	}
	bf.Flush()
	//err = os.RemoveAll(tmp)
	//if err != nil {
	//	color.Red("清理任务失败，跳过")
	//}
	color.HiGreen("生成完毕! 用时: %v", time.Since(start))
}
