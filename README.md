一个简单的代币空投管理平台

这是一个使用Go语言开发的代币空投管理平台，用于创建、管理和分发代币空投活动。

### 系统架构
采用标准的Go项目结构，使用MVC架构模式：
```
airdrop-system/
├── cmd/                   # 程序入口
│   └── server/
│       └── main.go       # 主程序入口
├── internal/              # 内部代码
│   ├── config/           # 配置管理
│   ├── database/         # 数据库操作
│   ├── handlers/         # HTTP处理函数
│   ├── middleware/       # 中间件
│   ├── models/           # 数据模型
│   └── services/         # 业务逻辑
├── templates/            # HTML模板
├── data/                 # 数据存储（SQLite数据库）
└── config.yaml           # 配置文件
```

### 核心功能模块

1. 配置管理 (internal/config/config.go)
   - 使用viper库管理配置
   - 支持YAML配置文件和环境变量
   - 自动创建必要的目录结构
   - 配置验证和默认值设置

2. 数据库模块 (internal/database/database.go)
   - 使用SQLite作为默认数据库（modernc.org/sqlite驱动）
   - 自动创建表结构
   - 初始化默认管理员账户
   - 表结构：
     - admins：管理员表
     - airdrops：空投活动表
     - participants：参与者表

3. 数据模型 (internal/models/models.go)
   - Airdrop：空投活动模型
   - Participant：参与者模型
   - Admin：管理员模型

4. HTTP处理函数 (internal/handlers/)
   - airdrop.go：空投创建和参与逻辑
   - distribute.go：奖励分发逻辑
   - common.go：页面渲染函数
   - auth.go：认证相关逻辑

5. 业务逻辑 (internal/services/)
   - distribution.go：实现随机选择中奖者和分配奖励的核心算法

6. 中间件 (internal/middleware/)
   - CORS中间件：支持跨域请求
   - Database中间件：为请求添加数据库连接
   - AuthRequired中间件：管理员认证检查
### 核心功能流程
1. 创建空投活动
   
   - 管理员通过界面创建空投活动
   - 设置活动名称、描述、时间、总代币数等
2. 用户参与
   
   - 用户访问空投页面
   - 提交Twitter账号和钱包地址
   - 系统验证空投活动是否有效
   - 检查用户是否已参与
   - 创建参与者记录
3. 分配奖励
   
   - 管理员设置中奖比例
   - 系统随机选择中奖者
   - 计算每个中奖者的奖励金额
   - 更新数据库记录
   - 标记空投活动为已分发
### 技术特点
- 轻量级 ：使用Go语言开发，资源占用少
- 易部署 ：使用SQLite数据库，无需额外数据库服务
- 灵活配置 ：支持YAML配置和环境变量
- 安全 ：简单的管理员认证机制
- 可扩展 ：模块化设计，便于添加新功能
### 启动和运行
1. 安装依赖： go mod tidy
2. 运行： go run cmd/server/main.go
3. 访问： http://localhost:8080
4. 管理员登录： http://localhost:8080/admin/login