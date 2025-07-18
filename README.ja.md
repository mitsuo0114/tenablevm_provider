# Tenable 脆弱性管理 Terraform Provider

[English version](README.md)

このリポジトリには、**Tenable Vulnerability Management (Tenable VM)** API と連携する Terraform Provider が含まれています。実装には [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) を利用した Go を使用しています。

この Provider では Tenable VM のユーザーを管理するリソースと、ユーザー・ロール・グループを取得するデータソースを提供します。あくまでローカルでの検証を想定したサンプル実装であり、公式プロダクトではありません。

## 必要環境

- Go 1.23 以降
- Terraform 1.5 以降

## ビルド方法

リポジトリをクローン後、次のコマンドでバイナリをビルドします。

```bash
go build -o terraform-provider-tenablevm
```

生成されたバイナリは通常 `~/.terraform.d/plugins/registry.terraform.io/tenable/tenablevm/<version>/` に配置します。開発時は任意のバージョン文字列で構いません。

## 初期設定

Provider では API アクセスに必要な認証情報を指定します。設定ブロックに直接記述するか環境変数で指定できます。

| 設定属性 | 環境変数 | 説明 |
|----------|----------|------|
| `access_key` | `TENABLE_ACCESS_KEY` | API のアクセスキー |
| `secret_key` | `TENABLE_SECRET_KEY` | API のシークレットキー (機密情報) |

`access_key` と `secret_key` の 2 つは必須です。

## Terraform での利用例

```hcl
terraform {
  required_providers {
    tenablevm = {
      source  = "registry.terraform.io/tenable/tenablevm"
      version = "0.1.0"
    }
  }
}

provider "tenablevm" {
  access_key = var.access_key
  secret_key = var.secret_key
}
```

### ユーザー管理

現在実装されているリソースは `tenablevm_user` のみです。簡単な例を以下に示します。

```hcl
resource "tenablevm_user" "example" {
  username    = "terraform-user"
  password    = "initialPassword123!"
  permissions = 16
  name        = "Terraform Example"
  email       = "tf@example.com"
  enabled     = true
}
```

その他の属性についてはソースコード内のスキーマ定義を参照してください。

### データソース

- `tenablevm_user` – ID またはユーザー名でユーザーを取得
- `tenablevm_role` – ロール情報を取得
- `tenablevm_group` – グループ情報を取得

例:

```hcl
data "tenablevm_user" "current" {
  username = "terraform-user"
}
```

## テスト実行

```bash
go test ./...
```

