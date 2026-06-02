# timecard-cli(已退役)

> **已退役:** 此 CLI 對應舊版 TimeCard 後端,因會與重建後的系統反覆誤觸發,已自
> plugin 移除。原始碼保留供參考,但**未**作為技能曝露、**未**由 `release.yml` /
> `build.sh` 建置、也**沒有** launcher 指令稿。工時相關作業請改用
> [nouveau-timecard-cli](../nouveau-timecard-cli/README.md)。

舊版 TimeCard(Java Web 應用)的命令列客戶端,透過解析 HTML 回應進行工時表操作。
背景說明見 [Cyberian](../README.md) monorepo 的「進階資訊」段落。

## 建置

僅供參考用途,仍可手動建置:

```bash
cd timecard-cli
go build -o timecard .
```

需要 Go 1.25,建置產物已列入 `.gitignore`。

## 設定

| 變數 | 必填 | 說明 |
|------|------|------|
| `TIMECARD_BASE_URL` | 是 | TimeCard 基底 URL |
| `TIMECARD_USERNAME` | 是 | 使用者帳號 |
| `TIMECARD_PASSWORD` | 是* | 密碼(*或以 `--pass-stdin` 傳入) |

每個變數都有對應的全域旗標:`--url`、`--user`、`--pass-stdin`,另有
`--session-file`(預設 `.timecard-session.json`)與 `--pretty`。旗標優先於環境變數。

## 指令

| 指令 | 用途 |
|------|------|
| `projects` | 列出可填報的專案 |
| `activities` | 列出活動 |
| `timesheet` | 顯示週工時表 |
| `summary` | 顯示工時彙總 |
| `save` | 將工時存為草稿(不送審) |
| `version` | 印出版本 / commit / 建置日期 |

各指令的旗標請以 `--help` 查看。

## 後端限制(此舊版專屬)

- 備註欄禁用字元:`#$%^&*=+{}[]|?'"`
- entry index 為 0-9(每週最多 10 筆)
- 每日工時不可超過 8 小時
- 僅支援草稿儲存,絕不送審

> 這些限制是舊版 TimeCard 後端專屬。重建後的系統改用自身的驗證規則,詳見
> [nouveau-timecard-cli](../nouveau-timecard-cli/README.md)。

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
