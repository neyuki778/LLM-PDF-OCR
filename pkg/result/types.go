package result

// Content 表示从 MinerU ZIP 结果中提取的内容
type Content struct {
	Markdown   string // full.md 的内容
	LayoutJSON string // layout.json 的内容
	SourcePDF  []byte // 源 PDF 文件的字节数据
}
