# azuredevops-cli

Azure DevOps Server(地端、IIS Basic Auth)的命令列客戶端,透過 REST API 管理
專案、儲存庫與 pull request。

本工具屬於 [Cyberian](../README.md) monorepo。一般使用者通常透過 `azuredevops`
Claude Code 技能操作,而非直接執行;完整用法與範例請見
[skills/azuredevops/SKILL.md](../skills/azuredevops/SKILL.md)。

## 建置

```bash
cd azuredevops-cli
go build -o azuredevops .
```

需要 Go 1.25,建置產物已列入 `.gitignore`。

## 設定

| 變數 | 必填 | 說明 |
|------|------|------|
| `AZDO_BASE_URL` | 是 | Azure DevOps Server 基底 URL |
| `AZDO_COLLECTION` | 是 | Collection 名稱 |
| `AZDO_USERNAME` | 是 | 使用者帳號 |
| `AZDO_PASSWORD` | 是* | 密碼(*或以 `--pass-stdin` 傳入) |
| `AZDO_DOMAIN` | 否 | AD 網域名稱 |
| `AZDO_PROJECT` | 否 | 預設專案名稱 |
| `AZDO_REPO` | 否 | 預設儲存庫名稱 |
| `AZDO_API_VERSION` | 否 | API 版本(預設 `5.0-preview.1`) |
| `AZDO_INSECURE` | 否 | 略過 TLS 憑證驗證(僅供開發) |

每個變數都有對應的全域旗標:`--base-url`、`--collection`、`--username`、
`--pass-stdin`、`--domain`、`--project`、`--repo`、`--api-version`、`--insecure`,
另有 `--pretty`。旗標優先於環境變數。

密碼不接受以明碼旗標傳入,請改用管線:

```bash
echo "$PASSWORD" | ./azuredevops my-prs --pass-stdin
```

## 指令

| 指令 | 用途 |
|------|------|
| `projects` | 列出 collection 內的專案 |
| `repos` | 列出某專案的儲存庫 |
| `prs` | 列出儲存庫的 pull request |
| `my-prs` | 列出與自己相關的 PR |
| `pr` | 顯示單一 PR 詳情 |
| `pr-create` | 建立 PR |
| `pr-update` | 更新 PR |
| `pr-approve` | 核可 PR |
| `pr-reject` | 否決 PR |
| `pr-comment` | 對 PR 留言或回覆既有 thread |
| `pr-comments` | 列出 PR 的留言 |
| `pr-attachment` | 上傳並在 PR 留言中嵌入圖片 |
| `pr-reviewers` | 列出 PR 的審查者 |
| `pr-add-reviewer` | 新增 PR 審查者 |
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
