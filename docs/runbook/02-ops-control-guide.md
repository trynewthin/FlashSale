# Ops 鎺у埗鍙颁娇鐢ㄦ寚鍗?
## 1. 瀹氫綅

Ops 鎺у埗鍙版槸鐙珛浜庝笟鍔″悗鍙扮殑鈥滆繍缁存帶鍒堕潰鈥濓紝涓昏鐢ㄤ簬锛?
- 鍚仠鐜涓庢湇鍔?- 鏁版嵁娓呯悊涓庤鍐欏～鍏?- 瑙﹀彂鍘嬫祴浠诲姟
- 瀹炴椂鏌ョ湅浠诲姟鏃ュ織锛圫SE 娴佸紡锛?
## 2. 鍚姩鏂瑰紡

### 2.0 浜や簰寮?shell 鍏ュ彛

濡傛灉鍙槸鍋氱増鏈洿鏂般€佸鍣ㄥ惎鍋溿€侀泦缇ゆ瀯寤虹瓑澶栧洿鎿嶄綔锛屼紭鍏堜娇鐢ㄦ牴鐩綍鐨?`ops.sh`锛?
```powershell
bash ./ops.sh
```

### 2.1 鍚姩鍙鍖栬繍缁村鍣紙鍙€夛級

```powershell
go run ./ops/cmd/fs env ops-up
```

榛樿绔彛锛?
- dozzle锛歚18081`
- portainer锛歚19000`

### 2.2 鍚姩 ops-control 鏈嶅姟

```powershell
$env:FLASHSALE_OPS_ACCESS_KEY="replace_me"
go run ./ops/cmd/fs runtime start-ops-control --addr 0.0.0.0:18080 --auth-key-env FLASHSALE_OPS_ACCESS_KEY
```

璁块棶锛歚http://127.0.0.1:18080`

## 3. CLI 璋冪敤 ops-control

```powershell
# 鍒椾换鍔?
# 瑙﹀彂浠诲姟

# 鐪嬩换鍔″垪琛?
# 鐪嬫煇浠诲姟鏃ュ織
```

## 4. 鐧藉悕鍗曚换鍔★紙褰撳墠鐗堟湰锛?
- 鐜锛歚env.start`銆乣env.stop`銆乣env.migrate_up`銆乣env.migrate_down`
- 鏁版嵁锛歚data.seed_overwrite`銆乣data.clear`
- 杩愯鏃讹細`runtime.start_backend`銆乣runtime.stop_backend`銆乣runtime.start_frontend`銆乣runtime.stop_frontend`銆乣runtime.start_ops_control`銆乣runtime.stop_ops_control`
- 鍘嬫祴锛歚perf.purchase_stress`銆乣perf.idempotency`銆乣perf.track_stress`銆乣perf.purchase_open`銆乣perf.track_open`

## 5. 鏃ュ織涓庤繍琛岃褰?
- 鏈嶅姟鏃ュ織锛歚log/services/*.log`锛坰tdout/stderr/pid锛?- 鍓嶇鏃ュ織锛歚log/frontends/*.log`
- 浠诲姟鏃ュ織锛歚log/ops-jobs/<job_id>/<yyyyMMddHH>.log`锛堟寜灏忔椂褰掓。锛?- 绉嶅瓙鏁版嵁缁撴灉锛歚log/data/seed-overwrite.result.json`

## 6. 杩滅▼璁块棶寤鸿锛堥潪鐢熶骇婕旂ず鐜锛?
- 浠呮毚闇插繀瑕佺鍙ｏ紙濡?`18080`锛夈€?- 浣跨敤鐜鍙橀噺瀵嗛挜鎺у埗璁块棶锛屼笉灏嗗瘑閽ュ啓鍏ヤ粨搴撱€?- 鑻ユ寕 Nginx锛屽澶栫粺涓€璧?HTTPS 涓庡弽鍚戜唬鐞嗐€?
