# FlashSale 鍩虹灞備笌鏍稿績妯″潡浠撳簱

鏈粨搴撴壙杞藉熀浜?`go-zero/goctl` 鐨勫熀纭€鑳藉姏涓庡凡钀藉湴鏍稿績妯″潡瀹炵幇銆?
## 褰撳墠瀹炵幇鑼冨洿

- 鏈湴 Docker 鐜锛歁ySQL銆丷edis銆並afka銆乪tcd
- etcd 鏈嶅姟娉ㄥ唽涓庡彂鐜帮細鎵€鏈?RPC 鏈嶅姟鑷姩娉ㄥ唽锛孏ateway 瀹㈡埛绔姩鎬佸彂鐜?- 鍙€夊彲瑙傛祴鎬х幆澧冿細Jaeger銆丳rometheus銆丟rafana锛堝惈棰勯厤 Dashboard锛?- 鍏叡 Go 鍩虹鍖咃細`pkg/base/*`
- 鍒嗗簱杩佺Щ鑴氭湰锛歚deploy/migrations/*`
- 鍩虹杩為€氭€ц嚜妫€锛歚ops/cmd/smoke/*`
- 鐢ㄦ埛 RPC 鏈嶅姟锛歚apps/user/rpc`
- 鍟嗗搧 RPC 鏈嶅姟锛歚apps/product/rpc`
- 璁㈠崟 RPC 鏈嶅姟锛歚apps/order/rpc`
- 绉掓潃 RPC 鏈嶅姟锛歚apps/seckill/rpc`
- 绠＄悊鍛?RPC 鏈嶅姟锛歚apps/admin/rpc`
- 鐢ㄦ埛缃戝叧锛歚apps/gateway/user`
- 绠＄悊鍛樼綉鍏筹細`apps/gateway/admin`

## 蹇€熷紑濮?
1. 鍚姩鍩虹鐜锛?
```powershell
go run ./ops/cmd env up
```

2. 鎵ц鏁版嵁搴撹縼绉伙細

```powershell
go run ./ops/cmd env migrate-up
```

3. 鎵ц smoke 妫€鏌ワ細

```powershell
go run ./ops/cmd env smoke
```

4. 鍋滄鍩虹鐜锛?
```powershell
go run ./ops/cmd env down
```

鍚敤鍙娴嬫€х粍浠讹細

```powershell
go run ./ops/cmd env up --observability
```

## 闆嗙兢绾у井鏈嶅姟鑳藉姏

鏈」鐩€氳繃 etcd 瀹炵幇鐪熸鐨勬湇鍔℃敞鍐屼笌鍙戠幇锛屽嵆浣垮湪鍗曟湇鍔″櫒閮ㄧ讲涓嬩篃鍏峰闆嗙兢绾ц兘鍔涳細

- **鏈嶅姟娉ㄥ唽**锛? 涓?RPC 鏈嶅姟鍚姩鍚庤嚜鍔ㄦ敞鍐屽埌 etcd锛圞ey: `user.rpc` / `product.rpc` / `order.rpc` / `seckill.rpc` / `admin.rpc`锛?- **鏈嶅姟鍙戠幇**锛? 涓?Gateway 閫氳繃 etcd 鍔ㄦ€佸彂鐜颁笅娓?RPC 瀹炰緥锛屾棤闇€纭紪鐮佸湴鍧€
- **瀹㈡埛绔礋杞藉潎琛?*锛歡o-zero 鍐呯疆 round-robin锛屾敮鎸?`docker compose up --scale seckill-rpc=3` 姘村钩鎵╁睍
- **缁撴瀯鍖栬娴?*锛歂ginx JSON access log + Prometheus 鎸囨爣閲囬泦 + Grafana 棰勯厤 Dashboard

鍏ㄥ鍣ㄥ寲閮ㄧ讲锛堝惈 etcd锛夛細

```powershell
# 鍚姩鍏ㄩ儴鏈嶅姟
go run ./ops/cmd env app-up

# 甯﹁娴嬫€х粍浠?docker compose -f deploy/compose/docker-compose.app.yml --profile observability up -d

# 鎵╁睍绉掓潃鏈嶅姟鍒?3 瀹炰緥
docker compose -f deploy/compose/docker-compose.app.yml up -d --scale seckill-rpc=3
```

## 浜や簰寮忚繍缁村叆鍙?
褰撳墠榛樿鐨勮繍缁村叆鍙ｄ负鏍圭洰褰曡剼鏈?`ops.sh`锛岀敤浜庢壙鎺ョ増鏈洿鏂般€佸鍣?闆嗙兢绠＄悊绛夊鍥存搷浣溿€?
```powershell
# Git Bash
bash ./ops.sh
```

鏃х殑 shell 鑴氭湰鍏ュ彛宸蹭粠褰撳墠杩愮淮鍏ュ彛涓Щ闄わ紝鐩稿叧鑳藉姏宸插垏鎹㈠埌 `ops.sh`銆?
## 鐙珛杩愮淮鎺у埗鍙帮紙绗竴鐗堬級

杩愮淮鎺у埗鍙颁笌涓氬姟鏈嶅姟瑙ｈ€︼紝鏀寔 Web 鍙鍖栦笌 CLI 鍙屽叆鍙ｏ細

```powershell
# 0) 閰嶇疆璁块棶瀵嗛挜锛堝缓璁啓鍏?configs/deploy.env锛?$env:FLASHSALE_OPS_ACCESS_KEY="replace_me_strong_key"

# 1) 鍚姩鐙珛鍙鍖栫粍浠讹紙瀹瑰櫒鏃ュ織涓庡鍣ㄧ鐞嗭級
go run ./ops/cmd env ops-up

# 2) 鍚姩 ops-control锛堜换鍔＄紪鎺?API + Web 椤甸潰锛?go run ./ops/cmd ops server --addr 0.0.0.0:18080 --repo-root . --auth-key-env FLASHSALE_OPS_ACCESS_KEY

# 3) 璁块棶
# ops-control: http://127.0.0.1:18080
# dozzle:     http://127.0.0.1:18081
# portainer:  http://127.0.0.1:19000
```

CLI 涔熷彲鐩存帴璋?ops-control锛?
```powershell
```

杩滅▼璁块棶寤鸿锛?- 浠呭紑鏀?`18080` 鍒板彈淇＄綉缁溿€?- 鐢熶骇寤鸿鍦?Nginx 鍚庢寕 TLS锛屽苟鍔?IP 鐧藉悕鍗曘€?- 瀵嗛挜鍙斁鐜鍙橀噺锛屼笉鍐欏叆浠撳簱銆?
## 绔彛涓庤繛鎺ラ厤缃?
寮€鍙?CLI 缁熶竴璇诲彇锛歚configs/deploy.env`

- `FLASH_*`锛欴ocker 瀵瑰绔彛
- `FLASHSALE_*`锛氬簲鐢ㄨ繛鎺ヨ鐩栧弬鏁?- `FLASH_MYSQL_ROOT_PASSWORD` / `FLASH_MYSQL_APP_PASSWORD`锛氭暟鎹簱瀹瑰櫒涓庡簲鐢ㄨ处鍙峰瘑鐮?- `FLASH_GRAFANA_ADMIN_USER` / `FLASH_GRAFANA_ADMIN_PASSWORD`锛欸rafana 绠＄悊鍛樺嚟鎹?
濡傛灉鏈満绔彛琚崰鐢紝鍙渶淇敼 `configs/deploy.env`锛屽啀鎵ц `fs` 鍛戒护鍗冲彲銆?
## 缁勪欢鐗堟湰

- MySQL: `mysql:8.4`
- Redis: `redis:7.2-alpine`
- Kafka: `confluentinc/cp-kafka:7.6.1`
- etcd: `quay.io/coreos/etcd:v3.5.18`
- Jaeger: `jaegertracing/all-in-one:1.57`
- Prometheus: `prom/prometheus:v2.53.1`
- Grafana: `grafana/grafana:10.4.5`

## 闀滃儚 Digest 璁板綍

棣栨鎷夊彇闀滃儚鍚庯紝鍙墽琛岋細

```powershell
docker image inspect --format='{{index .RepoDigests 0}}' mysql:8.4
docker image inspect --format='{{index .RepoDigests 0}}' redis:7.2-alpine
docker image inspect --format='{{index .RepoDigests 0}}' confluentinc/cp-kafka:7.6.1
```

骞跺皢缁撴灉鐧昏鍒?`docs/operations/image-digests.md`銆?
## 鏋舵瀯鏂囨。

- 鏂囨。绱㈠紩锛歚docs/README.md`
- 绯荤粺涓庝唬鐮佹€昏锛歚docs/architecture/01-system-and-code-architecture.md`
- 鏈嶅姟鍙戠幇涓庨泦缇よ兘鍔涳細`docs/architecture/03-service-discovery.md`
- 鐩綍棰勮锛歚docs/architecture/02-directory-preview.md`
- 缃戝叧妯″潡锛歚docs/architecture/modules/gateway-module.md`
- 鐢ㄦ埛妯″潡锛歚docs/architecture/modules/user-module.md`
- 鍟嗗搧妯″潡锛歚docs/architecture/modules/product-module.md`
- 璁㈠崟妯″潡锛歚docs/architecture/modules/order-module.md`
- 绉掓潃妯″潡锛歚docs/architecture/modules/seckill-module.md`
- 绠＄悊鍛樻ā鍧楋細`docs/architecture/modules/admin-module.md`


