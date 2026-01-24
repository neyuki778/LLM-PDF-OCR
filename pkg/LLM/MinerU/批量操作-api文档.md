# MinerU 批量文件解析 API 文档

## 目录
- [文件批量上传解析](#文件批量上传解析)
- [URL 批量上传解析](#url-批量上传解析)
- [批量获取任务结果](#批量获取任务结果)
- [常见错误码](#常见错误码)

---

## 文件批量上传解析

### 接口说明

适用于本地文件上传解析的场景，可通过此接口批量申请文件上传链接，上传文件后，系统会自动提交解析任务。

**注意事项：**

- 申请的文件上传链接有效期为 **24 小时**，请在有效期内完成文件上传
- 上传文件时，**无须设置** Content-Type 请求头
- 文件上传完成后，**无须调用**提交解析任务接口，系统会自动扫描已上传完成文件并自动提交解析任务
- 单次申请链接不能超过 **200 个**
- header 头中需要包含 Authorization 字段，格式为 `Bearer + 空格 + Token`

### Python 请求示例

#### PDF、DOC、PPT、图片文件

```python
import requests

token = "官网申请的api token"
url = "https://mineru.net/api/v4/file-urls/batch"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}
data = {
    "files": [
        {"name":"demo.pdf", "data_id": "abcd"}
    ],
    "model_version":"vlm"
}
file_path = ["demo.pdf"]
try:
    response = requests.post(url,headers=header,json=data)
    if response.status_code == 200:
        result = response.json()
        print('response success. result:{}'.format(result))
        if result["code"] == 0:
            batch_id = result["data"]["batch_id"]
            urls = result["data"]["file_urls"]
            print('batch_id:{},urls:{}'.format(batch_id, urls))
            for i in range(0, len(urls)):
                with open(file_path[i], 'rb') as f:
                    res_upload = requests.put(urls[i], data=f)
                    if res_upload.status_code == 200:
                        print(f"{urls[i]} upload success")
                    else:
                        print(f"{urls[i]} upload failed")
        else:
            print('apply upload url failed,reason:{}'.format(result.msg))
    else:
        print('response not success. status:{} ,result:{}'.format(response.status_code, response))
except Exception as err:
    print(err)
```

#### HTML 文件

```python
import requests

token = "官网申请的api token"
url = "https://mineru.net/api/v4/file-urls/batch"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}
data = {
    "files": [
        {"name":"demo.html", "data_id": "abcd"}
    ],
    "model_version":"MinerU-HTML"
}
file_path = ["demo.html"]
try:
    response = requests.post(url,headers=header,json=data)
    if response.status_code == 200:
        result = response.json()
        print('response success. result:{}'.format(result))
        if result["code"] == 0:
            batch_id = result["data"]["batch_id"]
            urls = result["data"]["file_urls"]
            print('batch_id:{},urls:{}'.format(batch_id, urls))
            for i in range(0, len(urls)):
                with open(file_path[i], 'rb') as f:
                    res_upload = requests.put(urls[i], data=f)
                    if res_upload.status_code == 200:
                        print(f"{urls[i]} upload success")
                    else:
                        print(f"{urls[i]} upload failed")
        else:
            print('apply upload url failed,reason:{}'.format(result.msg))
    else:
        print('response not success. status:{} ,result:{}'.format(response.status_code, response))
except Exception as err:
    print(err)
```

### CURL 请求示例

#### PDF、DOC、PPT、图片文件

```bash
curl --location --request POST 'https://mineru.net/api/v4/file-urls/batch' \
--header 'Authorization: Bearer ***' \
--header 'Content-Type: application/json' \
--header 'Accept: */*' \
--data-raw '{
    "files": [
        {"name":"demo.pdf", "data_id": "abcd"}
    ],
    "model_version": "vlm"
}'
```

#### HTML 文件

```bash
curl --location --request POST 'https://mineru.net/api/v4/file-urls/batch' \
--header 'Authorization: Bearer ***' \
--header 'Content-Type: application/json' \
--header 'Accept: */*' \
--data-raw '{
    "files": [
        {"name":"demo.html", "data_id": "abcd"}
    ],
    "model_version": "MinerU-HTML"
}'
```

#### 文件上传示例

```bash
curl -X PUT -T /path/to/your/file.pdf 'https://****'
```

### 请求体参数说明

| 参数 | 类型 | 必选 | 示例 | 描述 |
|------|------|------|------|------|
| `enable_formula` | bool | 否 | true | 是否开启公式识别，默认 true，仅对 pipeline 模型且非 html 文件有效 |
| `enable_table` | bool | 否 | true | 是否开启表格识别，默认 true，仅对 pipeline 模型且非 html 文件有效 |
| `language` | string | 否 | ch | 指定文档语言，默认 ch，其他可选值列表详见：[PaddleOCR 多语言文档](https://www.paddleocr.ai/latest/version3.x/algorithm/PP-OCRv5/PP-OCRv5_multi_languages.html#_3)，仅对 pipeline 模型且非 html 文件有效 |
| `file.name` | string | **是** | demo.pdf | 文件名，支持 `.pdf`、`.doc`、`.docx`、`.ppt`、`.pptx`、`.png`、`.jpg`、`.jpeg`、`.html` 多种格式，强烈建议文件名带上正确的后缀名 |
| `file.is_ocr` | bool | 否 | true | 是否启动 OCR 功能，默认 false，仅对 pipeline 模型且非 html 文件有效 |
| `file.data_id` | string | 否 | abc** | 解析对象对应的数据 ID。由大小写英文字母、数字、下划线（_）、短划线（-）、英文句号（.）组成，不超过 128 个字符，可用于唯一标识您的业务数据 |
| `file.page_ranges` | string | 否 | 1-600 | 指定页码范围，格式为逗号分隔的字符串。例如：`"2,4-6"` 表示选取第2页、第4页至第6页；`"2--2"` 表示从第2页一直选取到倒数第二页 |
| `callback` | string | 否 | http://127.0.0.1/callback | 解析结果回调通知您的 URL，支持 HTTP 和 HTTPS 协议。该字段为空时，您必须定时轮询解析结果。<br>**checksum**：由用户 uid + seed + content 拼成字符串，通过 SHA256 算法生成<br>**content**：JSON 字符串格式，需自行解析<br>**说明**：callback 接口返回 HTTP 200 表示接收成功，其他状态码视为失败。失败时将最多重复推送 5 次 |
| `seed` | string | 否 | abc** | 随机字符串，用于回调通知请求中的签名。由英文字母、数字、下划线（_）组成，不超过 64 个字符。**当使用 callback 时，该字段必须提供** |
| `extra_formats` | [string] | 否 | ["docx","html"] | markdown、json 为默认导出格式，无须设置。该参数仅支持 `docx`、`html`、`latex` 三种格式中的一个或多个。对源文件为 html 的文件无效 |
| `model_version` | string | 否 | vlm | MinerU 模型版本，三个选项：`pipeline`、`vlm`、`MinerU-HTML`，默认 pipeline。如果解析 HTML 文件，需明确指定为 `MinerU-HTML` |

### 响应参数说明

| 参数 | 类型 | 示例 | 说明 |
|------|------|------|------|
| `code` | int | 0 | 接口状态码，成功：0 |
| `msg` | string | ok | 接口处理信息，成功："ok" |
| `trace_id` | string | c876cd60b202f2396de1f9e39a1b0172 | 请求 ID |
| `data.batch_id` | string | 2bb2f0ec-a336-4a0a-b61a-**** | 批量提取任务 ID，可用于批量查询解析结果 |
| `data.file_urls` | [string] | ["https://mineru.oss-cn-shanghai.aliyuncs.com/api-upload/***"] | 文件上传链接 |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "batch_id": "2bb2f0ec-a336-4a0a-b61a-241afaf9cc87",
    "file_urls": [
        "https://***"
    ]
  },
  "msg": "ok",
  "trace_id": "c876cd60b202f2396de1f9e39a1b0172"
}
```

---

## URL 批量上传解析

### 接口说明

适用于通过 API 批量创建提取任务的场景。

**注意事项：**

- 单次申请链接不能超过 **200 个**
- 文件大小不能超过 **200MB**，文件页数不超出 **600 页**
- 因网络限制，github、aws 等国外 URL 会请求超时
- header 头中需要包含 Authorization 字段，格式为 `Bearer + 空格 + Token`

### Python 请求示例

#### PDF、DOC、PPT、图片文件

```python
import requests

token = "官网申请的api token"
url = "https://mineru.net/api/v4/extract/task/batch"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}
data = {
    "files": [
        {"url":"https://cdn-mineru.openxlab.org.cn/demo/example.pdf", "data_id": "abcd"}
    ],
    "model_version": "vlm"
}
try:
    response = requests.post(url,headers=header,json=data)
    if response.status_code == 200:
        result = response.json()
        print('response success. result:{}'.format(result))
        if result["code"] == 0:
            batch_id = result["data"]["batch_id"]
            print('batch_id:{}'.format(batch_id))
        else:
            print('submit task failed,reason:{}'.format(result.msg))
    else:
        print('response not success. status:{} ,result:{}'.format(response.status_code, response))
except Exception as err:
    print(err)
```

#### HTML 文件

```python
import requests

token = "官网申请的api token"
url = "https://mineru.net/api/v4/extract/task/batch"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}
data = {
    "files": [
        {"url":"https://***", "data_id": "abcd"}
    ],
    "model_version": "MinerU-HTML"
}
try:
    response = requests.post(url,headers=header,json=data)
    if response.status_code == 200:
        result = response.json()
        print('response success. result:{}'.format(result))
        if result["code"] == 0:
            batch_id = result["data"]["batch_id"]
            print('batch_id:{}'.format(batch_id))
        else:
            print('submit task failed,reason:{}'.format(result.msg))
    else:
        print('response not success. status:{} ,result:{}'.format(response.status_code, response))
except Exception as err:
    print(err)
```

### CURL 请求示例

#### PDF、DOC、PPT、图片文件

```bash
curl --location --request POST 'https://mineru.net/api/v4/extract/task/batch' \
--header 'Authorization: Bearer ***' \
--header 'Content-Type: application/json' \
--header 'Accept: */*' \
--data-raw '{
    "files": [
        {"url":"https://cdn-mineru.openxlab.org.cn/demo/example.pdf", "data_id": "abcd"}
    ],
    "model_version": "vlm"
}'
```

#### HTML 文件

```bash
curl --location --request POST 'https://mineru.net/api/v4/extract/task/batch' \
--header 'Authorization: Bearer ***' \
--header 'Content-Type: application/json' \
--header 'Accept: */*' \
--data-raw '{
    "files": [
        {"url":"https://***", "data_id": "abcd"}
    ],
    "model_version": "MinerU-HTML"
}'
```

### 请求体参数说明

| 参数 | 类型 | 必选 | 示例 | 描述 |
|------|------|------|------|------|
| `enable_formula` | bool | 否 | true | 是否开启公式识别，默认 true，仅对 pipeline 模型且非 html 文件有效 |
| `enable_table` | bool | 否 | true | 是否开启表格识别，默认 true，仅对 pipeline 模型且非 html 文件有效 |
| `language` | string | 否 | ch | 指定文档语言，默认 ch，其他可选值列表详见：[PaddleOCR 多语言文档](https://www.paddleocr.ai/latest/version3.x/algorithm/PP-OCRv5/PP-OCRv5_multi_languages.html#_3)，仅对 pipeline 模型且非 html 文件有效 |
| `file.url` | string | **是** | demo.pdf | 文件链接，支持 `.pdf`、`.doc`、`.docx`、`.ppt`、`.pptx`、`.png`、`.jpg`、`.jpeg`、`.html` 多种格式 |
| `file.is_ocr` | bool | 否 | true | 是否启动 OCR 功能，默认 false，仅对 pipeline 模型且非 html 文件有效 |
| `file.data_id` | string | 否 | abc** | 解析对象对应的数据 ID。由大小写英文字母、数字、下划线（_）、短划线（-）、英文句号（.）组成，不超过 128 个字符，可用于唯一标识您的业务数据 |
| `file.page_ranges` | string | 否 | 1-600 | 指定页码范围，格式为逗号分隔的字符串。例如：`"2,4-6"` 表示选取第2页、第4页至第6页；`"2--2"` 表示从第2页一直选取到倒数第二页 |
| `callback` | string | 否 | http://127.0.0.1/callback | 解析结果回调通知您的 URL，支持 HTTP 和 HTTPS 协议。该字段为空时，您必须定时轮询解析结果。<br>**checksum**：由用户 uid + seed + content 拼成字符串，通过 SHA256 算法生成<br>**content**：JSON 字符串格式，需自行解析<br>**说明**：callback 接口返回 HTTP 200 表示接收成功，其他状态码视为失败。失败时将最多重复推送 5 次 |
| `seed` | string | 否 | abc** | 随机字符串，用于回调通知请求中的签名。由英文字母、数字、下划线（_）组成，不超过 64 个字符。**当使用 callback 时，该字段必须提供** |
| `extra_formats` | [string] | 否 | ["docx","html"] | markdown、json 为默认导出格式，无须设置。该参数仅支持 `docx`、`html`、`latex` 三种格式中的一个或多个。对源文件为 html 的文件无效 |
| `model_version` | string | 否 | vlm | MinerU 模型版本，三个选项：`pipeline`、`vlm`、`MinerU-HTML`，默认 pipeline。如果解析 HTML 文件，需明确指定为 `MinerU-HTML` |

### 请求体示例

```json
{
    "files": [
        {"url":"https://cdn-mineru.openxlab.org.cn/demo/example.pdf", "data_id": "abcd"}
    ],
    "model_version": "vlm"
}
```

### 响应参数说明

| 参数 | 类型 | 示例 | 说明 |
|------|------|------|------|
| `code` | int | 0 | 接口状态码，成功：0 |
| `msg` | string | ok | 接口处理信息，成功："ok" |
| `trace_id` | string | c876cd60b202f2396de1f9e39a1b0172 | 请求 ID |
| `data.batch_id` | string | 2bb2f0ec-a336-4a0a-b61a-**** | 批量提取任务 ID，可用于批量查询解析结果 |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "batch_id": "2bb2f0ec-a336-4a0a-b61a-241afaf9cc87"
  },
  "msg": "ok",
  "trace_id": "c876cd60b202f2396de1f9e39a1b0172"
}
```

---

## 批量获取任务结果

### 接口说明

通过 `batch_id` 批量查询提取任务的进度。

### Python 请求示例

```python
import requests

token = "官网申请的api token"
batch_id = "your_batch_id"
url = f"https://mineru.net/api/v4/extract-results/batch/{batch_id}"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}

res = requests.get(url, headers=header)
print(res.status_code)
print(res.json())
print(res.json()["data"])
```

### CURL 请求示例

```bash
curl --location --request GET 'https://mineru.net/api/v4/extract-results/batch/{batch_id}' \
--header 'Authorization: Bearer *****' \
--header 'Accept: */*'
```

### 响应参数说明

| 参数 | 类型 | 示例 | 说明 |
|------|------|------|------|
| `code` | int | 0 | 接口状态码，成功：0 |
| `msg` | string | ok | 接口处理信息，成功："ok" |
| `trace_id` | string | c876cd60b202f2396de1f9e39a1b0172 | 请求 ID |
| `data.batch_id` | string | 2bb2f0ec-a336-4a0a-b61a-241afaf9cc87 | 批量任务 ID |
| `data.extract_result.file_name` | string | demo.pdf | 文件名 |
| `data.extract_result.state` | string | done | 任务处理状态：<br>- `done`: 完成<br>- `waiting-file`: 等待文件上传排队提交解析任务中<br>- `pending`: 排队中<br>- `running`: 正在解析<br>- `failed`: 解析失败<br>- `converting`: 格式转换中 |
| `data.extract_result.full_zip_url` | string | https://cdn-mineru.openxlab.org.cn/pdf/018e53ad-d4f1-475d-b380-36bf24db9914.zip | 文件解析结果压缩包，非 html 文件解析结果详见：[MinerU 输出文件说明](https://opendatalab.github.io/MinerU/reference/output_files/) |
| `data.extract_result.err_msg` | string | 文件格式不支持，请上传符合要求的文件类型 | 解析失败原因，当 `state=failed` 时有效 |
| `data.extract_result.data_id` | string | abc** | 解析对象对应的数据 ID，如果在解析请求参数中传入了 data_id，则此处返回对应的 data_id |
| `data.extract_result.extract_progress.extracted_pages` | int | 1 | 文档已解析页数，当 `state=running` 时有效 |
| `data.extract_result.extract_progress.start_time` | string | 2025-01-20 11:43:20 | 文档解析开始时间，当 `state=running` 时有效 |
| `data.extract_result.extract_progress.total_pages` | int | 2 | 文档总页数，当 `state=running` 时有效 |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "batch_id": "2bb2f0ec-a336-4a0a-b61a-241afaf9cc87",
    "extract_result": [
      {
        "file_name": "example.pdf",
        "state": "done",
        "err_msg": "",
        "full_zip_url": "https://cdn-mineru.openxlab.org.cn/pdf/018e53ad-d4f1-475d-b380-36bf24db9914.zip"
      },
      {
        "file_name":"demo.pdf",
        "state": "running",
        "err_msg": "",
        "extract_progress": {
          "extracted_pages": 1,
          "total_pages": 2,
          "start_time": "2025-01-20 11:43:20"
        }
      }
    ]
  },
  "msg": "ok",
  "trace_id": "c876cd60b202f2396de1f9e39a1b0172"
}
```

---

## 常见错误码

| 错误码 | 说明 | 解决建议 |
|--------|------|----------|
| A0202 | Token 错误 | 检查 Token 是否正确，请检查是否有 Bearer 前缀或者更换新 Token |
| A0211 | Token 过期 | 更换新 Token |
| -500 | 传参错误 | 请确保参数类型及 Content-Type 正确 |
| -10001 | 服务异常 | 请稍后再试 |
| -10002 | 请求参数错误 | 检查请求参数格式 |
| -60001 | 生成上传 URL 失败 | 请稍后再试 |
| -60002 | 获取匹配的文件格式失败 | 检测文件类型失败，请求的文件名及链接中需带有正确的后缀名，且文件为 pdf, doc, docx, ppt, pptx, png, jp(e)g 中的一种 |
| -60003 | 文件读取失败 | 请检查文件是否损坏并重新上传 |
| -60004 | 空文件 | 请上传有效文件 |
| -60005 | 文件大小超出限制 | 检查文件大小，最大支持 200MB |
| -60006 | 文件页数超过限制 | 请拆分文件后重试 |
| -60007 | 模型服务暂时不可用 | 请稍后重试或联系技术支持 |
| -60008 | 文件读取超时 | 检查 URL 可访问性 |
| -60009 | 任务提交队列已满 | 请稍后再试 |
| -60010 | 解析失败 | 请稍后再试 |
| -60011 | 获取有效文件失败 | 请确保文件已上传 |
| -60012 | 找不到任务 | 请确保 task_id 有效且未删除 |
| -60013 | 没有权限访问该任务 | 只能访问自己提交的任务 |
| -60014 | 删除运行中的任务 | 运行中的任务暂不支持删除 |
| -60015 | 文件转换失败 | 可以手动转为 pdf 再上传 |
| -60016 | 文件转换失败 | 文件转换为指定格式失败，可以尝试其他格式导出或重试 |
| -60017 | 重试次数达到上限 | 等后续模型升级后重试 |
| -60018 | 每日解析任务数量已达上限 | 明日再来 |
| -60019 | HTML 文件解析额度不足 | 明日再来 |
| -60020 | 文件拆分失败 | 请稍后重试 |
| -60021 | 读取文件页数失败 | 请稍后重试 |
| -60022 | 网页读取失败 | 可能因网络问题或者限频导致读取失败，请稍后重试 |
