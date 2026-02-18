# smoke-test.ps1 — FlashSale Docker 部署冒烟测试
#
# 使用场景：重建镜像并重新部署后，验证所有服务和 API 端点是否正常。
# 用法：powershell -ExecutionPolicy Bypass -File scripts/smoke-test.ps1
#       powershell -ExecutionPolicy Bypass -File scripts/smoke-test.ps1 -BaseURL http://10.0.0.5
#
# 注意：使用 127.0.0.1 而非 localhost，避免 Windows IPv6 解析超时。

param(
    [string]$BaseURL     = "http://127.0.0.1",
    [int]$NginxPort      = 18000,
    [int]$OpsPort        = 9100,
    [int]$TimeoutSec     = 8
)

$ErrorActionPreference = "Continue"

# ─── Helpers ───

$script:pass = 0
$script:fail = 0
$script:results = @()

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Url,
        [string]$Method = "GET",
        [string]$Body,
        [string]$ContentType = "application/json",
        [int]$ExpectedStatus = 200,
        [scriptblock]$Validate
    )

    $entry = @{ Name = $Name; Url = $Url; Status = ""; Result = "" }

    try {
        $params = @{
            Uri             = $Url
            Method          = $Method
            UseBasicParsing = $true
            TimeoutSec      = $TimeoutSec
        }
        if ($Body) {
            $params["Body"]        = $Body
            $params["ContentType"] = $ContentType
        }

        $resp = Invoke-WebRequest @params
        $entry.Status = $resp.StatusCode

        if ($resp.StatusCode -ne $ExpectedStatus) {
            $entry.Result = "FAIL (expected $ExpectedStatus, got $($resp.StatusCode))"
            $script:fail++
            Write-Host "  FAIL  $Name — HTTP $($resp.StatusCode) (expected $ExpectedStatus)" -ForegroundColor Red
        } elseif ($Validate) {
            $detail = & $Validate $resp
            if ($detail) {
                $entry.Result = "PASS ($detail)"
                $script:pass++
                Write-Host "  PASS  $Name — $detail" -ForegroundColor Green
            } else {
                $entry.Result = "PASS"
                $script:pass++
                Write-Host "  PASS  $Name" -ForegroundColor Green
            }
        } else {
            $entry.Result = "PASS"
            $script:pass++
            Write-Host "  PASS  $Name" -ForegroundColor Green
        }
    } catch {
        $msg = $_.Exception.Message
        # Invoke-WebRequest throws on 4xx/5xx — extract status if possible
        if ($_.Exception.Response) {
            $code = [int]$_.Exception.Response.StatusCode
            $entry.Status = $code
            if ($code -eq $ExpectedStatus) {
                $entry.Result = "PASS (expected $ExpectedStatus)"
                $script:pass++
                Write-Host "  PASS  $Name — HTTP $code (expected)" -ForegroundColor Green
            } else {
                $entry.Result = "FAIL (HTTP $code): $msg"
                $script:fail++
                Write-Host "  FAIL  $Name — HTTP $code : $msg" -ForegroundColor Red
            }
        } else {
            $entry.Status = "ERR"
            $entry.Result = "FAIL: $msg"
            $script:fail++
            Write-Host "  FAIL  $Name — $msg" -ForegroundColor Red
        }
    }

    $script:results += $entry
}

# ─── 开始测试 ───

$nginx = "${BaseURL}:${NginxPort}"
$ops   = "${BaseURL}:${OpsPort}"

Write-Host ""
Write-Host "═══════════════════════════════════════════" -ForegroundColor Cyan
Write-Host " FlashSale Smoke Test (Docker Deploy Mode)" -ForegroundColor Cyan
$infoLine = " Nginx: $nginx  --  Ops: $ops"
Write-Host $infoLine -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

# ──────────────────────────────
# Section 1: Infrastructure
# ──────────────────────────────
Write-Host "── [1] Infrastructure Health ──" -ForegroundColor Yellow

Test-Endpoint "Nginx healthz" "$nginx/nginx-healthz"
Test-Endpoint "User Gateway healthz" "$nginx/healthz"
Test-Endpoint "Admin Gateway healthz" "$nginx/admin-healthz"
Test-Endpoint "Ops Control healthz" "$ops/healthz"

# ──────────────────────────────
# Section 2: Ops Control APIs
# ──────────────────────────────
Write-Host ""
Write-Host "── [2] Ops Control APIs ──" -ForegroundColor Yellow

Test-Endpoint "Ops: status" "$ops/api/v1/status" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data
    "mode=$($d.deployment_mode)"
}

Test-Endpoint "Ops: containers/status" "$ops/api/v1/containers/status" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data.status
    "services=$($d.services.Count), containers=$($d.containers.Count)"
}

Test-Endpoint "Ops: tasks" "$ops/api/v1/tasks" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data.tasks
    "$($d.Count) tasks"
}

Test-Endpoint "Ops: etcd/services" "$ops/api/v1/etcd/services" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data.registry
    "available=$($d.available), services=$($d.services.Count)"
}

Test-Endpoint "Ops: metrics/catalog" "$ops/api/v1/metrics/catalog" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data.metrics
    "$($d.Count) metrics"
}

Test-Endpoint "Ops: observability/links" "$ops/api/v1/observability/links" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data.links
    "jaeger=$($d.jaeger.available), prometheus=$($d.prometheus.available)"
}

Test-Endpoint "Ops: jobs list" "$ops/api/v1/jobs"

Test-Endpoint "Ops: samples/days" "$ops/api/v1/samples/days"

Test-Endpoint "Ops: service-logs/files" "$ops/api/v1/service-logs/files"

# ──────────────────────────────
# Section 3: Business APIs (via Nginx)
# ──────────────────────────────
Write-Host ""
Write-Host "── [3] Business APIs ──" -ForegroundColor Yellow

Test-Endpoint "Products: list" "$nginx/api/v1/products" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data
    if ($d.PSObject.Properties["products"]) { "$($d.products.Count) products" } else { "ok" }
}

Test-Endpoint "Seckill: activities" "$nginx/api/v1/seckill/activities" -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data
    if ($d.PSObject.Properties["activities"]) { "$($d.activities.Count) activities" } else { "ok" }
}

# 注册 + 登录流程
$ts = [long]([DateTimeOffset]::Now.ToUnixTimeMilliseconds())
$phone = "138${ts}".Substring(0, 11)
$origPassword = "SmokeTest123!"
$newPassword  = "NewSmoke456!"
$regBody = @{ phone = $phone; password = $origPassword; nickname = "smoke_$ts" } | ConvertTo-Json

Test-Endpoint "User: register" "$nginx/api/v1/user/register" -Method POST -Body $regBody -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data
    if ($d.PSObject.Properties["user_id"]) { "userId=$($d.user_id)" } else { "ok" }
}

$loginBody = @{ phone = $phone; password = $origPassword } | ConvertTo-Json
$script:userToken = $null
$script:userId = $null

Test-Endpoint "User: login" "$nginx/api/v1/user/login" -Method POST -Body $loginBody -Validate {
    param($r)
    $d = ($r.Content | ConvertFrom-Json).data
    if ($d.PSObject.Properties["access_token"]) {
        $script:userToken = $d.access_token
        if ($d.PSObject.Properties["user_id"]) { $script:userId = $d.user_id }
        "token=YES"
    } else { "no token" }
}

# 401 测试（无 token 访问受保护接口）
Test-Endpoint "User: profile (no auth, expect 401)" "$nginx/api/v1/user/profile" -ExpectedStatus 401

# ──────────────────────────────
# Section 4: Authenticated User APIs
# ──────────────────────────────
Write-Host ""
Write-Host "── [4] Authenticated User APIs ──" -ForegroundColor Yellow

if ($script:userToken) {
    # 因为 Test-Endpoint 不支持自定义 header，直接用 Invoke-WebRequest 测试
    $authHeaders = @{ Authorization = "Bearer $($script:userToken)" }


    # 4a. Profile with token
    $entry = @{ Name = "User: profile (with auth)"; Url = ""; Status = ""; Result = "" }
    try {
        $resp = Invoke-WebRequest -Uri "$nginx/api/v1/user/profile" -Headers $authHeaders -UseBasicParsing -TimeoutSec $TimeoutSec
        $d = ($resp.Content | ConvertFrom-Json).data
        $entry.Status = $resp.StatusCode
        $entry.Result = "PASS (phone=$($d.phone))"
        $script:pass++
        Write-Host "  PASS  User: profile (with auth) — phone=$($d.phone)" -ForegroundColor Green
    } catch {
        $entry.Status = "ERR"
        $entry.Result = "FAIL: $($_.Exception.Message)"
        $script:fail++
        Write-Host "  FAIL  User: profile (with auth) — $($_.Exception.Message)" -ForegroundColor Red
    }
    $script:results += $entry

    # 4b. ChangePassword
    $changePwBody = @{ old_password = $origPassword; new_password = $newPassword } | ConvertTo-Json
    $entry = @{ Name = "User: change password"; Url = ""; Status = ""; Result = "" }
    try {
        $resp = Invoke-WebRequest -Uri "$nginx/api/v1/user/password" -Method PUT -Body $changePwBody -ContentType "application/json" -Headers $authHeaders -UseBasicParsing -TimeoutSec $TimeoutSec
        $d = ($resp.Content | ConvertFrom-Json)
        if ($d.code -eq "OK") {
            $entry.Status = $resp.StatusCode
            $entry.Result = "PASS"
            $script:pass++
            Write-Host "  PASS  User: change password" -ForegroundColor Green
        } else {
            $entry.Status = $resp.StatusCode
            $entry.Result = "FAIL (code=$($d.code))"
            $script:fail++
            Write-Host "  FAIL  User: change password — code=$($d.code)" -ForegroundColor Red
        }
    } catch {
        $entry.Status = "ERR"
        $entry.Result = "FAIL: $($_.Exception.Message)"
        $script:fail++
        Write-Host "  FAIL  User: change password — $($_.Exception.Message)" -ForegroundColor Red
    }
    $script:results += $entry

    # 4c. 用新密码登录验证
    $newLoginBody = @{ phone = $phone; password = $newPassword } | ConvertTo-Json
    Test-Endpoint "User: login with new password" "$nginx/api/v1/user/login" -Method POST -Body $newLoginBody -Validate {
        param($r)
        $d = ($r.Content | ConvertFrom-Json).data
        if ($d.PSObject.Properties["access_token"]) { "token=YES" } else { "no token" }
    }

    # 4d. 旧密码应该登录失败
    $oldLoginBody = @{ phone = $phone; password = $origPassword } | ConvertTo-Json
    Test-Endpoint "User: login with old password (expect 401)" "$nginx/api/v1/user/login" -Method POST -Body $oldLoginBody -ExpectedStatus 401

} else {
    Write-Host "  SKIP  Section 4: no user token captured" -ForegroundColor DarkYellow
}

# ──────────────────────────────
# Section 5: Admin User Management APIs (requires seed data)
# ──────────────────────────────
Write-Host ""
Write-Host "── [5] Admin User Management ──" -ForegroundColor Yellow

# 尝试 admin 登录（使用 seed 默认凭据）
$script:adminToken = $null
$adminLoginBody = @{ username = "admin_root"; password = "Admin12345" } | ConvertTo-Json
try {
    $resp = Invoke-WebRequest -Uri "$nginx/api/v1/admin/auth/login" -Method POST -Body $adminLoginBody -ContentType "application/json" -UseBasicParsing -TimeoutSec $TimeoutSec
    $d = ($resp.Content | ConvertFrom-Json).data
    if ($d.PSObject.Properties["access_token"]) {
        $script:adminToken = $d.access_token
        $script:pass++
        Write-Host "  PASS  Admin: login (seed account)" -ForegroundColor Green
        $script:results += @{ Name = "Admin: login"; Status = $resp.StatusCode; Result = "PASS" }
    } else {
        Write-Host "  SKIP  Admin: login — no token in response, skipping admin tests" -ForegroundColor DarkYellow
        $script:results += @{ Name = "Admin: login"; Status = $resp.StatusCode; Result = "SKIP (no token)" }
    }
} catch {
    Write-Host "  SKIP  Admin: login — seed account not available, skipping admin tests" -ForegroundColor DarkYellow
    $script:results += @{ Name = "Admin: login"; Status = "SKIP"; Result = "SKIP (no seed)" }
}

if ($script:adminToken) {
    $adminAuth = @{ Authorization = "Bearer $($script:adminToken)" }

    # 5a. ListUsers
    $entry = @{ Name = "Admin: list users"; Url = ""; Status = ""; Result = "" }
    try {
        $resp = Invoke-WebRequest -Uri "$nginx/api/v1/admin/users?page=1&page_size=10" -Headers $adminAuth -UseBasicParsing -TimeoutSec $TimeoutSec
        $d = ($resp.Content | ConvertFrom-Json).data
        $total = if ($d.PSObject.Properties["total"]) { $d.total } else { "?" }
        $count = if ($d.PSObject.Properties["list"]) { $d.list.Count } else { "?" }
        $entry.Status = $resp.StatusCode
        $entry.Result = "PASS (total=$total, page_count=$count)"
        $script:pass++
        Write-Host "  PASS  Admin: list users — total=$total, page_count=$count" -ForegroundColor Green
    } catch {
        $entry.Status = "ERR"
        $entry.Result = "FAIL: $($_.Exception.Message)"
        $script:fail++
        Write-Host "  FAIL  Admin: list users — $($_.Exception.Message)" -ForegroundColor Red
    }
    $script:results += $entry

    # 5b. ResetUserPassword (需要上面注册的用户的 userId)
    if ($script:userId) {
        $resetBody = @{ new_password = "ResetByAdmin99!" } | ConvertTo-Json
        $entry = @{ Name = "Admin: reset user password"; Url = ""; Status = ""; Result = "" }
        try {
            $resp = Invoke-WebRequest -Uri "$nginx/api/v1/admin/users/$($script:userId)/reset-password" -Method POST -Body $resetBody -ContentType "application/json" -Headers $adminAuth -UseBasicParsing -TimeoutSec $TimeoutSec
            $d = ($resp.Content | ConvertFrom-Json)
            if ($d.code -eq "OK") {
                $entry.Status = $resp.StatusCode
                $entry.Result = "PASS"
                $script:pass++
                Write-Host "  PASS  Admin: reset user password" -ForegroundColor Green
            } else {
                $entry.Status = $resp.StatusCode
                $entry.Result = "FAIL (code=$($d.code))"
                $script:fail++
                Write-Host "  FAIL  Admin: reset user password — code=$($d.code)" -ForegroundColor Red
            }
        } catch {
            $entry.Status = "ERR"
            $entry.Result = "FAIL: $($_.Exception.Message)"
            $script:fail++
            Write-Host "  FAIL  Admin: reset user password — $($_.Exception.Message)" -ForegroundColor Red
        }
        $script:results += $entry

        # 5c. 用管理员重置后的密码登录验证
        $resetLoginBody = @{ phone = $phone; password = "ResetByAdmin99!" } | ConvertTo-Json
        Test-Endpoint "User: login with admin-reset password" "$nginx/api/v1/user/login" -Method POST -Body $resetLoginBody -Validate {
            param($r)
            $d = ($r.Content | ConvertFrom-Json).data
            if ($d.PSObject.Properties["access_token"]) { "token=YES" } else { "no token" }
        }
    } else {
        Write-Host "  SKIP  Admin: reset user password — no userId captured" -ForegroundColor DarkYellow
    }

    # 5d. ListUsers 无权限 (无 token 应返回 401)
    Test-Endpoint "Admin: list users (no auth, expect 401)" "$nginx/api/v1/admin/users" -ExpectedStatus 401

} else {
    Write-Host "  SKIP  Section 5: no admin token (run 'fs data seed-overwrite --force' first)" -ForegroundColor DarkYellow
}

# ──────────────────────────────
# Summary
# ──────────────────────────────
Write-Host ""
Write-Host "═══════════════════════════════════════════" -ForegroundColor Cyan
$total = $script:pass + $script:fail
$passCount = $script:pass
$failCount = $script:fail
if ($failCount -eq 0) {
    Write-Host " ALL PASSED: $passCount / $total tests" -ForegroundColor Green
} else {
    Write-Host " RESULT: $passCount passed, $failCount failed / $total total" -ForegroundColor Red
}
Write-Host "═══════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

# 打印失败项
if ($script:fail -gt 0) {
    Write-Host "Failed tests:" -ForegroundColor Red
    foreach ($r in $script:results) {
        if ($r.Result -like "FAIL*") {
            Write-Host "  - $($r.Name): $($r.Result)" -ForegroundColor Red
        }
    }
    Write-Host ""
    exit 1
}
