# 變更日誌

此檔案記錄本專案各版本的重要變更。

## [0.2.4] - 2026-03-11

### 新增

- azuredevops-cli 新增 `pr-comments` 命令 — 列出 PR 的所有留言 thread，包含作者、內容、日期與 thread 狀態，自動過濾系統產生的 thread 與已刪除的留言

### 修正

- `CommentType` 欄位型別從 `int` 改為 `string`，與 Azure DevOps REST API 回傳格式一致（影響 `pr-comment` 建立留言的 request body）

## [0.2.3] - 2026-03-11

### 新增

- azuredevops-cli 新增 `pr-attachment` 命令 — 上傳圖片至 PR 附件並回傳可嵌入 Markdown 的絕對 URL
- jira skill 新增 custom field 操作 — 透過 REST API 支援 list、get、set、inspect schema，涵蓋 user/text/select 欄位類型

## [0.2.2] - 2026-03-06

### 變更

- outlook-calendar skill 支援多個行事曆 — 環境變數改為 `OUTLOOK_ICS_URLS`（逗號分隔多個 ICS URL），自動從 `X-WR-CALNAME` 取得行事曆名稱，新增 `--calendar` 篩選參數
- `OUTLOOK_ICS_URL`（單數）已棄用，CLI 偵測到舊環境變數時會提示遷移

### 修正

- 所有 ICS URL 取得失敗時，改為 exit code 4 並輸出明確錯誤訊息（原本靜默顯示「No events found」）

## [0.2.1] - 2026-03-04

### 新增

- azuredevops-cli 新增 `my-prs` 命令 — 跨專案與 repository 查詢目前使用者相關的 pull request（包含作為建立者或 reviewer 的 PR）

## [0.2.0] - 2026-03-04

### 新增

- **azuredevops-cli** — 全新 Go CLI，支援 Azure DevOps Server（地端）的 IIS Basic Auth，提供 12 個命令：`projects`、`repos`、`prs`、`pr`、`pr-create`、`pr-update`、`pr-approve`、`pr-reject`、`pr-comment`、`pr-reviewers`、`pr-add-reviewer`、`version`
- azuredevops skill 定義檔（`skills/azuredevops/SKILL.md`）
- azuredevops-cli launcher 腳本（`.sh` 與 `.ps1`）

## [0.1.0] - 2026-03-03

### 新增

- **timecard-cli** — Go CLI，透過 HTML scraping 管理 TimeCard 工時表，支援命令：`projects`、`activities`、`timesheet`、`summary`、`save`、`version`
- **wedaka-cli** — Go CLI，透過 REST API 管理 WeDaka 出勤系統，支援命令：`clock-in`、`clock-out`、`timelog`、`check-workday`、`version`
- Claude Code plugin，包含四個 skills：timecard、wedaka、jira、outlook-calendar
- JIRA skill，含自動下載 `jira-cli` binary 的 launcher 腳本
- Outlook calendar skill，含 ICS 訂閱解析器
- Marketplace manifest（`.claude-plugin/marketplace.json`），支援透過 `claude plugin marketplace add` 安裝
- PowerShell launcher 腳本，支援 Windows
- 跨平台 build 腳本與 GitHub Actions release workflow
- MIT 授權條款

### 變更

- 將 `skills/` 與 `scripts/` 從 `.claude-plugin/` 移至 repository 根目錄，符合官方 plugin 結構規範
- 簡化 `plugin.json`，改用 auto-discovery 取代明確指定 skill 路徑
- 重寫 README，以 plugin 安裝與設定流程為開頭

### 修正

- 修正 marketplace 與 plugin 同名導致 Linux 上安裝靜默失敗的問題
- 修正 jira-cli tar 解壓縮路徑錯誤
