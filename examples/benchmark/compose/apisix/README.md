### run
```bash
sudo docker-compose -p docker-apisix up -d
# 在浏览器中打开 ip:9000 并配置
./wrk -t12 -c400 -d30s --latency http://192.168.9.164:9080/helloworld/foo
```
