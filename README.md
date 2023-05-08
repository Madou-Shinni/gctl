### 代码生成器
go install github.com/Madou-Shinni/gctl@latest
```go
gctl -m Article 自动生成代码
生成代码如下
internal/domain/article.go
internal/service/article.go
internal/data/article.go
api/routers/article.go
api/handle/article.go

GLOBAL OPTIONS:
--module value, -m value  生成模块的名称
--help, -h                show help
```