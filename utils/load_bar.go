package utils

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/fatih/color"
)

// Bar 进度条
type Bar struct {
	total int32
	// 不要直接读取，可能读到脏数据
	current int32
	length  int
}

var (
	l = color.New(color.BgHiGreen).SprintFunc()
	r = color.New(color.BgHiRed).SprintFunc()
)

// NewBar 获取进度条实例
func NewBar(total int32, length int) *Bar {
	return &Bar{
		total,
		0,
		length,
	}
}

// AddAndShow 增加计数，并返回进度
func (bar *Bar) AddAndShow(delta int32) {
	now := atomic.AddInt32(&bar.current, delta)
	// 避免脏读，最好使用返回的值
	div := float32(now) / float32(bar.total)
	loaded := int(div * float32(bar.length))
	fmt.Fprintf(color.Output, "进度: |%s%s| %.2f%%\r",
		l(strings.Repeat(" ", loaded)),
		r(strings.Repeat(" ", bar.length-loaded)),
		div*100,
	)
}
