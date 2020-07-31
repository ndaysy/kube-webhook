**说明**  
kubernetes ValidatingAdmissionWebhook插件，用于管理service的nodeport端口，防止用户随意指定端口导致互相冲突。

端口按照namespace分配管理，只有分配给对应namespace的service才能创建成功
允许配置例外namespace，这些namespace不用提前申请nodeport

**运行参数示例**  
kube-webhook --port=6443 -tlsCertPath=res/cert.pem -tlsKeyPath=res/key.pem -portConfigFile=res/ports.conf -alsologtostderr -ignoredNS=testns

**镜像构建**  
git clone https://github.com/ndaysy/kube-webhook.git
cd kube-webhook
docker build -t reg.harbor.com/ikube/kube-webhook:v2 .

**部署**  
将构建好的镜像导入自己镜像库
docker push reg.harbor.com/ikube/kube-webhook:v2

cd kube-webhook/deployment

按需修改6-deployment.yaml中镜像地址

脚本2-webhook-patch-ca-bundle.sh对jq有依赖，如果没有请执行 sh 00-install-jq.sh 命令安装

依次执行：
kubectl apply -f 0-namespace.yaml
sh 1-webhook-create-signed-cert.sh
sh 2-webhook-patch-ca-bundle.sh
kubectl apply -f 3-validatingwebhook-ca.yaml
kubectl apply -f 4-configmap.yaml
kubectl apply -f 5-service.yaml
kubectl apply -f 6-deployment.yaml

**确认插件状态**
kubectl get all -n ikube

**nodeport端口授权配置**
nodeport授权数据保存在ikube namespace下从kube-webhook-configmap 配置中
可以通过编辑该configmap完成nodeport权限管理：
kubectl edit configmap kube-webhook-configmap

或者编辑4-configmap.yaml进行管理：
kubectl apply -f 4-configmap.yaml

权限配置为ports.conf段，规则为：
NodePort按照namespace授权，即指定namespace可用的一到多个端口
每个namespace一行，与端口以:分隔，注意缩进
多个端口以逗号分隔，如 30000,30001
连续端口以-连接，开始端口必须小于结束端口，如 30010-30020
端口范围 1到65535
错误的端口配置将被跳过，仅有效配置部分生效

配置变更后需要30秒左右生效

**在namespace上启用nodeport管理**
为需要管控nodeport端口的namespace增加label： admission-kube-webhook=enabled即可启用nodeport管理
kubectl label ns {namespaceName} admission-kube-webhook=enabled --overwrite=true

删除label或修改其值为非enabled则取消端口管控

在启用了nodeport管控的namespace中的某个nodeport类型service如果需要禁用端口管控，可以添加注解：
ikube.com/validate: 'no'