# MinerU API 文档

## 单个文件解析

### 创建解析任务

#### 接口说明

适用于通过 API 创建解析任务的场景，用户须先申请 Token。

**注意：**

- 单个文件大小不能超过 200MB，文件页数不超出 600 页
- 每个账号每天享有 2000 页最高优先级解析额度，超过 2000 页的部分优先级降低
- 因网络限制，GitHub、AWS 等国外 URL 会请求超时
- 该接口不支持文件直接上传
- Header 头中需要包含 Authorization 字段，格式为 Bearer + 空格 + Token

##### Python 请求示例（适用于 PDF、DOC、PPT、图片文件）

```python
import requests

token = "官网申请的api token"
url = "https://mineru.net/api/v4/extract/task"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}
data = {
    "url": "https://cdn-mineru.openxlab.org.cn/demo/example.pdf",
    "model_version": "vlm"
}

res = requests.post(url, headers=header, json=data)
print(res.status_code)
print(res.json())
print(res.json()["data"])
```

##### Python 请求示例（适用于 HTML 文件）

```python
import requests

token = "官网申请的api token"
url = "https://mineru.net/api/v4/extract/task"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}
data = {
    "url": "https://****",
    "model_version": "MinerU-HTML"
}

res = requests.post(url, headers=header, json=data)
print(res.status_code)
print(res.json())
print(res.json()["data"])
```

##### CURL 请求示例（适用于 PDF、DOC、PPT、图片文件）

```bash
curl --location --request POST 'https://mineru.net/api/v4/extract/task' \
--header 'Authorization: Bearer ***' \
--header 'Content-Type: application/json' \
--header 'Accept: */*' \
--data-raw '{
    "url": "https://cdn-mineru.openxlab.org.cn/demo/example.pdf",
    "model_version": "vlm"
}'
```

##### CURL 请求示例（适用于 HTML 文件）

```bash
curl --location --request POST 'https://mineru.net/api/v4/extract/task' \
--header 'Authorization: Bearer ***' \
--header 'Content-Type: application/json' \
--header 'Accept: */*' \
--data-raw '{
    "url": "https://****",
    "model_version": "MinerU-HTML"
}'
```

#### 请求体参数说明

| 参数          | 类型      | 是否必选 | 示例                                      | 描述 |
|---------------|-----------|----------|-------------------------------------------|------|
| url           | string    | 是       | https://static.openxlab.org.cn/opendatalab/pdf/demo.pdf | 文件 URL，支持 .pdf、.doc、.docx、.ppt、.pptx、.png、.jpg、.jpeg、.html 多种格式 |
| is_ocr        | bool      | 否       | false                                     | 是否启动 OCR 功能，默认 false，仅对 pipeline 模型且非 HTML 文件有效 |
| enable_formula| bool      | 否       | true                                      | 是否开启公式识别，默认 true，仅对 pipeline 模型且非 HTML 文件有效 |
| enable_table  | bool      | 否       | true                                      | 是否开启表格识别，默认 true，仅对 pipeline 模型且非 HTML 文件有效 |
| language      | string    | 否       | ch                                        | 指定文档语言，默认 ch，其他可选值列表详见：https://www.paddleocr.ai/latest/version3.x/algorithm/PP-OCRv5/PP-OCRv5_multi_languages.html#_3，仅对 pipeline 模型且非 HTML 文件有效 |
| data_id       | string    | 否       | abc**                                     | 解析对象对应的数据 ID。由大小写英文字母、数字、下划线（_）、短划线（-）、英文句号（.）组成，不超过 128 个字符，可以用于唯一标识您的业务数据。 |
| callback      | string    | 否       | http://127.0.0.1/callback                 | 解析结果回调通知您的 URL，支持使用 HTTP 和 HTTPS 协议的地址。该字段为空时，您必须定时轮询解析结果。callback 接口必须支持 POST 方法、UTF-8 编码、Content-Type:application/json 传输数据，以及参数 checksum 和 content。解析接口按照以下规则和格式设置 checksum 和 content，调用您的 callback 接口返回检测结果。<br>checksum：字符串格式，由用户 uid + seed + content 拼成字符串，通过 SHA256 算法生成。用户 UID，可在个人中心查询。为防篡改，您可以在获取到推送结果时，按上述算法生成字符串，与 checksum 做一次校验。<br>content：JSON 字符串格式，请自行解析反转成 JSON 对象。关于 content 结果的示例，请参见任务查询结果的返回示例，对应任务查询结果的 data 部分。<br>**说明:** 您的服务端 callback 接口收到 Mineru 解析服务推送的结果后，如果返回的 HTTP 状态码为 200，则表示接收成功，其他的 HTTP 状态码均视为接收失败。接收失败时，mineru 将最多重复推送 5 次检测结果，直到接收成功。重复推送 5 次后仍未接收成功，则不再推送，建议您检查 callback 接口的状态。 |
| seed          | string    | 否       | abc**                                     | 随机字符串，该值用于回调通知请求中的签名。由英文字母、数字、下划线（_）组成，不超过 64 个字符，由您自定义。用于在接收到内容安全的回调通知时校验请求由 Mineru 解析服务发起。<br>**说明：** 当使用 callback 时，该字段必须提供。 |
| extra_formats | [string]  | 否       | ["docx","html"]                           | markdown、json 为默认导出格式，无须设置，该参数仅支持 docx、html、latex 三种格式中的一个或多个。对源文件为 html 的文件无效。 |
| page_ranges   | string    | 否       | 1-600                                     | 指定页码范围，格式为逗号分隔的字符串。例如："2,4-6"：表示选取第2页、第4页至第6页（包含4和6，结果为 [2,4,5,6]）；"2--2"：表示从第2页一直选取到倒数第二页（其中"-2"表示倒数第二页）。 |
| model_version | string    | 否       | vlm                                       | MinerU 模型版本，三个选项: pipeline、vlm、MinerU-HTML，默认 pipeline。如果解析的是 HTML 文件，model_version 需明确指定为 MinerU-HTML，如果是非 HTML 文件，可选择 pipeline 或 vlm |

#### 响应参数说明

| 参数      | 类型   | 示例                                      | 说明 |
|-----------|--------|-------------------------------------------|------|
| code      | int    | 0                                         | 接口状态码，成功：0 |
| msg       | string | ok                                        | 接口处理信息，成功："ok" |
| trace_id  | string | c876cd60b202f2396de1f9e39a1b0172          | 请求 ID |
| data.task_id | string | a90e6ab6-44f3-4554-b459-b62fe4c6b436 | 提取任务 ID，可用于查询任务结果 |

#### 响应示例

```json
{
  "code": 0,
  "data": {
    "task_id": "a90e6ab6-44f3-4554-b4***"
  },
  "msg": "ok",
  "trace_id": "c876cd60b202f2396de1f9e39a1b0172"
}
```

## 获取任务结果

### 接口说明

通过 task_id 查询提取任务目前的进度，任务处理完成后，接口会响应对应的提取详情。

#### Python 请求示例

```python
import requests

token = "官网申请的api token"
url = f"https://mineru.net/api/v4/extract/task/{task_id}"
header = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {token}"
}

res = requests.get(url, headers=header)
print(res.status_code)
print(res.json())
print(res.json()["data"])
```

#### CURL 请求示例

```bash
curl --location --request GET 'https://mineru.net/api/v4/extract/task/{task_id}' \
--header 'Authorization: Bearer *****' \
--header 'Accept: */*'
```

#### 响应参数说明

| 参数                              | 类型   | 示例                                      | 说明 |
|-----------------------------------|--------|-------------------------------------------|------|
| code                              | int    | 0                                         | 接口状态码，成功：0 |
| msg                               | string | ok                                        | 接口处理信息，成功："ok" |
| trace_id                          | string | c876cd60b202f2396de1f9e39a1b0172          | 请求 ID |
| data.task_id                      | string | abc**                                     | 任务 ID |
| data.data_id                      | string | abc**                                     | 解析对象对应的数据 ID。<br>**说明：** 如果在解析请求参数中传入了 data_id，则此处返回对应的 data_id。 |
| data.state                        | string | done                                      | 任务处理状态，完成: done，pending: 排队中，running: 正在解析，failed：解析失败，converting：格式转换中 |
| data.full_zip_url                 | string | https://cdn-mineru.openxlab.org.cn/pdf/018e53ad-d4f1-475d-b380-36bf24db9914.zip | 文件解析结果压缩包，非 HTML 文件解析结果详细说明请参考：https://opendatalab.github.io/MinerU/reference/output_files/，HTML 文件解析结果略有不同 |
| data.err_msg                      | string | 文件格式不支持，请上传符合要求的文件类型 | 解析失败原因，当 state=failed 时有效 |
| data.extract_progress.extracted_pages | int | 1                                      | 文档已解析页数，当 state=running 时有效 |
| data.extract_progress.start_time  | string | 2025-01-20 11:43:20                      | 文档解析开始时间，当 state=running 时有效 |
| data.extract_progress.total_pages | int    | 2                                         | 文档总页数，当 state=running 时有效 |

#### 响应示例

```json
{
  "code": 0,
  "data": {
    "task_id": "47726b6e-46ca-4bb9-******",
    "state": "running",
    "err_msg": "",
    "extract_progress": {
      "extracted_pages": 1,
      "total_pages": 2,
      "start_time": "2025-01-20 11:43:20"
    }
  },
  "msg": "ok",
  "trace_id": "c876cd60b202f2396de1f9e39a1b0172"
}
```

```json
{
  "code": 0,
  "data": {
    "task_id": "47726b6e-46ca-4bb9-******",
    "state": "done",
    "full_zip_url": "https://cdn-mineru.openxlab.org.cn/pdf/018e53ad-d4f1-475d-b380-36bf24db9914.zip",
    "err_msg": ""
  },
  "msg": "ok",
  "trace_id": "c876cd60b202f2396de1f9e39a1b0172"
}
```