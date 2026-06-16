
# 输出语言限定
Always respond in Chinese-simplified.

All explanations must be written in Simplified Chinese.
All comments in generated code must be written in Simplified Chinese.

When generating Go code:
- Follow golangci-lint best practices
- Prefer idiomatic Go


# JSON_Tag_Naming_Convention
## 规则描述
强制要求所有JSON标签（键名）使用小写字母，多个单词之间以下划线连接。

## 执行级别
ERROR

## 规范细则
| 配置项         | 取值/说明                                                                 |
|----------------|---------------------------------------------------------------------------|
| 字符大小写     | 小写（lowercase）                                                         |
| 单词分隔符     | 下划线（_）                                                               |
| 允许字符       | 小写字母（a-z）、数字（0-9）、下划线（_）                                 |
| 禁用模式       | 包含大写字母（[A-Z]）、空白字符（\s）、连字符（-）、点号（.）的键名均不允许 |

## 示例
### 有效示例
```json
{
  "user_id": 123,
  "first_name": "John",
  "order_status": "shipped",
  "product_sku": "ABC-123"
}
```

### 无效示例
```json
{
  "userId": 123,
  "firstName": "John",
  "orderStatus": "shipped",
  "productSKU": "ABC-123"
}
```

## 代码生成提示
生成JSON时，请确保键名全部小写且多词以下划线连接。例如使用 'user_name' 而非 'userName' 或 'UserName'。


# 开发执行限定
- 框架内容必须使用glue(github.com/zhiyunliu/glue)进行处理开发
- 所有新功能必须使用glue-microservice技能进行处理
- 功能代码包和管理后台功能必须分为不同的目录
- 管理功能放在management目录下
- 功能代码放在根目录下


# 问题决策
- 当有不清除的问题时，请一定要提问，直到问题清楚为止,禁止随意发挥猜测