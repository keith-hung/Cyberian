# wedaka-cli

WeDaka 出勤系統的命令列客戶端,透過 REST API 進行上下班打卡與出勤查詢。

本工具屬於 [Cyberian](../README.md) monorepo。一般使用者通常透過 `wedaka`
Claude Code 技能操作,而非直接執行;完整用法與範例請見
[skills/wedaka/SKILL.md](../skills/wedaka/SKILL.md)。

## 建置

```bash
cd wedaka-cli
go build -o wedaka .
```

需要 Go 1.25,建置產物已列入 `.gitignore`。

## 設定

| 變數 | 必填 | 說明 |
|------|------|------|
| `WEDAKA_API_URL` | 是 | WeDaka API URL |
| `WEDAKA_USERNAME` | 是 | 使用者帳號 |
| `WEDAKA_EMP_NO` | 是 | 員工編號 |
| `WEDAKA_DEVICE_ID` | 是 | 裝置 ID |

每個變數都有對應的全域旗標:`--url`、`--username`、`--emp-no`、`--device-id`,
另有 `--pretty`。旗標優先於環境變數。

## 指令

| 指令 | 用途 |
|------|------|
| `clock-in` | 上班打卡 |
| `clock-out` | 下班打卡 |
| `timelog` | 查詢打卡紀錄 |
| `check-workday` | 檢查指定日期是否為工作日 |
| `version` | 印出版本 / commit / 建置日期 |

各指令的旗標請以 `--help` 查看。

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
