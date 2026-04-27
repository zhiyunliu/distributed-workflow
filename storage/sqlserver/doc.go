// Package sqlserver 提供基于 SQL Server 的 WorkflowRepository 实现。
// 使用 database/sql + microsoft/go-mssqldb 驱动，所有写操作通过事务保证原子性，
// 所有查询使用参数化预编译语句，防止 SQL 注入。
package sqlserver
