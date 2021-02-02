package scrapers

import (
	"bqg/utils"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	str "strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var (
	utf = false

	tmp    string
	max    chan struct{}
	wg     sync.WaitGroup
	bar    *utils.Bar
	client http.Client
)

func init() {
	// 反爬，顺便存储cookie
	jar, _ := cookiejar.New(nil)
	client = http.Client{
		Jar:     jar,
		Timeout: time.Second * 18,
	}
}

// Scrape 启动
func Scrape(meta string) {
	// 环境配置
	tmp = filepath.Join("chunk")
	os.MkdirAll(tmp, os.ModeDir)

	// 并发控制
	max = make(chan struct{}, Threads)
	wg = sync.WaitGroup{}

	// 处理目录中链接拼接问题
	if Extend {
		m, err1 := url.Parse(meta)
		h, err2 := url.Parse(c.Prefix)
		if err1 != nil || err2 != nil {
			color.Red("host解析错误")
			os.Exit(2)
		}
		h.Path = m.EscapedPath()
		c.Prefix = h.String()
	}

	// 开始计时
	start := time.Now()
	// 获取主页信息
	page, err := client.Do(mustGetRq(meta))
	if err != nil || page.StatusCode != http.StatusOK {
		color.Red("\n无法获取章节列表 %v", err)
		os.Exit(2)
	}
	defer page.Body.Close()

	doc, err := goquery.NewDocumentFromReader(page.Body)
	if err != nil {
		color.Red("%v", err)
		return
	}
	name := doc.Find(c.BookName).First().Text()
	// 编码探测
	head, _ := doc.Html()
	if str.Contains(head, "UTF") || str.Contains(head, "utf") {
		utf = true
	}
	// 遍历目录, 下载书籍
	all := doc.Find(c.ContentList)
	bar = utils.NewBar(int32(all.Length()))
	all.Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		max <- struct{}{}
		go fetchContent(i, s.AttrOr("href", ""), 3)
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
	if err != nil {
		color.Red("\n无法创建文件")
		os.Exit(2)
	}
	defer f.Close()
	bf := bufio.NewWriter(f)

	// 按数字顺序读取
	dir, err := os.Open(tmp)
	if err != nil {
		color.Red("\n无法打开临时目录")
		os.Exit(2)
	}
	chunks, err := dir.Readdirnames(-1)
	if err != nil {
		color.Red("\n临时目录信息无法获取")
		os.Exit(2)
	}
	// 需要在删除前关闭
	dir.Close()
	sort.Slice(chunks, func(i, j int) bool {
		a, _ := strconv.Atoi(chunks[i][:str.LastIndex(chunks[i], ".")])
		b, _ := strconv.Atoi(chunks[j][:str.LastIndex(chunks[j], ".")])
		return a < b
	})

	// 写入需要的内容
	for _, v := range chunks[Jump:] {
		ct, err := ioutil.ReadFile(path.Join(tmp, v))
		if err != nil {
			color.Red("\n无法获取块")
			os.Exit(2)
		}
		_, err = bf.Write(ct)
		if err != nil {
			color.Red("\n写入失败")
			os.Exit(2)
		}
		// 写入一个空行
		bf.WriteString("\n\n")
	}
	bf.Flush()

	// 确保dir已经Close
	if Single {
		err = os.RemoveAll(tmp)
		if err != nil {
			color.Red("清理任务失败，跳过")
		}
	}
	color.HiGreen("生成完毕! 用时: %v", time.Since(start))
}

// 编码转换
func g2u(rd io.Reader) io.Reader {
	if !utf {
		return simplifiedchinese.GBK.NewDecoder().Reader(rd)
	}
	return rd
}

// 生成请求
func mustGetRq(uri string) *http.Request {
	rq, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		color.Red("\n请求构建失败")
		os.Exit(2)
	}
	rq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Edg/88.0.705.53")
	return rq
}

// fetchContent 获取章节内容并写入文件
func fetchContent(id int, subpath string, retry int) {
	if retry < 0 {
		color.Red("\n达到最大重试次数，%s下载失败！", subpath)
		return
	}
	// 处理拼接路径问题
	u, _ := url.Parse(c.Prefix)
	u.Path = path.Join(u.Path, subpath)
	//fmt.Println(u.String(), "\t", subpath)
	spage, err := client.Do(mustGetRq(u.String()))
	if err != nil || spage.StatusCode != http.StatusOK {
		//color.Yellow("\n重试: %s", path.Base(subpath))
		time.Sleep(time.Second * 3)
		fetchContent(id, subpath, retry-1)
		return
	}
	defer spage.Body.Close()

	// 获取内容并格式化
	doc, err := goquery.NewDocumentFromReader(g2u(spage.Body))
	if err != nil {
		//color.Yellow("\n重试: %s", path.Base(subpath))
		time.Sleep(time.Second * 3)
		fetchContent(id, subpath, retry-1)
		return
	}
	title := doc.Find(c.ChapterName).First().Text()
	txt, _ := doc.Find(c.Content).First().Html()
	rp := str.NewReplacer("&nbsp", " ", "\n", "", "<br/>", "\n").Replace(txt)
	txt = regexp.MustCompile(`<.+>`).ReplaceAllString(rp, "")

	// 写入到文件
	f, err := os.Create(filepath.Join(tmp, fmt.Sprintf("%d.txt", id+1)))
	if err != nil {
		color.Red("\n无法创建文件")
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n\n", title)
	f.WriteString(txt)

	// 避免重试导致多减,不要放在defer里
	wg.Done()
	<-max
	fmt.Print(color.HiMagentaString(bar.AddAndShow(1)))
}
