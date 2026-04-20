# AGENTS.md - minibp 项目指南

## 项目概述

minibp 是一个用 Go 编写的最小化 Android.bp (Blueprint) 解析器和 Ninja 构建文件生成器。它解析 `.bp` 文件，解析依赖关系，处理架构变体，并输出 `build.ninja` 文件。

## 技术栈

- **语言**: Go 1.21+
- **无外部依赖**: 仅使用 Go 标准库
- **构建系统**: Go 原生 (`go build`, `go test`)
- **目标输出**: Ninja 构建文件格式

## 项目结构

```
minibp/
├── cmd/minibp/          # CLI 入口点
│   ├── main.go          # 主程序：参数解析、文件加载、依赖图构建、Ninja 生成
│   └── main_test.go     # 集成测试（glob 展开、属性合并、拓扑排序等）
├── parser/              # Blueprint 解析器
│   ├── ast.go           # AST 定义（Module, Map, Property, String, List, Variable, Select 等）
│   ├── lexer.go         # 词法分析器（基于 text/scanner）
│   ├── parser.go        # 递归下降解析器
│   ├── eval.go          # 表达式求值器（变量替换、字符串插值、select 求值）
│   ├── eval_test.go     # 求值器测试
│   └── parser_test.go   # 解析器测试
├── dag/                 # 有向无环图（DAG）依赖管理
│   ├── graph.go         # 图结构、拓扑排序（Kahn 算法，按层级并行化）
│   └── graph_test.go    # 图测试
├── module/              # 模块类型系统
│   ├── module.go        # Module 接口和 BaseModule 基础实现
│   ├── registry.go      # 线程安全的模块工厂注册表
│   ├── types.go         # 各模块类型定义（CCLibrary, GoBinary, JavaLibrary 等）及工厂
│   └── module_test.go   # 模块测试
├── ninja/               # Ninja 文件生成
│   ├── gen.go           # Generator 核心：从模块图生成 Ninja 构建文件
│   ├── rules.go         # BuildRule 接口、RuleRenderContext、工具函数
│   ├── writer.go        # Ninja 语法写入器（转义、格式化）
│   ├── cc.go            # C/C++ 规则（cc_library, cc_binary 等）
│   ├── go.go            # Go 规则（go_library, go_binary, go_test）
│   ├── java.go          # Java 规则（java_library, java_binary 等）
│   ├── filegroup.go     # filegroup 规则
│   ├── custom.go        # custom 和 proto 规则
│   ├── defaults.go      # defaults, package, soong_namespace 规则
│   ├── ninja_test.go    # Ninja 生成测试
│   └── soong_test.go    # Soong 语法测试
├── examples/            # 示例构建文件
│   ├── Android.bp       # 示例 Blueprint 文件
│   ├── soong_features.bp # Soong 特性示例
│   ├── api.proto        # Proto 示例
│   ├── *.c / *.cpp / *.h # C/C++ 示例源码
│   ├── cmd/server/main.go # Go 示例源码
│   ├── src/             # Go/Java 示例源码
│   └── assets/          # 资源文件示例
├── Android.bp           # minibp 自身的构建定义（用 minibp 构建自身）
├── go.mod               # Go 模块定义
├── README.md            # 项目文档
└── CONTRIBUTING.md      # 贡献指南
```

## 架构与数据流

```
.bp 文件 → [Lexer] → Token 流 → [Parser] → AST (File/Module/Assignment)
    → [Evaluator] → 求值后的 Module
    → [Graph] → 拓扑排序的构建层级
    → [Generator + BuildRules] → Ninja 构建文件
```

1. **词法分析**: `lexer.go` 将源码转为 Token 流
2. **语法分析**: `parser.go` 递归下降构建 AST
3. **求值**: `eval.go` 处理变量赋值、字符串插值 `${var}`、select() 条件求值
4. **变体合并**: `main.go` 中 `mergeVariantProps` 处理 arch/host/target 属性覆盖
5. **Glob 展开**: `main.go` 中 `expandGlobsInModule` 展开 `**/*.go` 等通配符
6. **依赖图**: `dag/graph.go` 构建模块依赖图并拓扑排序
7. **Ninja 生成**: `ninja/gen.go` 遍历排序后的模块，调用各 BuildRule 生成规则和边

## 支持的模块类型

### C/C++ (8 种)
- `cc_library`, `cc_library_static`, `cc_library_shared`, `cc_object`, `cc_binary`
- `cpp_library`, `cpp_binary`, `cc_library_headers`

### Go (3 种)
- `go_library`, `go_binary`, `go_test`

### Java (7 种)
- `java_library`, `java_library_static`, `java_library_host`
- `java_binary`, `java_binary_host`, `java_test`, `java_import`

### Proto (2 种)
- `proto_library`, `proto_gen`

### Soong 语法 (3 种)
- `defaults` (属性复用), `package` (包级默认值), `soong_namespace` (命名空间)

### 其他 (2 种)
- `filegroup`, `custom`

## Soong 语法特性

- **defaults 模块**: 通过 `defaults: ["name"]` 复用属性
- **package 模块**: 包级 `default_visibility` 设置
- **soong_namespace**: 命名空间定义
- **模块引用**: `:module` 和 `:module{.tag}` 语法
- **可见性控制**: `//visibility:public/private/override/legacy_public/any_partition`
- **select 语句**: `select(arch, { arm: [...], default: [...] })` 条件编译
- **arch/host/target 覆盖**: 架构/主机/目标特定属性合并
- **变量赋值**: `name = value` 和 `name += value`
- **字符串插值**: `${variable_name}` 在字符串中替换变量
- **Desc 注释**: `//<source_dir>:<module_name> <action> <src_file>` Soong 风格构建描述
- **传递性头文件**: Option B 风格 — A 依赖 B，B 依赖 C，A 自动包含 C 的头文件
- **通配符支持**: `filegroup` 支持 `**` 递归 glob
- **自定义命令**: custom 规则支持 `$in` 和 `$out` 变量
- **重复规则处理**: 避免重复 Ninja rule 定义

## 命令行用法

```bash
# 解析单个 .bp 文件
go run ./cmd/minibp/main.go Android.bp

# 解析目录下所有 .bp 文件
go run ./cmd/minibp/main.go -a .

# 指定输出文件
go run ./cmd/minibp/main.go -o build.ninja Android.bp

# 指定工具链
go run ./cmd/minibp/main.go -cc clang -cxx clang++ -ar llvm-ar Android.bp

# 指定目标架构
go run ./cmd/minibp/main.go -arch arm64 Android.bp

# 主机构建
go run ./cmd/minibp/main.go -host Android.bp

# 指定目标 OS
go run ./cmd/minibp/main.go -os linux Android.bp

# 运行示例
cd examples && go run ../cmd/minibp/main.go -a . && ninja
```

## 构建与测试命令

```bash
# 构建
go build ./cmd/minibp

# 运行所有测试
go test ./...

# 格式化代码（提交前必须运行）
go fmt ./...

# 完整测试流程（按 CONTRIBUTING.md 要求）
go build -o minibp cmd/minibp/main.go
./minibp -a
ninja
cd examples && ../minibp -a && ninja
```

## 编码规范

- **格式化**: 提交前必须运行 `go fmt`
- **命名**: 导出标识符用 `MixedCaps` (PascalCase)，未导出用 `camelCase`
- **命名质量**: 禁止使用无意义或模糊的标识符名称
- **注释语言**: 代码注释必须使用英语
- **Commit 消息**: 首字母大写，标题不超过 50 字符，正文解释 what/why/how，不超过 200 字符
- **不使用 Conventional Commits**: 不要使用 `feat:`, `fix:` 等前缀
- **AI 代码**: 接受 AI 辅助/生成的代码，但需理解至少一半逻辑，PR 需标明指令者

## 关键接口

### BuildRule (`ninja/rules.go`)
```go
type BuildRule interface {
    Name() string
    NinjaRule(ctx RuleRenderContext) string
    NinjaEdge(m *parser.Module, ctx RuleRenderContext) string
    Outputs(m *parser.Module, ctx RuleRenderContext) []string
    Desc(m *parser.Module, srcFile string) string
}
```

### Module (`module/module.go`)
```go
type Module interface {
    Name() string
    Type() string
    Srcs() []string
    Deps() []string
    Props() map[string]interface{}
    GetProp(key string) interface{}
}
```

### Factory (`module/registry.go`)
```go
type Factory interface {
    Create(ast *parser.Module, eval *parser.Evaluator) (Module, error)
}
```

### Graph (`dag/graph.go`)
```go
type Graph interface {
    TopoSort() ([][]string, error)
}
```

## 常见开发模式

### 添加新模块类型
1. 在 `module/types.go` 中定义模块结构体和工厂
2. 在 `registerBuiltInModuleTypes()` 中注册
3. 在 `ninja/` 下创建或扩展规则文件，实现 `BuildRule` 接口
4. 在 `ninja/rules.go` 的 `GetAllRules()` 中注册规则
5. 编写测试

### 添加新属性
1. 在对应模块结构体中添加字段
2. 在工厂的 `Create` 方法中提取属性
3. 在对应的 Ninja 规则中使用该属性

### 添加新 Soong 语法特性
1. 在 `parser/ast.go` 中扩展 AST 节点（如需要）
2. 在 `parser/parser.go` 中添加解析逻辑
3. 在 `parser/eval.go` 中添加求值逻辑
4. 在 `ninja/` 中添加对应的规则处理

## 注意事项

- `cmd/minibp/main.go` 中包含一个内联的 `Graph` 结构体和拓扑排序实现，与 `dag/graph.go` 中的实现不同（前者使用 `*parser.Module`，后者使用 `module.Module` 接口）
- Ninja 路径转义：`$` → `$$`，`:` → `$:`，`#` → `$#`，空格 → `$ `
- 模块引用格式：`:moduleName` 或 `:moduleName{.tag}`
- Glob 展开在变体合并之后执行，确保 arch/host/target 中的 glob 也被展开
- `defaults` 属性合并时，列表追加，标量覆盖，目标模块已有属性优先