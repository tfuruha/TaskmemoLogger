# Project Guidelines (SSoT)

このファイルはプロジェクト固有のガイドラインを記述する場所です。
技術的な決定事項、アーキテクチャ、コマンドなどをここに追記してください。

## 参照先
- **行動規範:** `.agent/rules/` を参照してください。
- **ワークフロー:** `.agent/workflows/` を参照してください。
- **スキル:** `.agent/skills/` を参照してください。

## Architecture & Frameworks
- **バックエンド:** Go 1.23+
- **フロントエンド:** Vanilla HTML / CSS / JS (フレームワークなし)
- **ビルド・パッケージング:** Wails v2

## Setup & Build Commands
```bash
# 開発モード（ホットリロード）
wails dev

# ユニットテスト実行
go test ./... -v

# リリースビルド
wails build
```

## Code style
- Google Style Guideを基本のコーディングスタイルとして参照してください。
  - Go言語のコードについては、Google Go Style Guideを推奨します。
  - フロントエンド(HTML/CSS/JS)についても、Google HTML/CSS Style Guide や Google JavaScript Style Guideを参考にしてください。
- **※注意:** 既存のコードに対する上記スタイルの適用は任意です。無理なリファクタリングは不要です。