# 浠庢媺鍙栧埌璺戦€氾紙鍚庣 + 鍓嶇 + 娴嬭瘯锛?
## 1. 鍓嶇疆鐜

- Go锛氬缓璁?`1.22+`
- Docker Desktop锛堝惈 compose锛?- Bun锛堝墠绔級
- Git

## 2. 鎷夊彇涓庤繘鍏ラ」鐩?
```powershell
git clone <your-repo-url> FlashSale
cd FlashSale
```

## 3. 鍑嗗鏈湴鐜鍙橀噺

```powershell
# 棣栨鍙鍒舵ā鏉?Copy-Item configs/deploy.env.example configs/deploy.env
```

> 闈炵敓浜х幆澧冨彲鐩存帴浣跨敤榛樿鍊硷紱濡傜鍙ｅ啿绐侊紝淇敼 `configs/deploy.env` 鐨?`FLASH_*` 绔彛銆?
## 3.1 浜や簰寮忚繍缁村叆鍙?
褰撳墠榛樿鐨?shell 杩愮淮鍏ュ彛涓烘牴鐩綍鐨?`ops.sh`锛岀敤浜庣増鏈洿鏂般€佸鍣?/ 闆嗙兢绠＄悊绛夊鍥存搷浣滐細

```powershell
bash ./ops.sh
```

## 4. 鍚姩 Docker 鍩虹鐜

```powershell
go run ./ops/cmd/fs env up
```

鍙€夛紙甯﹀彲瑙傛祴缁勪欢锛夛細

```powershell
go run ./ops/cmd/fs env up --observability
```

## 5. 鎵ц杩佺Щ涓庤繛閫氭€ф鏌?
```powershell
go run ./ops/cmd/fs env migrate-up
go run ./ops/cmd/fs env smoke
```

## 6. 鍚姩鍚庣鏈嶅姟

```powershell
go run ./ops/cmd/fs runtime start-backend
```

鏈嶅姟绔彛榛樿锛?
- user-rpc: `8081`
- user-gateway: `8082`
- admin-gateway: `8083`
- product-rpc: `8084`
- order-rpc: `8085`
- seckill-rpc: `8086`
- admin-rpc: `8087`

## 7. 鍒濆鍖栨紨绀烘暟鎹紙瑕嗗啓锛?
```powershell
go run ./ops/cmd/fs data seed-overwrite --force
```

璇ュ懡浠や細锛?
- 娓呯悊涓氬姟鏁版嵁
- 閲嶅缓瓒呯骇绠＄悊鍛樸€佺瀛愮敤鎴枫€佸晢鍝併€佹椿鍔ㄣ€佽鍗?- 杈撳嚭缁撴灉鍒?`log/data/seed-overwrite.result.json`

## 8. 鍚姩涓や釜鍓嶇

```powershell
# 瀹夎渚濊禆锛堥娆★級
go run ./ops/cmd/fs runtime start-frontend --install-deps

# 闈為娆″彲鐩存帴鍚姩
go run ./ops/cmd/fs runtime start-frontend
```

璁块棶鍦板潃锛?
- 鐢ㄦ埛绔細`http://127.0.0.1:5173`
- 绠＄悊绔細`http://127.0.0.1:5174`

## 9. 杩愯娴嬭瘯

### 9.1 鍚庣

```powershell
go test ./... -short
go vet ./...
```

### 9.2 鍓嶇

```powershell
cd frontend/user
bun run lint
bun run test:run
bun run build

cd ../admin
bun run lint
bun run test:run
bun run build
```

## 10. 绉掓潃鍘嬫祴锛堢ず渚嬶級

```powershell
# 闂幆璐拱鍘嬫祴
go run ./ops/cmd/fs perf purchase-stress -concurrency 200 -requests 4000 -timeout 7s -output json

# 骞傜瓑鍘嬫祴
go run ./ops/cmd/fs perf idempotency -concurrency 50 -requests 200 -expect-max-success 1 -output json

# 寮€鐜喘涔板帇娴?go run ./ops/cmd/fs perf purchase-open -rate 120 -open-duration 30s -concurrency 300 -output json
```

## 11. 鍋滄鏈嶅姟

```powershell
go run ./ops/cmd/fs runtime stop-frontend
go run ./ops/cmd/fs runtime stop-backend
go run ./ops/cmd/fs env down
```

濡傞渶鍒犻櫎瀹瑰櫒鍗凤細

```powershell
go run ./ops/cmd/fs env down --remove-volumes
```

