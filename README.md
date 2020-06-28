用于管理service的nodeport端口，防止用户随意指定端口导致互相冲突。

端口按照namespace分配管理，只有分配给对应namespace的service才能创建成功
允许配置例外namespace，这些namespace不用提前申请nodeport

启动参数示例：
--port=6443 -tlsCertPath=res/cert.pem -tlsKeyPath=res/key.pem -portConfigFile=res/ports.conf -alsologtostderr -ignoredNS=testns
