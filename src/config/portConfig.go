package config

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/golang/glog"
	//"github.com/sdbaiguanghe/glog"
	"io"
	"kube-webhook/src/cache"
	"os"
	"strconv"
	"strings"
)

//var (
//	configMD5  = ""
//	configPath = "ports.conf"
//)

type PortConfig struct {
	configMD5  string
	configPath string
}

func (p *PortConfig) Init(configPath string) {
	p.configMD5 = ""
	p.configPath = configPath
}

func (p *PortConfig) RefreshPortConfig() {
	curMD5 := p.getFileMd5(p.configPath)
	if curMD5 != "" && curMD5 != p.configMD5 {
		glog.Infof("ports config changed, reloading cache ...")
		if p.loadPortConfig(p.configPath) == nil {
			p.configMD5 = curMD5
		}
		cache.PortCacheInstance().PrintCache()
	}
}

func (p *PortConfig) getFileMd5(filename string) string {
	f, err := os.Open(filename)
	defer f.Close()

	if err != nil {
		glog.Errorf("cannot open file: %s. err: %v", filename, err)
		return ""
	}

	md5hash := md5.New()
	if _, err := io.Copy(md5hash, f); err != nil {
		glog.Errorf("copy %s failed. err: %v", filename, err)
		return ""
	}

	return hex.EncodeToString(md5hash.Sum(nil))
}

func (p *PortConfig) loadPortConfig(configPath string) error {
	file, err := os.Open(configPath)
	defer file.Close()
	if err != nil {
		glog.Errorf("cannot open file: %s. err: %v", configPath, err)
		return err
	}

	portsMap := map[int]string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineSrc := scanner.Text()
		p.getPortsInline(lineSrc, portsMap)
	}

	if err := scanner.Err(); err != nil {
		glog.Errorf("cannot scanner config file: %s, err: %v", configPath, err)
		return err
	}

	// 刷新缓存
	cache.PortCacheInstance().ReloadCache(portsMap)

	return nil
}

// 解析端口配置字符串，获取所有端口号
// 端口配置按照逗号切分后，如果某段配置有错误会跳过继续处理下一个分割后的字符串
// 不会因为某一段配置错误导致所有端口都无法生成
//  如存在错误的配置：80,-,81-,-82,83--84,,,90-85,100 解析后端口为：80,100
// srcPortStr: 端口配置，如：8080,9000-9100，端口范围 1 - 65535
// namespace: 返回结果map的value，为端口所在的namespace
// portMap: 存放返回结果，key: 端口号 value: 端口对应的namespace
func (p *PortConfig) getPortsInline(lineSrc string, portsMap map[int]string) {
	lineTmp := lineSrc
	if strings.Trim(lineTmp, " ") == "" {
		return
	}

	lineTmp = strings.ReplaceAll(lineTmp, " ", "")
	// 检查行内是否有:分隔符，没有说明配置错误，或者以:开头、结尾也是错误配置;:大于1个错误
	sepCount := strings.Count(lineTmp, ":")
	if sepCount == 0 || sepCount > 1 || strings.HasSuffix(lineTmp, ":") || strings.HasPrefix(lineTmp, ":") {
		glog.Errorf("port config error:(001) number of : is incorrect. line: %s", lineSrc)
		return
	}

	// 安装 : 分割出 namespace 和 端口配置字符串
	nsConfArray := strings.Split(lineTmp, ":")
	namespace := nsConfArray[0]
	lineTmp = nsConfArray[1]

	// 按逗号分割
	portItemArray := strings.Split(lineTmp, ",")
	for _, v := range portItemArray {
		if v == "" {
			continue
		}

		// 不能以 - 开头或结尾
		if strings.HasPrefix(v, "-") || strings.HasSuffix(v, "-") {
			glog.Errorf("port config error:(002) port cannot start or end with -. line: %s", v)
			continue
		}

		// 每个端口配置中最多只能有一个 - 连接符
		portArr2 := strings.Split(strings.Trim(v, "-"), "-")
		if len(portArr2) > 2 {
			glog.Errorf("port config error:(003) too many - in line. line: %s", v)
			continue
		}

		// 检查端口是否为数字，并且在1 - 65535 之间
		startPort, err := p.filterPort(portArr2[0])
		if err != nil {
			glog.Errorf("%v", err)
			continue
		}

		// 只有一个开始端口，直接添加
		if len(portArr2) == 1 {
			portsMap[startPort] = namespace
			continue
		}

		// 端口配置为一个范围，检查结束端口是否为数字，并且在1 - 65535 之间
		endPort, err2 := p.filterPort(portArr2[1])
		if err2 != nil {
			glog.Errorf("%v", err2)
			continue
		}

		// 检查开始端口是否小于结束端口
		if startPort > endPort {
			glog.Errorf("port config error:(006) start port should bigger than end port. start: %d end: %d ", startPort, endPort)
			continue
		}

		// 循环添加范围端口
		for {
			if startPort > endPort {
				break
			}
			// key: 端口号 value: 端口对应的namespace
			portsMap[startPort] = namespace
			startPort++

		}
	}
}

// 端口合法性检查，端口应该为数字并且在 1 -65535 之间
// return: int型端口 error：非数字或范围超出异常
func (p *PortConfig) filterPort(portString string) (int, error) {
	// 端口是否为数字
	port, err := strconv.Atoi(portString)
	if err != nil {
		return 0, errors.New("port config error:(004) port should be a number. port: " + portString)
	}

	// 端口是否在 1 - 65535 之间
	if port < 1 || port > 65535 {
		return 0, errors.New("port config error:(005) port should be a number between 1 to 65535")
	}
	return port, nil
}
