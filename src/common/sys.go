package common

import (
	"fmt"
	"github.com/golang/glog"
	"runtime"
)

type SYS struct{}

// 获取当前应用的内存信息字符串
// 避免频繁调用，会挂起其他线程
// Alloc：     当前堆上对象占用的内存大小;
// TotalAlloc：堆上总共分配出的内存大小;
// Sys：       程序从操作系统总共申请的内存大小;
// NumGC：     垃圾回收运行的次数
func (s *SYS) memoryInfo() string {

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("Alloc:%vKb TotalAlloc:%vKb Sys:%vKb NumGC:%v", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
}

// 打印应用内存信息
func (s *SYS) PrintMemoryInfo() {
	fmt.Println(s.memoryInfo())
}

// 记录应用内存信息到日志
func (s *SYS) LogMemoryInfo() {
	glog.Info(s.memoryInfo())
}

// 主动垃圾回收
func (s *SYS) GC() {
	runtime.GC()
}
