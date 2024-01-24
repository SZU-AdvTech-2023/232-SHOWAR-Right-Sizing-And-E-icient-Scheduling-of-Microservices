# 介绍
本项目是基于“SHOWAR: Right-Sizing And Eﬀicient Scheduling of Microservices”论文的复现代码
文章链接参考 [https://dl.acm.org/doi/abs/10.1145/3472883.3486999](https://dl.acm.org/doi/abs/10.1145/3472883.3486999)

# 项目结构说明
[affinityRuleGenerator](affinityRuleGenerator) 目录是更具cpu来生成亲和性的代码入口

[autoscaling](autoscaling) 是伸缩器的入口

[prometheus](prometheus) 是获取Prometheus上指标的计算工具类

[monitoring](monitoring) 用于监控集群运行报错pod的工具，用于实验结果的测试

[其他]() 其他目录都是一些工具类或者在写代码的时候进行相关组件测试的代码

# 依赖安装
1. 首选需要确认你的环境是否有golang的环境
2. 直接使用go mod tidy 安装所需要的依赖
3. 本地需要有kubernetes集群的入口令牌文件，需要从kubernetes服务器上面获取config文件，放置到C:/Users/用户目录/.kube/ 目录下面
4. kubernetes集群需要安装prometheus组件以及istio组件，相关文档可以参考：
   https://github.com/prometheus-operator/kube-prometheus 、 https://istio.io/latest/zh/docs/

# 项目启动
1. 进入autoscaling/cmd2目录下
2. 运行 go run AutoRun.go 命令
