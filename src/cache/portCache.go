package cache

import (
	"fmt"
	"sync"
)

var (
	// cache单例锁
	oncePortCache sync.Once
	// 缓存实例
	portCacheInstance *PortCache
)

type PortCache struct {
	// 缓存读写锁
	gRWLock *sync.RWMutex
	// 缓存数据 key:端口号 value:命名空间名
	portMap map[int]string
}

func PortCacheInstance() *PortCache {
	oncePortCache.Do(func() {
		portCacheInstance = new(PortCache)
		portCacheInstance.init()
	})
	return portCacheInstance
}

func (p *PortCache) init() {
	p.gRWLock = new(sync.RWMutex)
	p.portMap = map[int]string{}
}

func (p *PortCache) ExistKeyValue(key int, value string) bool {
	p.gRWLock.RLock()
	defer p.gRWLock.RUnlock()

	if _, ok := p.portMap[key]; ok {
		if p.portMap[key] == value {
			return true
		}
	}

	return false
}

func (p *PortCache) ReloadCache(portMap map[int]string) {
	p.gRWLock.Lock()
	defer p.gRWLock.Unlock()

	p.portMap = nil
	p.portMap = portMap
}

func (p *PortCache) PrintCache()  {
	p.gRWLock.Lock()
	defer p.gRWLock.Unlock()

	fmt.Println("port cache data:")
	for k,v := range p.portMap {
		fmt.Println(k," -> ", v)
	}
}