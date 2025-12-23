# 异步生成 API 文档

> 图片和视频生成均采用异步任务模式

## 概述

本 API 提供图片和视频的异步生成能力。所有生成请求都会立即返回一个任务 ID，您需要通过查询接口获取任务状态和结果。

## 认证

所有 API 请求需要在请求头中携带 Bearer Token：

```
Authorization: Bearer YOUR_API_KEY
```

---

## 图片生成

### 1. 提交图片生成任务

**请求**

```
POST /v1/images/generations
Content-Type: application/json
```

**请求参数**

| 参数    | 类型    | 必填 | 说明                       |
| ------- | ------- | ---- | -------------------------- |
| model   | string  | ✅   | 模型名称                   |
| prompt  | string  | ✅   | 图片描述提示词             |
| n       | integer | ❌   | 生成数量，默认 1           |
| size    | string  | ❌   | 尺寸比例，如 `1:1`, `16:9` |
| quality | string  | ❌   | 质量，如 `standard`, `hd`  |

**支持的模型**

| 模型名称                     | 说明              |
| ---------------------------- | ----------------- |
| `nano-banana-pro`            | 高质量图片生成    |
| `nano-banana`                | 标准图片生成      |
| `gpt-4o-image`               | GPT-4o 图片生成   |
| `gemini-3-pro-image-preview` | Gemini 图片生成   |
| `seedream-4.5`               | Seedream 图片生成 |
| `seedream-4`                 | Seedream 标准版   |

**请求示例**

```bash
curl -X POST "https://your-domain.com/v1/images/generations" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nano-banana-pro",
    "prompt": "一只穿着宇航服的猫咪在月球上行走",
    "size": "1:1",
    "n": 1
  }'
```

**响应示例**

```json
{
  "id": "task_img_abc123def456",
  "object": "generation.task",
  "model": "nano-banana-pro",
  "status": "queued",
  "progress": 0,
  "created_at": 1703884800,
  "metadata": {}
}
```

---

### 2. 查询图片生成任务

**请求**

```
GET /v1/images/generations/:task_id
```

**请求示例**

```bash
curl "https://your-domain.com/v1/images/generations/task_img_abc123def456" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**响应示例（进行中）**

```json
{
  "id": "task_img_abc123def456",
  "object": "generation.task",
  "model": "nano-banana-pro",
  "status": "in_progress",
  "progress": 60,
  "created_at": 1703884800,
  "metadata": {}
}
```

**响应示例（成功）**

```json
{
  "id": "task_img_abc123def456",
  "object": "generation.task",
  "model": "nano-banana-pro",
  "status": "completed",
  "progress": 100,
  "created_at": 1703884800,
  "completed_at": 1703884830,
  "expires_at": 1703971200,
  "result": {
    "type": "image",
    "data": [
      {
        "url": "https://cdn.example.com/images/generated_abc123.png",
        "revised_prompt": "一只穿着白色宇航服的橘色猫咪在月球表面行走，背景是地球"
      }
    ]
  },
  "metadata": {}
}
```

---

## 视频生成

### 1. 提交视频生成任务

**请求**

```
POST /v1/videos/generations
Content-Type: application/json
```

**请求参数**

| 参数         | 类型    | 必填 | 说明                      |
| ------------ | ------- | ---- | ------------------------- |
| model        | string  | ✅   | 模型名称                  |
| prompt       | string  | ✅   | 视频描述提示词            |
| duration     | integer | ❌   | 视频时长（秒），默认 5    |
| size         | string  | ❌   | 尺寸，如 `1920x1080`      |
| aspect_ratio | string  | ❌   | 宽高比，如 `16:9`, `9:16` |

**支持的模型**

| 模型名称     | 说明         | 最大时长 |
| ------------ | ------------ | -------- |
| `sora-2`     | Sora 标准版  | 10s      |
| `sora-2-pro` | Sora 专业版  | 15s      |
| `veo-3.1`    | Veo 视频生成 | 8s       |

**请求示例**

```bash
curl -X POST "https://your-domain.com/v1/videos/generations" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sora-2-pro",
    "prompt": "一只快乐的金毛犬在草地上奔跑，阳光明媚",
    "duration": 10,
    "aspect_ratio": "16:9"
  }'
```

**响应示例**

```json
{
  "id": "task_vid_xyz789ghi012",
  "object": "generation.task",
  "model": "sora-2-pro",
  "status": "queued",
  "progress": 0,
  "created_at": 1703884800,
  "metadata": {}
}
```

---

### 2. 查询视频生成任务

**请求**

```
GET /v1/videos/generations/:task_id
```

**请求示例**

```bash
curl "https://your-domain.com/v1/videos/generations/task_vid_xyz789ghi012" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**响应示例（进行中）**

```json
{
  "id": "task_vid_xyz789ghi012",
  "object": "generation.task",
  "model": "sora-2-pro",
  "status": "in_progress",
  "progress": 35,
  "created_at": 1703884800,
  "metadata": {}
}
```

**响应示例（成功）**

```json
{
  "id": "task_vid_xyz789ghi012",
  "object": "generation.task",
  "model": "sora-2-pro",
  "status": "completed",
  "progress": 100,
  "created_at": 1703884800,
  "completed_at": 1703885100,
  "expires_at": 1703971500,
  "result": {
    "type": "video",
    "data": [
      {
        "url": "https://cdn.example.com/videos/generated_xyz789.mp4",
        "duration": 10,
        "size": "1920x1080",
        "format": "mp4"
      }
    ]
  },
  "metadata": {}
}
```

---

## 任务状态说明

| 状态          | 说明                     | 是否终态 | 建议操作                 |
| ------------- | ------------------------ | -------- | ------------------------ |
| `queued`      | 任务已提交，排队等待处理 | ❌       | 等待 2-5 秒后重试查询    |
| `in_progress` | 任务正在处理中           | ❌       | 等待 3-10 秒后重试查询   |
| `completed`   | 任务成功完成             | ✅       | 获取 result 中的资源 URL |
| `failed`      | 任务处理失败             | ✅       | 检查 error 信息          |

---

## 轮询策略建议

### 图片生成

```
初始等待: 2 秒
轮询间隔: 3 秒
最大等待: 120 秒
```

### 视频生成

```
初始等待: 5 秒
轮询间隔: 10 秒
最大等待: 300 秒
```

### 示例代码（JavaScript）

```javascript
async function waitForTask(taskId, type = "images") {
  const endpoint = `https://your-domain.com/v1/${type}/generations/${taskId}`;
  const maxAttempts = type === "videos" ? 30 : 40;
  const interval = type === "videos" ? 10000 : 3000;

  for (let i = 0; i < maxAttempts; i++) {
    const response = await fetch(endpoint, {
      headers: { Authorization: "Bearer YOUR_API_KEY" },
    });
    const data = await response.json();

    if (data.status === "completed") {
      return data.result;
    }
    if (data.status === "failed") {
      throw new Error(data.error?.message || "Task failed");
    }

    await new Promise((r) => setTimeout(r, interval));
  }
  throw new Error("Task timeout");
}

// 使用示例
const submitResponse = await fetch(
  "https://your-domain.com/v1/images/generations",
  {
    method: "POST",
    headers: {
      Authorization: "Bearer YOUR_API_KEY",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      model: "nano-banana-pro",
      prompt: "一只可爱的猫咪",
    }),
  }
);

const { id } = await submitResponse.json();
const result = await waitForTask(id, "images");
console.log("Generated image:", result.data[0].url);
```

### 示例代码（Python）

```python
import time
import requests

def wait_for_task(task_id: str, task_type: str = 'images') -> dict:
    endpoint = f"https://your-domain.com/v1/{task_type}/generations/{task_id}"
    headers = {"Authorization": "Bearer YOUR_API_KEY"}

    max_attempts = 30 if task_type == 'videos' else 40
    interval = 10 if task_type == 'videos' else 3

    for _ in range(max_attempts):
        response = requests.get(endpoint, headers=headers)
        data = response.json()

        if data['status'] == 'completed':
            return data['result']
        if data['status'] == 'failed':
            raise Exception(data.get('error', {}).get('message', 'Task failed'))

        time.sleep(interval)

    raise Exception('Task timeout')


# 使用示例
response = requests.post(
    'https://your-domain.com/v1/images/generations',
    headers={
        'Authorization': 'Bearer YOUR_API_KEY',
        'Content-Type': 'application/json'
    },
    json={
        'model': 'nano-banana-pro',
        'prompt': '一只可爱的猫咪'
    }
)

task_id = response.json()['id']
result = wait_for_task(task_id, 'images')
print(f"Generated image: {result['data'][0]['url']}")
```

---

## 错误码说明

| HTTP 状态码 | 错误类型                   | 说明                   |
| ----------- | -------------------------- | ---------------------- |
| 400         | `invalid_request`          | 请求参数无效           |
| 401         | `unauthorized`             | 认证失败，检查 API Key |
| 402         | `insufficient_quota`       | 余额不足               |
| 404         | `task_not_found`           | 任务不存在             |
| 422         | `content_policy_violation` | 内容违规               |
| 429         | `rate_limit_exceeded`      | 请求频率超限           |
| 500         | `internal_error`           | 服务器内部错误         |

**错误响应格式**

```json
{
  "error": {
    "code": "invalid_request",
    "message": "The 'prompt' field is required.",
    "param": "prompt",
    "type": "invalid_request_error"
  }
}
```

---

## 资源有效期

- 生成的图片和视频资源 URL 有效期为 **24 小时**
- 请在有效期内下载保存资源
- `expires_at` 字段标识资源过期时间（Unix 时间戳）

---

## 费用说明

| 模型            | 单价   | 计费单位 |
| --------------- | ------ | -------- |
| nano-banana-pro | ¥0.04  | 每张图片 |
| nano-banana     | ¥0.02  | 每张图片 |
| gpt-4o-image    | ¥0.006 | 每张图片 |
| sora-2          | ¥0.30  | 每个视频 |
| sora-2-pro      | ¥0.50  | 每个视频 |
| veo-3.1         | ¥0.08  | 每个视频 |

> 注：任务失败不计费

---

## 常见问题

### Q: 任务提交后多久能完成？

| 类型     | 典型耗时 |
| -------- | -------- |
| 图片生成 | 5-30 秒  |
| 视频生成 | 1-5 分钟 |

### Q: 如何处理任务超时？

如果超过预期时间任务仍未完成，建议：

1. 继续等待并增大轮询间隔
2. 检查任务状态是否为 `failed`
3. 联系技术支持

### Q: 支持批量生成吗？

图片生成支持通过 `n` 参数指定数量（最大 4 张），视频暂不支持批量生成。

### Q: 资源 URL 过期后怎么办？

资源过期后无法访问，建议在有效期内下载保存。如需重新获取，需要重新提交生成任务。
