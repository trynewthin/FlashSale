# 甯歌闂涓庢帓闅?
## 1. 鍓嶇鐧诲綍鎺ュ彛 404

鐜拌薄锛?
- 娴忚鍣ㄨ姹?`http://127.0.0.1:5174/api/v1/admin/auth/login` 杩斿洖 404銆?
鎺掓煡锛?
1. 纭 `admin-gateway` 宸插惎鍔紙`http://127.0.0.1:8083/healthz`锛夈€?2. 纭鍓嶇 Vite 浠ｇ悊閰嶇疆鐢熸晥锛坄/api` 浠ｇ悊鍒?`8083` 鎴?`8082`锛夈€?3. 閲嶅惎鍓嶇寮€鍙戞湇鍔°€?
## 2. Docker 鍐呮湇鍔′簰閫氬け璐?
鐜拌薄锛?
- RPC 杩炴帴鎶ラ敊 `connection refused 127.0.0.1:*`銆?
鎺掓煡锛?
1. 瀹瑰櫒闂磋繛鎺ヤ笉鑳戒緷璧栧涓绘満 `127.0.0.1`銆?2. 妫€鏌ユ槸鍚﹀凡浣跨敤鐜鍙橀噺瑕嗙洊 RPC Target銆?3. 鍦?compose 缃戠粶鍐呭簲浣跨敤鏈嶅姟鍚嶆垨 host gateway銆?
## 3. 绉掓潃鍘嬫祴缃戠粶閿欒楂?
鐜拌薄锛?
- `network_errors` 鏄庢樉鍋忛珮銆?
鎺掓煡锛?
1. 鍏堥檷寮€鐜?`-rate`锛岀‘璁ょ郴缁熷湪绋冲畾鍖洪棿銆?2. 璋冩暣 HTTP 杩炴帴姹犲弬鏁帮細`max-idle-conns`銆乣max-idle-conns-per-host`銆乣max-conns-per-host`銆?3. 妫€鏌ュ悗绔槸鍚﹁Е鍙戦檺娴佹垨瓒呮椂锛堟煡鐪?`log/services`锛夈€?
## 4. 绠＄悊绔棤鏉冮檺锛?03锛?
鐜拌薄锛?
- 宸茬櫥褰曚絾鎺ュ彛杩斿洖 403銆?
鎺掓煡锛?
1. 妫€鏌ョ鐞嗗憳 `domains` 鏄惁鍖呭惈鐩爣鍩燂紙濡?`product_management`锛夈€?2. 妫€鏌?`data_scope` 鏄惁婊¤冻瑕佹眰锛堥儴鍒嗙鐞嗗啓鎺ュ彛瑕佹眰 `all`锛夈€?3. 鏉冮檺鍙樻洿鍚庨渶瑕侀噸鏂扮櫥褰曟垨鍒锋柊 token 鐢熸晥銆?
## 5. 杩佺Щ澶辫触

鐜拌薄锛?
- `migrate-up` 鎶ヨ繛鎺ユ垨鐗堟湰閿欒銆?
鎺掓煡锛?
1. `go run ./ops/cmd/fs env up` 纭繚 MySQL 宸插氨缁€?2. 妫€鏌?`configs/deploy.env` 鐨?MySQL 鐢ㄦ埛涓庡瘑鐮併€?3. 濡傝剰鐗堟湰锛屽厛璇勪及鏄惁闇€瑕?`migrate-down --all` 鍚庨噸寤恒€?
## 6. 蹇€熷仴搴锋鏌ュ懡浠?
```powershell
# 鍩虹杩為€氭€?go run ./ops/cmd/fs env smoke

# 缃戝叧鍋ュ悍
curl http://127.0.0.1:8082/healthz
curl http://127.0.0.1:8083/healthz

# 浠诲姟鍒楄〃
```

