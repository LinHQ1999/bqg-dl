package scrapers

import (
	"bqg/utils"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	str "strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	timeout    = time.Second * 30
	retryDelay = time.Second * 5
)

var (
	utf = false

	// 并发限制
	max chan struct{}
	wg  sync.WaitGroup
	// 进度条
	bar *utils.Bar

	client http.Client
	// 只读
	// 网址信息
	basic string
	// chunk 目录
	tmp string
)

func init() {
	// 反爬，存储cookie
	jar, _ := cookiejar.New(nil)
	client = http.Client{
		Jar:     jar,
		Timeout: timeout,
	}
}

/*
Scrape 直接 rerun，调用的函数中 panic
*/

// Scrape 启动
func Scrape(meta string) {
	if Dry {
		color.Green("配置文件: %v", c)
	}
	// 环境配置
	tmp = filepath.Join("chunks")
	os.MkdirAll(tmp, os.ModeDir)
	// 执行完之后清理，以及 panic 处理
	defer func() {
		if !Dry {
			os.RemoveAll(tmp)
		}
		if r := recover(); r != nil {
			color.Red("内部错误")
			if Dry {
				color.Yellow("错误原因: %v", r)
			}
		}
	}()

	// 并发控制
	max = make(chan struct{}, Threads)
	wg = sync.WaitGroup{}
	// 开始计时
	t_start := time.Now()

	// 获取章节列表
	page, err := client.Do(mustGetRq(meta, "https://cn.bing.com/"))
	if err != nil || page.StatusCode != http.StatusOK {
		color.Red("无法获取章节列表 %v", page.Header)
		return
	}
	pgContent, _ := io.ReadAll(page.Body)
	defer page.Body.Close()

	// 编码探测:`<meta ... charset="utf-8">`
	if regexp.MustCompile(`meta.+(utf|UTF)`).Match(pgContent) {
		utf = true
	}

	// 在设定utf后
	doc, err := goquery.NewDocumentFromReader(g2u(pgContent))
	if err != nil {
		color.Red("%v", err)
		return
	}

	// 获取名称
	name := doc.Find(c.Name).First().Text()

	// 获取目录，初始化进度条
	all := doc.Find(c.ContentList)
	bar = utils.NewBar(int32(all.Length()), 50)

	// 由用户选择是否扩展链接
	ex, flg := false, ""
	color.Yellow("目录的链接格式为:< %s >，使用扩展模式（保留 path）？[y/n]", all.First().AttrOr("href", ""))
	fmt.Scanf("%s", &flg)
	switch flg {
	case "y":
		ex = true
	case "n":
		ex = false
	default:
		ex = false
		color.HiRed("无效选项，设定为%v!", ex)
	}

	// 根据情况裁剪 basic，防止连接拼接出错
	m, err := url.Parse(meta)
	if err != nil {
		color.Red("host解析错误")
		return
	}
	if !ex {
		m.Path = ""
	} else {
		// 有的网站得去掉末尾 index.html
		m.Path = str.ReplaceAll(m.Path, "index.html", "")
	}
	// 存储网站链接信息
	basic = m.String()

	// 遍历并下载
	all.Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		max <- struct{}{}
		go fetchContent(i, s.AttrOr("href", ""), Retry)
	})
	close(max)
	// 等待所有的协程完成
	wg.Wait()

	color.Green("\n下载完毕（%.1f 秒），正在生成 ...", time.Since(t_start).Seconds())

	// 开始记录生成时间
	t_temp := time.Now()

	// 创建最终文件
	f, err := os.Create(path.Join(name + ".txt"))
	if err != nil {
		color.Red("\n无法创建文件")
		return
	}
	defer f.Close()
	bf := bufio.NewWriter(f)

	/*
		文件名是编号，所以可以直接根据编号打开文件，而不用全部打开再排序
	*/
	for chunkID := Jump + 1; chunkID <= all.Length(); chunkID++ {
		ct, err := os.ReadFile(path.Join(tmp, fmt.Sprintf("%d.txt", chunkID)))
		if err != nil {
			color.Red("\n无法获取指定分块(%-4d)，跳过", chunkID)
			continue
		}
		_, err = bf.Write(ct)
		if err != nil {
			color.Red("\n写入失败(%-4d)，跳过", chunkID)
			continue
		}
		// 写入一个空行
		bf.WriteString("\n\n")
	}
	bf.Flush()
	color.HiGreen("生成完毕（%d 毫秒）！", time.Since(t_temp).Milliseconds())
}

// g2u 根据utf开关转换编码
func g2u(txt []byte) io.Reader {
	if !utf {
		rs, err := simplifiedchinese.GBK.NewDecoder().Bytes(txt)
		if err != nil {
			color.Red("编码转换失败!")
			panic(err)
		}
		return bytes.NewReader(rs)
	}
	return bytes.NewReader(txt)
}

// MustGetRq Build a Request
func mustGetRq(uri string, referrer ...string) *http.Request {
	rq, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		color.Red("\n请求构建失败")
		panic(err)
	}
	rq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36 Edg/92.0.902.67")
	if !NoReferrer && len(referrer) != 0 {
		rq.Header.Set("referer", referrer[0])
	}
	return rq
}

// FetchContent 获取章节内容并写入文件
func fetchContent(id int, subpath string, retry int) {

	if retry < 0 {
		color.Red("\n达到最大重试次数，块(%-4d)下载失败！", id)
		wg.Done()
		<-max
		bar.AddAndShow(1)
		return
	}

	// 处理拼接路径问题，注意不要直接修改 basic
	bsc, _ := url.Parse(basic)
	bsc.Path = path.Join(bsc.Path, subpath)
	//fmt.Println(bsc.String(), "\t", subpath)
	// 小说页面的 Referer 是引用的 basic
	spage, err := client.Do(mustGetRq(bsc.String(), basic))
	if err != nil || spage.StatusCode != http.StatusOK {
		color.Red("\n 块(%-4d)将重新下载", id)
		time.Sleep(retryDelay)
		fetchContent(id, subpath, retry-1)
		return
	}
	defer spage.Body.Close()
	spgContent, err := io.ReadAll(spage.Body)
	if err != nil {
		color.Red("\n无法读取 块(%-4d)内容! -> %s", err.Error())
		time.Sleep(retryDelay)
		fetchContent(id, subpath, retry-1)
		return
	}

	// 获取内容并格式化
	doc, err := goquery.NewDocumentFromReader(g2u(spgContent))
	if err != nil {
		time.Sleep(retryDelay)
		fetchContent(id, subpath, retry-1)
		return
	}
	title := doc.Find(c.ChapterName).First().Text()
	txt, _ := doc.Find(c.Content).First().Html()
	rp := str.NewReplacer("&nbsp;", " ", "\n", "", "<br/>", "\n").Replace(str.TrimSpace(txt))
	txt = regexp.MustCompile(`<.+>`).ReplaceAllString(rp, "")

	// 写入到文件，第一个的 i = 0，应当从 1 开始
	f, err := os.Create(filepath.Join(tmp, fmt.Sprintf("%d.txt", id+1)))
	if err != nil {
		color.Red("\n无法创建文件")
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n\n", title)
	f.WriteString(txt)

	// 不要放在defer里，避免重试导致的多减
	wg.Done()
	<-max
	bar.AddAndShow(1)
}
