# internal/app

本目录用于后续 glue 应用层模块聚合。

规划用途：
- 聚合 API/RPC/CRON/MQC 模块注册。
- 承接业务模块与启动装配之间的边界。
- 在迁移期间支持与 legacy handler 并行存在。
