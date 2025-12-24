# APIMart 统一任务适配器 - 实施总结

## 实施日期

2025-12-22

## 核心功能

成功为 new-api 项目开发了统一的 APIMart 平台适配器，同时支持视频生成和图片生成任务。

## 关键优势

1. **一套代码双模态**：视频和图片使用相同的适配器逻辑
2. **自动类型检测**：根据请求路径自动识别任务类型
3. **统一响应格式**：利用 APIMart 的标准化响应
4. **简化配置**：只需配置一个渠道即可支持所有模型
5. **易于维护**：代码复用率高，维护成本低

## 修改文件清单

### 新增文件（2 个）

- `relay/channel/task/apimart/constants.go` - 常量和模型列表定义
- `relay/channel/task/apimart/adaptor.go` - 统一任务适配器实现

### 修改文件（8 个）

1. `constant/task.go` - 添加 APIMart 平台常量和任务动作
2. `constant/channel.go` - 添加 APIMart 渠道类型（58）
3. `relay/relay_adaptor.go` - 注册 APIMart 适配器
4. `router/relay-router.go` - 图片生成路由支持 APIMart
5. `router/video-router.go` - 添加视频生成路由
6. `controller/task.go` - 添加 APIMart 任务轮询逻辑
7. `model/task.go` - 扩展任务属性支持任务类型
8. `setting/ratio_setting/model_ratio.go` - 配置所有模型价格

## 支持的模型

### 视频生成模型

- `sora-2` (价格倍率: 0.3)
- `sora-2-pro` (价格倍率: 0.5)
- `veo-3.1` (价格倍率: 0.08)

### 图片生成模型

- `gpt-4o-image` (价格倍率: 0.006)
- `gemini-3-pro-image-preview` (价格倍率: 0.055)
- `gemini-2.5-flash-image-preview` (价格倍率: 0.03)
- `nano-banana-pro` (价格倍率: 0.04)
- `nano-banana` (价格倍率: 0.02)
- `seedream-4.5` (价格倍率: 0.05)
- `seedream-4` (价格倍率: 0.03)

## API 路由

### 视频生成

- `POST /v1/videos/generations` - 提交视频生成任务
- `GET /v1/videos/generations/:task_id` - 查询视频生成任务状态

### 图片生成

- `POST /v1/images/generations` - 提交图片生成任务（自动检测 APIMart 渠道）
- `GET /v1/images/generations/:task_id` - 查询图片生成任务状态

## 配置示例

```yaml
渠道名称: APIMart 统一渠道
渠道类型: APIMart (58)
Base URL: https://api.apimart.ai
API Key: your_apimart_key

支持的模型（视频+图片）:
  - sora-2
  - sora-2-pro
  - veo-3.1
  - gpt-4o-image
  - gemini-3-pro-image-preview
  - gemini-2.5-flash-image-preview
  - nano-banana-pro
  - nano-banana
  - seedream-4.5
  - seedream-4

用户分组: default,vip
优先级: 10
权重: 1
状态: 启用
```

## 使用示例

### 视频生成

```bash
curl -X POST "http://your-domain/v1/videos/generations" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sora-2-pro",
    "prompt": "a happy dog running in the garden",
    "duration": 10
  }'
```

### 图片生成

```bash
curl -X POST "http://your-domain/v1/images/generations" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nano-banana-pro",
    "prompt": "a serene lake at sunset",
    "size": "1:1",
    "n": 1
  }'
```

### 查询任务状态

```bash
curl "http://your-domain/v1/videos/generations/task_xxx" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 技术实现细节

### 自动类型检测

适配器通过请求路径自动识别任务类型：

- 路径包含 `/videos/` → 视频任务
- 路径包含 `/images/` → 图片任务

### 统一轮询机制

- 每 15 秒轮询一次未完成的任务
- 根据任务的 `action` 字段确定任务类型
- 失败任务自动退款（防止重复退款）
- 支持任务状态映射：`submitted`, `processing`, `succeeded`, `failed`

### 状态映射

```go
submitted/queued → SUBMITTED
processing/in_progress → IN_PROGRESS
succeeded/completed → SUCCESS
failed/error → FAILURE
```

## 验证状态

✅ 编译成功，无语法错误
✅ 无 linter 错误
✅ 所有 TODO 项已完成

## 下一步建议

1. 在测试环境中配置 APIMart 渠道
2. 测试视频生成和图片生成功能
3. 监控任务轮询性能
4. 根据实际使用情况调整模型价格倍率

## 注意事项

- 确保 APIMart API Key 正确配置
- 注意模型价格倍率需要根据实际成本调整
- 任务轮询间隔为 15 秒，可根据需要调整
- 失败任务会自动退款，请确保额度计算准确
