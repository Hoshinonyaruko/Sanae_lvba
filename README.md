"# Sanae_lvba" 
# 绿坝机器人内容审计接口

**适用于有中心服务器的机器人，适用于中文场景，通过关键词过滤、记录，及时发现机器人被邀请到危险的群并及时处理**

**适合中文环境下的敏感词匹配，不太适合英文环境下的**

## 简介

这是一个使用 Go 语言实现的审计接口项目，用于实时监控敏感词，并将触发敏感词的相关信息记录到日志中。这个审计接口采用 Aho-Corasick 字符串匹配算法来快速检查是否有敏感词出现，并通过 HTTP API 的形式对外提供服务.

## 主要功能

- **敏感词匹配**：使用 Aho-Corasick 算法，对输入的文字进行敏感词匹配。
- **实时监控**：接口采用 HTTP 形式，可以与其他服务集成，实现实时敏感词审计。
- **日志记录**：将触发敏感词的事件记录到日志文件中，日志文件会按日期自动切割。
- **异步日志**：通过 channel 和 goroutine 实现异步日志记录，提高性能。

## 代码结构

- **ACNode 和 AhoCorasick**：用于实现 Aho-Corasick 算法的数据结构和相关方法。
- **loadWordsIntoAC**：从文件中加载敏感词，并构建 Aho-Corasick 树。
- **initLoggerForToday 和 initGlobalLogger**：用于初始化日志设置。
- **auditHandler**：核心审计逻辑，处理 HTTP 请求并执行敏感词匹配。

## 如何使用

1. 将敏感词列表存放在 `sensitive_words.txt` 文件中。
2. 运行程序，它将监听 5002 端口。
3. 通过 HTTP 请求向 `/audit` 发送要审计的词、机器人名称、群组 ID 和好友 ID。
   - **示例**：`http://localhost:5002/audit?word=测试&bot=bot1&groupid=12345&friendid=67890`

## 日志

日志文件将保存在 `audit_logs` 文件夹下，并按日期进行切割。

## 注意事项

- 确保 `sensitive_words.txt` 文件存在并且可读。
- 由于是实时审计，应确保网络通信无延迟和丢包。

## 依赖

- [github.com/natefinch/lumberjack](https://github.com/natefinch/lumberjack): 用于日志文件的管理。
- [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus): 用于日志的记录。

## 应用场景

这个审计接口适用于需要实时监控敏感词或信息安全审计的场景，如聊天应用、社交网络、评论系统等。
