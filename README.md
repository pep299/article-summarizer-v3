# Article Summarizer v3

TypeScript脱却版：Go言語による記事要約システム

## 🎯 プロジェクト概要

URL要約をGemini APIで行い、Slack通知するシステムのGo言語移行版。
TypeScriptのライブラリアップデート等のメンテナンスコストを削減し、長期安定運用を目指したシステム。

## 🚀 移行の理由

- **メンテナンス性向上**: TypeScriptライブラリの頻繁なアップデートからの脱却
- **長期安定性**: Goの後方互換性による安定した運用
- **シンプルさ**: フロントエンドが不要な場面でのTypeScript依存の排除
- **デプロイの簡単さ**: 単一バイナリによる簡単なデプロイ

## 📋 移行手順

### Phase 1: Go実装（初期フェーズ）

#### 1.1 Go環境セットアップ
```bash
# Go開発環境確認
go version

# プロジェクト初期化
cd /Users/pepe/ghq/github.com/pep299/article-summarizer-v3
go mod init github.com/pep299/article-summarizer-v3

# 基本的なディレクトリ構成作成
mkdir -p {cmd,internal,pkg,configs,scripts,docs}
mkdir -p internal/{rss,cache,gemini,slack,handlers}
```

#### 1.2 既存実装の分析と設計
- [x] v1 (GAS版) の機能分析
- [x] v2 (TypeScript/Cloudflare Workers版) の機能分析
- [x] Go版アーキテクチャ設計
- [x] 依存ライブラリ選定

#### 1.3 コア機能実装
```bash
# 必要なライブラリインストール
go get github.com/gorilla/mux           # HTTPルーティング
go get github.com/joho/godotenv         # 環境変数管理
go get github.com/golang/glog           # ログ出力
go get github.com/stretchr/testify      # テスト
```

**実装順序:**
1. [x] 基本的なHTTPサーバー設定
2. [x] 環境変数・設定管理
3. [x] RSS取得・解析機能
4. [x] Gemini API連携
5. [x] Slack通知機能
6. [x] キャッシュ機能（メモリ）
7. [x] Webhook API実装
8. [x] 定期処理（CLI実行）

### Phase 2: 機能移植

#### 2.1 既存機能の完全移植
- [x] **RSS記事取得**: はてブ・Lobsters対応
- [x] **記事フィルタリング**: 重複排除・カテゴリ除外
- [x] **Gemini API要約**: フォールバック機能付き
- [x] **Slack通知**: チャンネル指定対応
- [x] **キャッシュ**: 重複チェック・統計取得
- [x] **Webhook API**: オンデマンド要約リクエスト
- [x] **定期処理**: RSS取得・要約の自動化

#### 2.2 Go特有の最適化
- [ ] Goroutineによる並行処理
- [ ] コネクションプールの活用
- [ ] メモリ効率の最適化
- [ ] エラーハンドリングの改善

### Phase 3: テスト・品質保証

#### 3.1 テスト実装
```bash
# テスト実行
go test ./...
go test -race ./...        # 競合状態チェック
go test -cover ./...       # カバレッジ確認
```

- [x] 単体テスト作成
- [ ] 統合テスト作成
- [ ] パフォーマンステスト
- [ ] 既存システムとの動作比較

#### 3.2 品質管理
```bash
# コード品質チェック
go vet ./...
go fmt ./...
golint ./...
```

### Phase 4: デプロイ・環境構築（決定済み）✅

#### 4.1 デプロイ先決定
**採用: Google Cloud Functions + Cloud Scheduler**
- **理由**: 月額ほぼ無料（0-30円）、設定最シンプル、540秒制限で十分
- **URL**: 固定URL取得可能
- **監視**: Cloud Logging/Monitoring無料枠内で十分

#### 4.2 技術スタック決定
- **実行環境**: Google Cloud Functions（Go 1.21 runtime）
- **定時実行**: Cloud Scheduler（cron設定）
- **認証**: Bearer Token（iOSショートカット対応）
- **キャッシュ**: Cloud Storage + JSON形式
- **インフラ管理**: Cloud Deployment Manager（YAML）
- **監視**: Google Cloud標準（無料枠内）

#### 4.3 実装変更点
- [x] ~~HTTPサーバー実装~~ → Cloud Functions用ラッパー
- [x] ~~SQLite/CSV~~ → JSON形式キャッシュ
- [x] ~~Docker~~ → ソースコード直接デプロイ
- [ ] Cloud Functions対応
- [ ] JSONキャッシュ実装
- [ ] Cloud Deployment Manager設定

#### 4.4 削除対象ファイル
- [ ] Dockerfile
- [ ] docker-compose.yml
- [ ] .dockerignore

### Phase 5: 本番移行

#### 5.1 段階的移行
- [ ] 開発環境での動作確認
- [ ] ステージング環境での統合テスト
- [ ] 本番環境での並行運用
- [ ] 既存システムからの完全移行

#### 5.2 運用監視
- [ ] ヘルスチェック実装
- [ ] エラー監視・アラート設定
- [ ] パフォーマンス監視
- [ ] ログ分析基盤

## 📁 プロジェクト構成

```
article-summarizer-v3/
├── cmd/
│   ├── server/          # HTTPサーバーメイン
│   └── cli/             # CLI版ツール
├── internal/            # 内部パッケージ
│   ├── config/         # 設定管理
│   ├── rss/            # RSS取得・解析
│   ├── cache/          # キャッシュ機能
│   ├── gemini/         # Gemini API連携
│   ├── slack/          # Slack通知
│   └── handlers/       # HTTPハンドラー
├── pkg/                # 外部パッケージ
├── configs/            # 設定ファイル
├── scripts/            # デプロイ・管理スクリプト
├── docs/               # ドキュメント
├── Dockerfile          # コンテナ設定
├── docker-compose.yml  # ローカル開発環境
├── go.mod              # Go依存関係
├── go.sum              # 依存関係チェックサム
├── Makefile            # ビルド・タスク管理
└── README.md           # このファイル
```

## 🔄 既存システムとの比較

| 項目 | v1 (GAS) | v2 (TypeScript/CF) | v3 (Go) |
|------|----------|-------------------|---------|
| **言語** | JavaScript | TypeScript | Go |
| **実行環境** | Google Apps Script | Cloudflare Workers | **Google Cloud Functions** |
| **メンテナンス性** | 低 | 中 | **高** |
| **ライブラリ更新** | 手動・困難 | 頻繁・自動化必要 | **最小限** |
| **パフォーマンス** | 低 | 高 | **最高** |
| **デプロイ複雑さ** | 簡単 | 中程度 | **シンプル** |
| **スケーラビリティ** | 限定的 | 高 | **高** |
| **運用コスト** | 低 | 中 | **月額0-30円** |

## 🛠️ 開発環境

### 必要なツール
- Go 1.21+
- Docker & Docker Compose
- Make
- Git

### ローカル開発
```bash
# 依存関係インストール
go mod download

# 開発サーバー起動
make dev

# テスト実行
make test

# ビルド
make build
```

## 📋 移行チェックリスト

### ✅ Phase 1: Go実装（優先）
- [ ] プロジェクト初期化
- [ ] 基本的なHTTPサーバー
- [ ] 環境変数管理
- [ ] ログ設定
- [ ] RSS取得・解析
- [ ] Gemini API連携
- [ ] Slack通知
- [ ] キャッシュ機能
- [ ] Webhook API
- [ ] 定期処理

### ✅ Phase 2: 環境構築・デプロイ（決定済み）
- [x] デプロイ先選定（Google Cloud Functions）
- [x] キャッシュ方式決定（Cloud Storage + JSON）
- [x] 認証方式決定（Bearer Token）
- [ ] Cloud Functions実装
- [ ] Cloud Deployment Manager設定
- [ ] モニタリング設定

### ⏳ Phase 3: 本番移行（最終）
- [ ] ステージング環境テスト
- [ ] パフォーマンステスト
- [ ] セキュリティ監査
- [ ] 本番デプロイ
- [ ] 既存システムからの移行

## 📖 参考資料

### 既存実装
- [v1 (GAS版)](../article-summarizer/) - 本番稼働中のGoogle Apps Script実装
- [v2 (TypeScript版)](../article-summarizer-v2/) - Cloudflare Workers + TypeScript実装

### Go関連リソース
- [Go公式ドキュメント](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

---

🚀 **Phase 4 完了！Google Cloud Functionsで安定したシステムを構築します！**

次のフェーズではCloud Functions実装とデプロイ設定を進めます。
