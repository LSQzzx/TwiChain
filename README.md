# Social System based on Blockchain
# 基于区块链的社交系统

This is a social platform system based on blockchain technology, supporting functions such as posting, commenting, and liking.

这是一个基于区块链技术的社交平台系统，支持发帖、评论、点赞等社交功能。

## Project Structure

```
twichain/
├── cmd/
│   └── main.go
├── configs/
│   └── config.yaml
├── internal/
│   ├── blockchain/
│   ├── config/
│   ├── crypto/
│   ├── network/
│   └── storage/
└── test/
```

## Quick Start

### 1. Requirements

#### Production
- Go 1.23.3+
#### Development
- Go 1.23.3+
- Python 3.9+ （for test）
- SQLite3

### 2. Installation

```bash
go mod tidy
// go build -o twichain cmd/main.go
```

### 3. Run Server

```bash
go run cmd/main.go
// ./twichain -config configs/config.yaml
```

The default server will start at http://localhost:8080

## API

### 1. new transaction

```http
POST /transactions/new
Content-Type: application/json

{
    "sender": "Sender's Public Key (256-bit hexadecimal)",
    "receiver": "Recipient's Public Key (256-bit hexadecimal)",
    "message": "Message content",
    "signature": "EdDSA Signature",
    "is_like": false,
    "target_post_id": "Target post transaction ID when liking or commenting"
}
```

### 2. get block chain

Return complete blockchain data

```http
GET /chain
```

### 3. register a node

register a node on an exsit network

```http
POST /nodes/register
Content-Type: application/json

{
    "node": "localhost:8080"
}
```

### 4. broadcast a node

For broadcasting a node

```http
POST /nodes/new
Content-Type: application/json

{
    "node": "localhost:8080"
}
```

### 5. broadcast a block

For broadcasting a block

```http
POST /block/receive
Content-Type: application/json

{
    "node": "localhost:8080"
}
```

## Signature Verification

The system uses Ed25519 for signature verification:

1. Generate a key pair：
```python
# python example
private_key_bytes = os.urandom(32)
private_key = ed25519.SigningKey(private_key_bytes)
public_key = private_key.get_verifying_key()
```

2. 签名消息：
- Normal posting/commenting: Use message content signature
- Like: Use target post ID signature

## Configuration Instructions

config.yaml:
```yaml
server:
  port: 8080

database:
  path: "data/blockchain.db"

blockchain:
  difficulty: 2
  node_address: ""
```

## Test

```bash
go build -o test/twichain cmd/main.go
cd test
uv init .
uv add requests ed25519 tqdm
uv run consensus_test.py
```

## Example **(VERY IMPORTANT!!!)**

1. Post in the main_space：

**receiver for main_space is a fixed value**

```python
tx_data = {
    "sender": "User Public Key",
    "receiver": "69c5f684026e6bd3e2a8f175a892ca6858cb9936b3c525ce11b981f848a69fc2",
    "message": "This is a test post",
    "is_like": false
    "target_post_id": ""
}
```

2. Like post：
```python
tx_data = {
    "sender": "User Public Key",
    "receiver": "Post Author's Public Key",
    "message": "",
    "is_like": true,
    "target_post_id": "Transaction ID of Target Post"
}
```

3. Comment post：
```python
tx_data = {
    "sender": "User Public Key",
    "receiver": "Post Author's Public Key",
    "message": "This is a comment",
    "is_like": false,
    "target_post_id": "Transaction ID of Target Post"
}
```

## Contribution Guide

Welcome to submit a Pull Request or raise an Issue!

## License

This project is open-sourced under the MIT License, see the LICENSE file for details.