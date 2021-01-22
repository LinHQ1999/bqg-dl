package scrapers

/*
仅供微调
*/

const host = "https://www.biquduo.com"

var (
	// DebugMode 调试模式
	DebugMode bool
	// 以下为选择器
	// 书名
	sBookName = "#info>h1"
	// 目录
	sContentList = "#list dd>a"
	// 章节名
	sChapterName = ".bookname h1"
	// 正文
	sContent = "#content"
)
