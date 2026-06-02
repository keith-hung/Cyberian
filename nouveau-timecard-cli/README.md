# nouveau-timecard-cli

重建版**智慧工時系統**(Smart Timecard,ASP.NET Core Razor Pages)的命令列客戶端。
僅支援草稿儲存 —— 刻意不提供送審指令。

本工具屬於 [Cyberian](../README.md) monorepo。一般使用者通常透過 `nouveau-timecard`
Claude Code 技能操作,而非直接執行;完整用法、範例與欄位說明請見
[skills/nouveau-timecard/SKILL.md](../skills/nouveau-timecard/SKILL.md)。

## 建置

```bash
cd nouveau-timecard-cli
go build -o nouveau-timecard .
```

需要 Go 1.25,建置產物已列入 `.gitignore`。

## 設定

沿用 timecard 系列環境變數(與舊版 `timecard-cli` 共用):

| 變數 | 必填 | 說明 |
|------|------|------|
| `TIMECARD_BASE_URL` | 是 | 伺服器基底 URL |
| `TIMECARD_USERNAME` | 是 | LDAP 帳號 |
| `TIMECARD_PASSWORD` | 是* | 密碼(*或以 `--pass-stdin` 傳入) |
| `TIMECARD_INSECURE` | 否 | 略過 TLS 驗證(僅供開發) |

每個變數都有對應的全域旗標:`--url`、`--user`、`--pass-stdin`、`--insecure`,
另有 `--session-file`(預設 `.nouveau-timecard-session.json`)與 `--pretty`。
旗標優先於環境變數。

密碼不接受以明碼旗標傳入,請改用管線:

```bash
echo "$PASSWORD" | ./nouveau-timecard timesheet --pass-stdin --year 2026 --month 6
```

## 指令

| 指令 | 用途 |
|------|------|
| `projects` | 列出該月份可填報的專案 |
| `activities` | 列出某專案的活動(`--project`) |
| `timesheet` | 顯示該月份的草稿工時 |
| `save` | 將紀錄存為草稿(`--records '<json>'`) |
| `sync-leave` | 依 BPM 請假資料填入「休假」活動 |
| `version` | 印出版本 / commit / 建置日期 |

月份以 `--date YYYY-MM-DD` 或 `--year` + `--month`(兩者須同時提供)指定,未指定時
預設為當月。各指令的旗標請以 `--help` 查看。

## 輸出與離開碼

所有輸出皆為可機器解析的 JSON:結果寫至 stdout,錯誤以
`{"success":false,"error":"..."}` 寫至 stderr。

| 離開碼 | 意義 |
|--------|------|
| 0 | 成功 |
| 1 | 一般錯誤 |
| 2 | 認證 / 設定 |
| 3 | 驗證 |
| 4 | 網路 |
