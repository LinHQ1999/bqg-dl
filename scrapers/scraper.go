package scrapers

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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

	// 并发控制
	var total int32 = 0
	max := make(chan struct{}, Threads)
	wg := sync.WaitGroup{}

	// 处理目录中链接拼接问题
	if Extend {
		m, err1 := url.Parse(meta)
		h, err2 := url.Parse(c.Host)
		if err1 != nil || err2 != nil {
			color.Red("host解析错误")
			os.Exit(2)
		}
		h.Path = m.EscapedPath()
		c.Host = h.String()
	}

	// 开始计时
	start := time.Now()
	// 获取主页信息
	page, err := http.Get(meta)
	if err != nil || page.StatusCode == 302 {
		color.Red("无法获取章节列表 %s", err.Error())
		os.Exit(2)
	}
	defer page.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(page.Body)
	name := doc.Find(c.BookName).First().Text()
	// 遍历目录, 下载书籍
	all := doc.Find(c.ContentList)
	all.Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		max <- struct{}{}
		go func(id int, url string) {
			defer func() {
				wg.Done()
				<-max
				// 进度条处理
				now := atomic.AddInt32(&total, 1)
				fmt.Printf("已下载: %.2f%% 总计%d章\r", float32(now)/float32(len(all.Nodes))*100, len(all.Nodes))
			}()
			spage, err := http.Get(c.Host + url)
			if err != nil || page.StatusCode == 302 {
				color.Red("本章下载失败！")
				return
			}
			defer spage.Body.Close()

			// 内容操作
			doc, _ := goquery.NewDocumentFromReader(convertEncoding(spage.Body))
			// 标题和内容（原始）
			title := strings.Trim(doc.Find(c.ChapterName).First().Text(), `\n~\t`)
			txt, _ := doc.Find(c.Content).First().Html()
			// 多重替换，稍后写入
			rp := str.NewReplacer("&nbsp", " ", "\n", "", "<br/>", "\n")
			// 打开文件
			f, err := os.Create(filepath.Join(tmp, fmt.Sprintf("%d.txt", id+1)))
			if err != nil {
				color.Red("无法创建文件")
				return
			}
			// 执行写入
			fmt.Fprintf(f, "%s\n\n", title)
			rp.WriteString(f, txt)

		}(i, s.AttrOr("href", ""))
		// 限制速度
		if Limit {
			time.Sleep(time.Millisecond * 150)
		}
	})
	close(max)
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

	// 生成列表
	dir, err := os.Open(tmp)
	defer dir.Close()
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

	// 跳过不需要的
	for _, v := range chunks[Jump:] {
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
		// 写入一个空行
		bf.WriteString("\n\n")
	}
	bf.Flush()
	if !Single {
		err = os.RemoveAll(tmp)
		if err != nil {
			color.Red("清理任务失败，跳过")
		}
	}
	color.HiGreen("生成完毕! 用时: %v", time.Since(start))
}

// 编码转换
func convertEncoding(rd io.Reader) io.Reader {
	if !Unicode {
		return simplifiedchinese.GBK.NewDecoder().Reader(rd)
	}
	return rd
}
