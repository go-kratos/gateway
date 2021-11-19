### run
```bash
sudo docker-compose up -d
# 修改 conf/config.yaml 里面的后端配置
./wrk -t12 -c400 -d30s --latency http://192.168.9.164:8888/helloworld/foo
```

