# FlashSale 鍏ㄩ」鐩繁搴︿唬鐮佽瘎瀹℃姤鍛?
**璇勫鏃ユ湡**: 2026-02-20
**璇勫浜?*: AI Code Reviewer
**璇勫鑼冨洿**: 鍏ㄩ」鐩紙鏋舵瀯 / 鍚庣 / 鍓嶇 / 鍩虹璁炬柦 / 瀹夊叏 / 鏂囨。锛?**椤圭洰鍒嗘敮**: dev

> 娉細鏈姤鍛婃槸 2026-02-20 鐨勫巻鍙茶瘎瀹″揩鐓э紝鏂囦腑鎻愬埌鐨勬棫 shell 杩愮淮浣撶郴宸插湪褰撳墠浠撳簱涓笅绾匡紝涓嶄唬琛ㄥ綋鍓嶅叆鍙ｇ粨鏋勩€?
---

## 涓€銆佹€讳綋璇勪环

> **鏁翠綋璇勭骇: 猸愨瓙猸愨瓙 (4/5) 鈥?宸ョ▼鎴愮啛搴﹂珮锛屽皯閲忔敼杩涚┖闂?*

杩欐槸涓€涓伐绋嬪畬鎴愬害寰堥珮鐨勭鏉€鐢靛晢绯荤粺銆備粠鏋舵瀯璁捐銆佷唬鐮佽川閲忋€侀儴缃蹭綋绯诲埌鏂囨。瑕嗙洊锛岄兘浣撶幇浜嗚緝楂樼殑涓撲笟姘村噯銆傛牳蹇冧寒鐐规槸绉掓潃閾捐矾鐨勭簿瀵嗚璁″拰瀹屽杽鐨勮嚜鍔ㄥ寲杩愮淮鑴氭湰浣撶郴銆?
### 浠ｇ爜缁熻

| 缁村害 | 鏁伴噺 |
|------|------|
| Go 婧愭枃浠讹紙apps/锛?| ~188 鏂囦欢 |
| Go 娴嬭瘯鏂囦欢 | 44 鏂囦欢 |
| Proto 鏂囦欢 | 5 鏂囦欢 (~1062 琛岋級 |
| pkg/base 瀛愬寘 | 15 涓?|
| 鍓嶇椤甸潰锛坲ser锛?| 9 椤?|
| 鍓嶇椤甸潰锛坅dmin锛?| 10 椤?|
| 鍓嶇 API 妯″潡 | user 11 + admin 16 |
| Shell 鑴氭湰 | 16 鏂囦欢 |
| Docker Compose | 1 涓诲叆鍙?+ 4 瀛愭枃浠?+ 2 杈呭姪 |
| 鏂囨。 | ~48 鏂囦欢 |

---

## 浜屻€佸垎缁村害璇勫

### 馃搻 2.1 鏋舵瀯璁捐 (璇勫垎: 9/10)

#### 浼樼偣

| 椤?| 璇勪环 |
|---|------|
| 寰湇鍔¤竟鐣?| 鉁?5+2 (5 RPC + 2 Gateway) 娓呮櫚鍒嗗壊锛屾瘡鏈嶅姟鍙嫭绔嬫墿缂?|
| 鍙岀綉鍏虫ā寮?| 鉁?user-gateway / admin-gateway 鏉冮檺妯″瀷瀹屽叏闅旂锛屽畨鍏ㄨ竟鐣屾竻鏅?|
| 浠ｇ爜鍒嗗眰 | 鉁?缁熶竴 6 灞傜粨鏋?(`config鈫抯vc鈫抦odel鈫抮epo鈫抣ogic鈫抯erver`)锛屽叏鏈嶅姟涓€鑷?|
| 鏈嶅姟鍙戠幇 | 鉁?浣跨敤 etcd锛実o-zero 鍘熺敓鏀寔锛岃В鑰︽湇鍔″湴鍧€ |
| 鍙娴嬫€?| 鉁?Jaeger + Prometheus + Grafana 涓変欢濂楋紝Docker profile 鍙€夊惎鍋?|
| CDN 鏋舵瀯 | 鉁?Nginx 鍙 + media-store 璇诲啓鐨?volume 鍏变韩璁捐锛岃亴璐ｆ竻鏅?|
| CLI 闆嗘垚 | 鉁?`ops/cmd` 缁熶竴鍏ュ彛锛岄泦鎴愰儴缃?绉嶅瓙/鍘嬫祴/杩佺Щ绛夎兘鍔?|

#### 鏀硅繘寤鸿

| # | 寤鸿 |
|---|------|
| A1 | Gateway 浣跨敤鍘熺敓 `http.ServeMux` 鑰岄潪 go-zero 鐨?`rest.Server`锛屾槸鏈夋剰鐨勮交閲忓寲閫夋嫨锛屼絾搴斿湪鏋舵瀯鏂囨。涓鏄庢鍙栬垗锛堝け鍘讳簡 go-zero 鍐呯疆鐨勭啍鏂?鎸囨爣/鏈嶅姟鍙戠幇绛?middleware锛?|
| A2 | 缂哄皯 API 鐗堟湰绠＄悊绛栫暐 鈥?褰撳墠 `/api/v1/` 鏄‖缂栫爜鐨勶紝娌℃湁鐪嬪埌鐗堟湰杩佺Щ璁″垝鏂囨。 |

---

### 馃敡 2.2 鍚庣 Go 浠ｇ爜 (璇勫垎: 8.5/10)

#### 浼樼偣

| 椤?| 璇勪环 |
|---|------|
| 閿欒鐮佷綋绯?| 鉁?`errorx.Code` 瀛楃涓叉灇涓?+ `HTTPStatus()` 鏄犲皠琛紝瑕嗙洊 39 绉嶄笟鍔￠敊璇爜锛屾墿灞曞弸濂?|
| gRPC 閿欒妗ユ帴 | 鉁?`grpcerr.FromStatus()` 涓?`grpcerr.ToStatus()` 鍙屽悜杞崲锛岀綉鍏充笌 RPC 涔嬮棿闆舵崯鑰?|
| JSON 瑙ｆ瀽瀹夊叏 | 鉁?`DisallowUnknownFields()` + trailing data 妫€娴嬶紝闃叉鍨冨溇瀛楁娉ㄥ叆鍜岃姹傚€嶅鏀诲嚮 |
| 绉掓潃閾捐矾 | 鉁?鏋佸叾绮惧瘑锛歊edis 缂撳瓨棰勬墸 鈫?MySQL 浜嬪姟棰勬墸(dead lock 鑷姩閲嶈瘯) 鈫?RPC 寤哄崟(channel 闂搁棬) 鈫?澶辫触琛ュ伩鍥炴粴锛屽叏閾捐矾骞傜瓑淇濇姢 |
| 闄愭祦璁捐 | 鉁?鎸夋潵婧?IP 鐨?keyed token-bucket limiter锛屾敮鎸?14 涓幆澧冨彉閲忓姩鎬侀厤缃?|
| 娴嬭瘯瑕嗙洊 | 鉁?44 涓祴璇曟枃浠讹紝瑕嗙洊鏍稿績 logic / repository / config / authz / cache / benchmark |
| 骞傜瓑璁捐 | 鉁?鍏ㄩ摼璺?`idempotency_key`锛屽簱瀛橀鎵ｅ拰寤哄崟閮芥湁骞傜瓑淇濇姢 |
| 鏉冮檺閴存潈 | 鉁?RPC 灞?`authorizeTargetUser` 鍙屼护鐗岋紙user/admin锛夎嚜鍔ㄨ瘑鍒?+ domain/data_scope 缁嗙矑搴︽鏌?|
| 鎬ц兘鎸囨爣宓屽叆 | 鉁?绉掓潃閾捐矾鍚勯樁娈靛潎鏈?`Perf.Mark*()` 绮剧粏鍩嬬偣锛屾敮鎸佸閮ㄦ€ц兘鍒嗘瀽 |

#### 鍙戠幇鐨勯棶棰?
| # | 涓ラ噸搴?| 闂 | 浣嶇疆 | 楠岃瘉鐘舵€?|
|---|--------|------|------|---------|
| B1 | 馃煛 涓?| **`writeOK` / `writeFail` / `decodeJSON` 鍦ㄤ袱涓?gateway 涓畬鍏ㄩ噸澶嶅畾涔?*锛垀50 琛岄噸澶嶄唬鐮侊級锛屽簲鎶藉彇鍒?`pkg/base/handlerx` | `apps/gateway/user/internal/handler/handler.go` vs `admin/internal/handler/handler.go` | 鉁?宸查獙璇?|
| B2 | 馃煛 涓?| **`writeRPCFail` 瀹炵幇涓嶄竴鑷?* 鈥?user gateway 瀵?`DeadlineExceeded / Unavailable / ResourceExhausted` 杩斿洖 503 + 鍙嬪ソ娑堟伅锛宎dmin gateway 娌℃湁姝ゅ鐞嗭紝鐩存帴鏆撮湶 gRPC 閿欒鐮佺粰鍓嶇 | user gateway:267-280 vs admin gateway:410-416 | 鉁?宸查獙璇?|
| B3 | 馃煝 浣?| **`panic` 鐢ㄤ簬鍚姩澶辫触** 鈥?11 澶?`panic(fmt.Sprintf(...))` 鍦ㄥ悇鏈嶅姟 main 鍑芥暟涓€傝櫧鐒跺惎鍔ㄩ樁娈?panic 鏄?Go 绀惧尯鍙帴鍙楃殑妯″紡锛屼絾缁熶竴鏀逛负 `log.Fatal` 浼氭洿涓€鑷?| 鍚?`main.go` 鏂囦欢 | 鉁?宸查獙璇?|
| B4 | 馃煝 淇℃伅 | ~~**`ListUsers` 缂哄皯 token 娉ㄥ叆**~~ 鈫?**淇锛氳繖鏄湁鎰忚璁?*銆俁PC Server 绔敞閲?"绠＄悊绔皟鐢紝鏃犵敤鎴烽壌鏉?锛宍ListUsers` 鍜?`ResetUserPassword` 涓嶉渶瑕?access token锛屽叾鏉冮檺鐢?gateway 灞?`middleware.RequireDomain(authz.RoleDomainUserManagement)` 淇濋殰 | `userrpcserver.go:96` 鍜?`:106` 娉ㄩ噴 | 鉁?澶嶅淇 |
| B5 | 馃煝 浣?| **Dashboard 缁熻 `OnSale = Total` 杩戜技鍊?* 鈥?娉ㄩ噴涓鏄庡師鍥狅紙proto 鏃?status 杩囨护瀛楁锛夛紝浣嗕袱涓€肩浉绛夊鍓嶇浣撻獙涓嶅ソ锛堢敤鎴蜂細鐤戞儜锛?| `dashboard_handler.go:141` | 鉁?宸查獙璇?|
| B6 | 馃煝 浣?| **keyedLimiterStore 浠呭湪 `len > 1024` 鏃舵墠瑙﹀彂杩囨湡娓呯悊** 鈥?浣庢祦閲忓満鏅笅杩囨湡鏉＄洰姘镐笉娓呯悊锛屾湁杞诲井鍐呭瓨娉勬紡闅愭偅銆傚缓璁鍔犲畾鏃舵竻鐞嗘垨闄嶄綆瑙﹀彂闃堝€?| `rate_limit.go:145` | 鉁?宸查獙璇?|

---

### 馃帹 2.3 鍓嶇浠ｇ爜 (璇勫垎: 8/10)

#### 浼樼偣

| 椤?| 璇勪环 |
|---|------|
| 鎶€鏈爤 | 鉁?React 19 + Vite 7 + TailwindCSS 4 + shadcn + React Query 鈥?鐜颁唬涓旂粺涓€ |
| API 灞?| 鉁?`JSONbig({ storeAsString: true })` 澶勭悊 int64 瀹夊叏锛岄伩鍏?JS 绮惧害涓㈠け |
| 浼氳瘽绠＄悊 | 鉁?鑷姩妫€娴?`AUTH_UNAUTHORIZED` / `USER_NOT_FOUND`锛岃烦杞櫥褰曚笖闃查噸澶嶈烦杞?|
| 涓夌缁熶竴 | 鉁?user / admin / ops 浣跨敤鐩稿悓鐨勭粍浠跺簱(shadcn)銆佺姸鎬佺鐞?zustand)銆佽矾鐢?react-router) |
| 绫诲瀷瀹夊叏 | 鉁?TypeScript strict mode + `tsc -b` 鏋勫缓楠岃瘉 |
| 閿欒澶勭悊 | 鉁?`ApiError` 缁熶竴鍖呰锛屽悓鏃跺鐞?2xx 涓氬姟閿欒鍜岄潪 2xx HTTP 閿欒 |

#### 鍙戠幇鐨勯棶棰?
| # | 涓ラ噸搴?| 闂 | 浣嶇疆 | 楠岃瘉鐘舵€?|
|---|--------|------|------|---------|
| F1 | 馃煛 涓?| **涓変釜鍓嶇 `package.json` 鐨?`name` 閮芥槸 `"vite-app"`** 鈥?搴旇鍒嗗埆鍛藉悕涓?`flashsale-user` / `flashsale-admin` / `flashsale-ops`锛屽惁鍒?npm/yarn 缂撳瓨鍜?monorepo 宸ュ叿鏃犳硶鍖哄垎 | `frontend/*/package.json:2` | 鉁?宸查獙璇?|
| F2 | 馃煝 浣?| **user 鍜?admin 鐨勪緷璧栧嚑涔庡畬鍏ㄤ竴鑷?*锛垀95% 鐩稿悓锛夛紝浣嗘病鏈変娇鐢?monorepo锛堝 workspaces 鎴?turborepo锛夋潵鍏变韩渚濊禆鍜岀粍浠讹紝`node_modules` 鍗犵敤纾佺洏 ~3x | `frontend/` | 鉁?宸查獙璇?|
| F3 | 馃煝 浣?| **鍚屾椂寮曞叆 `date-fns` 鍜?`dayjs` 涓や釜鏃ユ湡搴?*锛屽鍔?bundle size 绾?25KB銆傚缓璁粺涓€閫変竴涓?| `frontend/user/package.json` 鍜?`frontend/admin/package.json` | 鉁?宸查獙璇?|
| F4 | 馃煝 浣?| **ops 绔己灏?vitest / testing-library** 鈥?admin 鍜?user 鏈夋祴璇曢厤缃拰 `test` / `test:run` 鑴氭湰锛宱ps 娌℃湁 | `frontend/ops/package.json` | 鉁?宸查獙璇?|

---

### 馃彈锔?2.4 鍩虹璁炬柦 (璇勫垎: 9/10)

#### 浼樼偣

| 椤?| 璇勪环 |
|---|------|
| Compose 鏋舵瀯 | 鉁?`docker-compose.app.yml` 閫氳繃 include 鎷嗗垎涓?4 涓瓙鏂囦欢锛坕nfra / backend / proxy / observability锛夛紝鑱岃矗娓呮櫚 |
| Dockerfile | 鉁?澶氶樁娈垫瀯寤?+ parallel/serial 鍙屾ā寮忥紙`BUILD_MODE` 鐜鍙橀噺鎺у埗锛? go-build-cache 鎸傝浇 |
| 鏃ц繍缁磋剼鏈綋绯?| 鉁?褰撴椂鐨?shell 杩愮淮鍏ュ彛鏀寔鍏ㄩ噺/鍒嗙粍/鐑洿鏂扮瓑澶氱妯″紡锛屽畬鏁村害杈冮珮 |
| 鏃у啋鐑熸祴璇曚綋绯?| 鉁?褰撴椂鐨勮嚜鍔ㄥ寲娴嬭瘯鎸?infra / ops / user / auth / admin / cdn 鍒嗘缁勭粐锛屽弬鏁板寲绋嬪害杈冮珮 |
| Nginx 閰嶇疆 | 鉁?JSON 鏃ュ織銆丏ocker DNS resolver銆乲eepalive upstream銆乤uth_request 瀛愯姹傞壌鏉?|
| 杩佺Щ绠＄悊 | 鉁?`golang-migrate` + 鐙珛杩佺Щ瀹瑰櫒锛屼覆琛屽厛浜庝笟鍔″鍣ㄥ惎鍔紙compose depends_on + service_healthy锛?|
| 浣庡唴瀛樻ā寮?| 鉁?`--low-mem` 鏀寔 鈮?GB RAM 鐨勪簯鏈嶅姟鍣紝鑷姩涓茶缂栬瘧 + GOMAXPROCS=2 |

#### 鍙戠幇鐨勯棶棰?
| # | 涓ラ噸搴?| 闂 | 浣嶇疆 | 楠岃瘉鐘舵€?|
|---|--------|------|------|---------|
| I1 | 馃煛 涓?| **`docker-compose.yml` 涓?`docker-compose.app.yml` 鍏卞瓨瀵艰嚧娣锋穯** 鈥?鍓嶈€呮槸鏃╂湡寮€鍙戠増鏈紙nginx 鎸囧悜 host.docker.internal锛夛紝鍚庤€呮槸鍏ㄥ鍣ㄥ寲鐢熶骇鐗堟湰銆傚簲鏍囨敞搴熷純鎴栧垹闄?| `deploy/compose/docker-compose.yml` | 鉁?宸查獙璇?|
| I2 | 馃敶 楂?| **Kafka topic 涓嶄竴鑷?* 鈥?`docker-compose.yml` 鐨?kafka-init 浠呭垱寤?3 涓?topic锛坄order.create` / `order.create.dlq` / `stock.compensate`锛夛紝鑰?`app/infra.yml` 鍒涘缓 6 涓紙澶氬嚭 `seckill.traffic.*` 鍜?`seckill.order.state`锛夈€備娇鐢ㄦ棫 compose 鏂囦欢閮ㄧ讲浼氬鑷寸鏉€娴侀噺浜嬩欢涓㈠け | `deploy/compose/docker-compose.yml` vs `deploy/compose/app/infra.yml` | 鉁?宸查獙璇?|
| I3 | 鈥?| ~~**ops.Dockerfile 缂哄け**~~ 鈫?**淇锛氭枃浠跺瓨鍦ㄤ簬 `deploy/docker/ops.Dockerfile`** | `deploy/docker/ops.Dockerfile` | 鉁?澶嶅淇 |

---

### 馃攼 2.5 瀹夊叏 (璇勫垎: 7.5/10)

#### 瀹夊叏浼樺娍

| 椤?| 璇勪环 |
|---|------|
| 瀵嗙爜瀛樺偍 | 鉁?bcrypt 鍝堝笇锛坄golang.org/x/crypto`锛?|
| JWT 鍩熼殧绂?| 鉁?user / admin 浣跨敤涓嶅悓 secret銆乮ssuer銆乤udience |
| 璺緞绌胯秺 | 鉁?media-store 鐨?LocalStore 鏈夎矾寰勫畨鍏ㄥ寲 |
| 閴存潈鍒嗗眰 | 鉁?Gateway 灞傚煙鏉冮檺妫€鏌?+ RPC 灞?`authorizeTargetUser` 鍙岄噸鏍￠獙 |
| RBAC | 鉁?domain + data_scope 鍙岀淮搴︾害鏉?|
| 璇锋眰浣撳畨鍏?| 鉁?`DisallowUnknownFields()` + trailing data 鎷掔粷 |
| auth_request | 鉁?Nginx 瀵?media-store 璺敱浣跨敤 auth_request 瀛愯姹傦紝涓嶆毚闇?token 缁欏鎴风 |

#### 瀹夊叏闂

| # | 涓ラ噸搴?| 闂 | 浣嶇疆 | 楠岃瘉鐘舵€?|
|---|--------|------|------|---------|
| S1 | 馃敶 楂?| **Nginx media-store secret 纭紪鐮?* 鈥?`proxy_set_header Authorization "Bearer flashsale-media-dev"` 鐩存帴鍐欐锛屾敞閲婅"鐢熶骇鐜浣跨敤 envsubst 鏇挎崲"浣嗘湭瀹炴柦銆備换浣曡杩囬厤缃枃浠剁殑浜洪兘鑳界洿鎺ヤ笂浼犳枃浠?| `deploy/nginx/nginx.app.conf:122` | 鉁?宸查獙璇?|
| S2 | 馃煛 涓?| **`deploy.env` 涓?`deploy.env.example` 瀹屽叏鐩稿悓**锛坄diff` 杈撳嚭 IDENTICAL锛夛紝鍖呭惈榛樿寮卞瘑鐮?`user-secret-change-me`銆乣admin-secret-change-me`銆乣root123` 绛夈€傞儴缃叉椂濡傛灉鏈慨鏀瑰氨鐩存帴鐢?| `configs/deploy.env` vs `configs/deploy.env.example` | 鉁?宸查獙璇?|
| S3 | 馃煝 浣?| **Redis 鏃犲瘑鐮佷繚鎶?* 鈥?`redis-server --appendonly yes` 鏈 `requirepass`锛岃櫧鐒朵粎瀹瑰櫒鍐呯綉鍙揪 | `app/infra.yml` | 鉁?宸查獙璇?|
| S4 | 馃煝 淇℃伅 | **Gateway RPC timeout 榛樿 3 绉?* 鈥?`defaultRPCTimeout = 3 * time.Second`銆傚浜?Dashboard 鑱氬悎 4 涓?RPC 鐨勫苟鍙戣皟鐢ㄥ満鏅紝鍥犱娇鐢?goroutine 骞跺彂鎵€浠ヤ笉鏄?4脳3s锛岃€屾槸 max(鍚?RPC 鑰楁椂)锛? 绉掓甯告儏鍐典笅瓒冲 | 涓や釜 gateway 鐨?handler.go | 鉁?澶嶅淇 |

---

### 馃摎 2.6 鏂囨。 (璇勫垎: 7.5/10)

#### 浼樼偣

- 鏂囨。缁撴瀯瀹屽杽锛坅rchitecture + handbook + runbook + reference 鍥涚被锛?- `gateway-api-matrix.md` 璺敱鐭╅樀鏄瀬濂界殑璁捐鏂囨。
- 鍐掔儫娴嬭瘯鑴氭湰涓庢枃妗?`scripts-guide.md` 淇濇寔涓€鑷?- `from-clone-to-green.md` 鎻愪緵浜嗗畬鏁寸殑浠庨浂鍒拌窇閫氱殑鎸囧崡

#### 鍙戠幇鐨勯棶棰?
| # | 涓ラ噸搴?| 闂 | 浣嶇疆 | 楠岃瘉鐘舵€?|
|---|--------|------|------|---------|
| D1 | 馃煛 涓?| **`02-directory-preview.md` 涓嶅噯纭?* 鈥?绗?84 琛屽垪鍑?`idempotency`锛堝凡鍒犻櫎锛夛紝绗?85 琛屽垪鍑?`metrics`锛堝凡鍒犻櫎锛夈€傚悓鏃剁己灏戝疄闄呭瓨鍦ㄧ殑鍖咃細`middleware`銆乣responsex`銆乣rpcmeta`銆乣snowflakex` | `docs/architecture/02-directory-preview.md:84-85` | 鉁?宸查獙璇?|
| D2 | 馃煝 浣?| **`base-module.md` 浠嶅紩鐢ㄥ凡鍒犻櫎鐨?`idempotency` 鍖?* 鈥?绗?41-42 琛屾槧灏勪簡涓嶅瓨鍦ㄧ殑鏂囦欢璺緞 | `docs/architecture/modules/base-module.md:41-42` | 鉁?宸查獙璇?|
| D3 | 馃煝 浣?| 缂哄皯 `CONTRIBUTING.md` 鈥?瀵逛簬寰湇鍔￠」鐩紝闇€瑕佽鏄庡浣曟坊鍔犳柊 RPC 鏂规硶鐨勬爣鍑嗘祦绋嬶紙proto 鈫?codegen 鈫?logic 鈫?handler 鈫?route 鈫?test锛?| 椤圭洰鏍圭洰褰?| 鉁?宸茬‘璁?|

---

## 涓夈€侀棶棰樻眹鎬讳笌浼樺厛绾?
### 鍏ㄩ儴闂娓呭崟

| # | 涓ラ噸搴?| 绫诲埆 | 鎽樿 | 浼樺厛绾?|
|---|--------|------|------|-------|
| S1 | 馃敶 楂?| 瀹夊叏 | Nginx media-store secret 纭紪鐮?| **P0** |
| I2 | 馃敶 楂?| 鍩虹璁炬柦 | Kafka topic 鍦ㄦ柊鏃?compose 鏂囦欢涓笉涓€鑷?| **P0** |
| B1 | 馃煛 涓?| 鍚庣 | 涓や釜 gateway 閲嶅瀹氫箟 ~50 琛屽伐鍏峰嚱鏁?| P1 |
| B2 | 馃煛 涓?| 鍚庣 | `writeRPCFail` 閿欒澶勭悊閫昏緫涓嶄竴鑷?| P1 |
| S2 | 馃煛 涓?| 瀹夊叏 | `deploy.env` 浣跨敤榛樿寮卞瘑鐮佷笖涓?`.example` 瀹屽叏涓€鑷?| P1 |
| I1 | 馃煛 涓?| 鍩虹璁炬柦 | 鏃х増 `docker-compose.yml` 搴旀爣娉ㄥ簾寮冩垨鍒犻櫎 | P1 |
| D1 | 馃煛 涓?| 鏂囨。 | `02-directory-preview.md` 鍒楀嚭宸插垹闄ょ殑鍖?| P1 |
| D2 | 馃煛 涓?| 鏂囨。 | `base-module.md` 寮曠敤宸插垹闄ょ殑 idempotency 鏂囦欢 | P1 |
| F1 | 馃煛 涓?| 鍓嶇 | 涓変釜鍓嶇 package name 閮芥槸 `"vite-app"` | P1 |
| B3 | 馃煝 浣?| 鍚庣 | main 鍑芥暟鐢?panic 鏇夸唬 log.Fatal | P2 |
| B5 | 馃煝 浣?| 鍚庣 | Dashboard OnSale = Total 杩戜技鍊?| P2 |
| B6 | 馃煝 浣?| 鍚庣 | keyedLimiter 浣庢祦閲忓満鏅笉娓呯悊杩囨湡 | P2 |
| S3 | 馃煝 浣?| 瀹夊叏 | Redis 鏃犲瘑鐮佷繚鎶?| P2 |
| F2 | 馃煝 浣?| 鍓嶇 | 涓変釜鍓嶇鏈娇鐢?monorepo锛屼緷璧栭噸澶?| P2 |
| F3 | 馃煝 浣?| 鍓嶇 | 鍚屾椂寮曞叆 date-fns 鍜?dayjs | P2 |
| F4 | 馃煝 浣?| 鍓嶇 | ops 绔己灏戞祴璇曢厤缃?| P2 |
| D3 | 馃煝 浣?| 鏂囨。 | 缂哄皯 CONTRIBUTING.md | P2 |
| A1 | 馃煝 淇℃伅 | 鏋舵瀯 | Gateway 浣跨敤 http.ServeMux 鐨勮璁￠€夋嫨搴旀枃妗ｅ寲 | P2 |
| A2 | 馃煝 淇℃伅 | 鏋舵瀯 | 缂哄皯 API 鐗堟湰绠＄悊绛栫暐鏂囨。 | P2 |

### 鎸変紭鍏堢骇鐨勪慨澶嶅缓璁?
#### 馃敶 P0 鈥?寤鸿灏藉揩淇锛? 椤癸級

1. **S1: Nginx media-store secret 纭紪鐮?*
   - 鏂规 A锛堟帹鑽愶級锛氫娇鐢?`envsubst` 妯℃澘锛氬皢 `nginx.app.conf` 鏀逛负 `nginx.app.conf.template`锛宑ompose 鍚姩鏃堕€氳繃 `envsubst '$$FLASH_MEDIA_STORE_SECRET' < template > conf`
   - 鏂规 B锛氬湪 compose 涓€氳繃 `configs` 鎸傝浇鍔ㄦ€佺敓鎴愮殑閰嶇疆鐗囨

2. **I2: Kafka topic 涓嶄竴鑷?*
   - 鏂规锛氬垹闄ゆ垨鏍囨敞 `docker-compose.yml` 涓?deprecated锛岀粺涓€浣跨敤 `docker-compose.app.yml`

#### 馃煛 P1 鈥?寤鸿杩戞湡鏀硅繘锛? 椤癸級

3. **B1 + B2: Gateway 閲嶅浠ｇ爜 + 琛屼负涓嶄竴鑷?*
   - 灏?`writeOK` / `writeFail` / `writeRPCFail` / `decodeJSON` 鎶藉彇鍒?`pkg/base/handlerx`
   - 缁熶竴 admin gateway 涔熷鐞?`DeadlineExceeded / Unavailable / ResourceExhausted`

4. **S2: 榛樿寮卞瘑鐮?*
   - 鏂规 A锛氳 `deploy.env` 鎴愪负 `.gitignore` 鍐呭锛堝彧淇濈暀 `.example`锛?   - 鏂规 B锛氳嚦灏戝湪 README 閱掔洰鎻愮ず蹇呴』淇敼

5. **I1: 鏃?compose 鏂囦欢**
   - 鍦ㄦ枃浠堕《閮ㄥ姞娉ㄩ噴 `# DEPRECATED: Use docker-compose.app.yml instead` 鎴栫洿鎺ュ垹闄?
6. **D1 + D2: 鏂囨。杩囨椂**
   - 鏇存柊 `02-directory-preview.md` 绗?84-85 琛岋紝鍒犻櫎 `idempotency` / `metrics`锛岃ˉ鍏?`middleware` / `responsex` / `rpcmeta` / `snowflakex`
   - 鏇存柊 `base-module.md` 鍒犻櫎宸插垹鍖呯殑鏂囦欢鏄犲皠

7. **F1: package name 缁熶竴**
   - `frontend/user/package.json` 鈫?`"name": "flashsale-user"`
   - `frontend/admin/package.json` 鈫?`"name": "flashsale-admin"`
   - `frontend/ops/package.json` 鈫?`"name": "flashsale-ops"`

#### 馃煝 P2 鈥?寤鸿鍚庣画浼樺寲锛?0 椤癸級

8. **F2**: 鑰冭檻 pnpm workspaces / turborepo 鍏变韩渚濊禆
9. **F3**: 缁熶竴閫?`date-fns` 鎴?`dayjs`锛堟帹鑽?`date-fns`锛宼ree-shaking 鍙嬪ソ锛?10. **B6**: limiter 澧炲姞瀹氭椂娓呯悊 goroutine 鎴栭檷浣庨槇鍊艰嚦 256
11. **S3**: Redis 璁?`requirepass`锛屽嵆浣垮唴缃?12. **B5**: 缁?`ListProductsAdminReq` 鍔?`status` 杩囨护瀛楁
13. **B3**: 鑰冭檻缁熶竴鐢?`log.Fatalf` 鏇夸唬 `panic`
14. **F4**: ops 绔ˉ鍏?vitest 閰嶇疆
15. **D3**: 缂栧啓 `CONTRIBUTING.md`
16. **A1**: 鍦ㄦ灦鏋勬枃妗ｄ腑璇存槑 Gateway 鎶€鏈€夊瀷鐞嗙敱
17. **A2**: 缂栧啓 API 鐗堟湰绠＄悊绛栫暐

---

## 鍥涖€佸瀹′慨姝ｈ褰?
浠ヤ笅椤圭洰鍦ㄥ垵娆¤瘎瀹″悗缁忚繃浜屾楠岃瘉锛岃繘琛屼簡淇锛?
| 鍒濆缂栧彿 | 鍘熷缁撹 | 淇鍚庣粨璁?| 淇鍘熷洜 |
|---------|---------|-----------|---------|
| B4 | `ListUsers` 鍜?`ResetUserPassword` 缂哄皯 token 娉ㄥ叆锛堭煙?涓級 | 鏈夋剰璁捐锛屼笉鏄棶棰橈紙馃煝 淇℃伅锛?| RPC Server 娉ㄩ噴 "绠＄悊绔皟鐢紝鏃犵敤鎴烽壌鏉?锛屾潈闄愮敱 Gateway 灞?`RequireDomain` 淇濋殰 |
| I3 | `ops.Dockerfile` 鍙兘涓嶅瓨鍦紙馃煛 涓級 | 鏂囦欢瀛樺湪浜?`deploy/docker/ops.Dockerfile`锛堝垹闄ら棶棰橈級 | 鍒濇鏂囦欢鎼滅储鑼冨洿涓嶅畬鏁?|
| S4 | RPC timeout 3s 鍙兘涓嶅锛堭煙?浣庯級 | 骞跺彂璋冪敤鍦烘櫙涓?3s 瓒冲锛堭煙?淇℃伅锛?| Dashboard 浣跨敤 goroutine 骞跺彂锛屽疄闄呰秴鏃朵负 max 鑰岄潪 sum |
| A6 (Batch1) | `fs.exe` 琚彁浜ゅ埌 Git锛堭煙?涓級 | 鏈 Git 杩借釜锛堭煙?淇℃伅锛?| `/*.exe` 宸插湪 `.gitignore` 涓紝`git ls-files` 纭鏈拷韪?|

---

## 浜斻€佷寒鐐规€荤粨

鍊煎緱鐗瑰埆琛ㄦ壃鐨勮璁″拰瀹炵幇锛?
### 1. 绉掓潃璐拱閾捐矾锛坄purchase_logic.go`锛?
585 琛岄珮瀵嗗害浠ｇ爜锛岃鐩栦簡鐢熶骇绾х鏉€绯荤粺闇€瑕佽€冭檻鐨勫嚑涔庢墍鏈夎竟鐣屾儏鍐碉細
- **涓夊眰搴撳瓨鎵ｅ噺**锛歊edis 缂撳瓨棰勬墸 鈫?MySQL 浜嬪姟棰勬墸锛堝惈姝婚攣閲嶈瘯锛?鈫?RPC 寤哄崟
- **鑷€傚簲瓒呮椂**锛氭牴鎹姹傚墿浣欐椂闂村姩鎬佽绠楅鎵ｈ秴鏃讹紝淇濊瘉涓轰笅娓哥暀澶熶綑閲?- **闂搁棬鎺у埗**锛歚OrderCreateLimiter` channel 闄愬埗寤哄崟骞跺彂锛岄槻姝?DB 闆穿
- **骞傜瓑鎭㈠**锛氬箓绛夊啿绐佹椂涓嶇洿鎺ュけ璐ワ紝鑰屾槸灏濊瘯閲嶆斁寤哄崟鎭㈠
- **琛ュ伩鍥炴粴**锛氭瘡涓け璐ヨ矾寰勯兘鏈夊搴旂殑搴撳瓨琛ュ伩鍜岀紦瀛樺洖婊?- **鎬ц兘鍩嬬偣**锛氭瘡涓叧閿楠ら兘鏈?`Perf.Mark*()` 璋冪敤

### 2. 鏃ц繍缁磋剼鏈綋绯?
褰撴椂鐨?legacy shell 杩愮淮浣撶郴鏋勬垚浜嗕竴濂楀畬鏁寸殑 CI/CD 鏇夸唬鏂规锛?- 鏀寔鍏ㄩ噺/鍗曢泦缇?鐑洿鏂颁笁绉嶆ā寮?- 浣庡唴瀛樻ā寮忛€傞厤 鈮?GB 浜戞湇鍔″櫒
- 鍐掔儫娴嬭瘯 6 section 鍏ㄨ嚜鍔ㄥ寲

### 3. 閿欒鐮佷綋绯?
`pkg/base/errorx` 鐨?`Code` 鈫?`HTTPStatus()` 鏄犲皠 + `grpcerr` 鍙屽悜妗ユ帴锛屽疄鐜颁簡浠?RPC 鍒?HTTP 鐨勯浂鎹熻€楅敊璇紶閫掞紝鍓嶇鍙互鐩存帴浣跨敤 `code` 瀛楁鍋氫笟鍔″垽鏂€?
---

## 鍏€佽瘎瀹℃柟娉曡鏄?
鏈璇勫閲囩敤浠ヤ笅鏂规硶锛?
1. **鏂囦欢閬嶅巻**锛氶€氳繃 `find_by_name` 閬嶅巻椤圭洰缁撴瀯
2. **婧愮爜闃呰**锛氶€愭枃浠堕槄璇诲叧閿ā鍧楋紙gateway handler銆丷PC server銆乴ogic銆乵iddleware銆乵odel銆乧onfig锛?3. **妯″紡鎼滅储**锛氶€氳繃 `grep_search` 鏌ユ壘 `TODO`銆乣FIXME`銆乣panic`銆乣sql.DB` 绛夋ā寮?4. **瀵规瘮楠岃瘉**锛氬 `deploy.env` vs `.example`銆乽ser gateway vs admin gateway 鐨勮涓轰竴鑷存€?5. **閾捐矾杩借釜**锛氫粠 gateway handler 鈫?RPC server 鈫?logic 鈫?repository 閫愬眰杩借釜鍏抽敭鍑芥暟璋冪敤
6. **澶嶅淇**锛氬鍒濇鍙戠幇鐨勯棶棰樿繘琛屼簩娆￠獙璇侊紝淇璇垽

---

*鎶ュ憡缁撴潫*


