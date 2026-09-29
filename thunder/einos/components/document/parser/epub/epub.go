// Package epub provides a document parser for EPUB files.
// 声明当前 Go 包名为 epub
// 声明当前 Go 包名为 epub
package epub

// import 多行导入块开始
// import 多行导入块开始
import (
	// 语句或表达式
	// 语句或表达式
	"archive/zip"
	// 语句或表达式
	// 语句或表达式
	"context"
	// 语句或表达式
	// 语句或表达式
	"encoding/xml"
	// 语句或表达式
	// 语句或表达式
	"fmt"
	// 语句或表达式
	// 语句或表达式
	"io"
	// 语句或表达式
	// 语句或表达式
	"path/filepath"
	// 语句或表达式
	// 语句或表达式
	"strings"

	// 语句或表达式
	// 语句或表达式
	"github.com/cloudwego/eino/components/document/parser"
	// 语句或表达式
	// 语句或表达式
	"github.com/cloudwego/eino/schema"
	// 语句或表达式
	// 语句或表达式
	"github.com/jaytaylor/html2text"
	// 语句或表达式
	// 语句或表达式
	"github.com/mszlu521/go-epub"
	// 闭合括号（import/var/参数列表等）
// 闭合括号（import/var/参数列表等）
)

// Parser is a document parser for EPUB files.
// 定义结构体类型 Parser
// 定义结构体类型 Parser
type Parser struct {
	// StripHTML determines whether to strip HTML tags from the content
	// 语句或表达式
	// 语句或表达式
	StripHTML bool
	// 语句或表达式
// 语句或表达式
}

// Option is a function that configures the EPUB parser.
// 类型定义
// 类型定义
type Option func(*Parser)

// WithStripHTML sets whether to strip HTML tags from the content.
// 定义函数 WithStripHTML（签名开始）
// 定义函数 WithStripHTML（签名开始）
func WithStripHTML(stripHTML bool) Option {
	// 返回语句
	// 返回语句
	return func(p *Parser) {
		// 语句或表达式
		// 语句或表达式
		p.StripHTML = stripHTML
		// 语句或表达式
	// 语句或表达式
	}
	// 语句或表达式
// 语句或表达式
}

// NewParser creates a new EPUB parser.
// 定义函数 NewParser（签名开始）
// 定义函数 NewParser（签名开始）
func NewParser(ctx context.Context, config *Config, opts ...Option) (parser.Parser, error) {
	// 短变量声明并赋值
	// 短变量声明并赋值
	p := &Parser{}

	// Apply options
	// for 循环
	// for 循环
	for _, opt := range opts {
		// 语句或表达式
		// 语句或表达式
		opt(p)
		// 语句或表达式
	// 语句或表达式
	}

	// If config is provided, apply its settings
	// 条件判断 if
	// 条件判断 if
	if config != nil {
		// 语句或表达式
		// 语句或表达式
		p.StripHTML = config.StripHTML
		// 语句或表达式
	// 语句或表达式
	}

	// 返回语句
	// 返回语句
	return p, nil
	// 语句或表达式
// 语句或表达式
}

// Config holds configuration for the EPUB parser.
// 定义结构体类型 Config
// 定义结构体类型 Config
type Config struct {
	// StripHTML determines whether to strip HTML tags from the content
	// 语句或表达式
	// 语句或表达式
	StripHTML bool
	// 语句或表达式
// 语句或表达式
}

// Parse parses an EPUB file from a reader and returns a slice of documents.
// This implementation uses the go-epub library to parse the EPUB file.
// 定义函数 Parse（签名开始）
// 定义函数 Parse（签名开始）
func (p *Parser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	// Handle common options
	// 短变量声明并赋值
	// 短变量声明并赋值
	commonOpts := parser.GetCommonOptions(&parser.Options{}, opts...)
	// Use the go-epub library to open and parse the EPUB file
	// 短变量声明并赋值
	// 短变量声明并赋值
	epubBook, err := epub.NewReader(reader)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, fmt.Errorf("failed to open EPUB file: %w", err)
		// 语句或表达式
	// 语句或表达式
	}
	// defer 延迟执行
	// defer 延迟执行
	defer epubBook.Close()

	// 短变量声明并赋值
	// 短变量声明并赋值
	title := epubBook.GetTitle()
	// 短变量声明并赋值
	// 短变量声明并赋值
	author := epubBook.GetAuthor()
	// Get chapters from the EPUB file
	// 短变量声明并赋值
	// 短变量声明并赋值
	chapters, err := epubBook.GetChapters()
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, fmt.Errorf("failed to get chapters: %w", err)
		// 语句或表达式
	// 语句或表达式
	}

	// Convert chapters to documents
	// 短变量声明并赋值
	// 短变量声明并赋值
	documents := make([]*schema.Document, 0, len(chapters))

	// Process each chapter
	// for 循环
	// for 循环
	for i, chapter := range chapters {
		// Skip empty chapters
		// 条件判断 if
		// 条件判断 if
		if strings.TrimSpace(chapter.Content) == "" {
			// 语句或表达式
			// 语句或表达式
			continue
			// 语句或表达式
		// 语句或表达式
		}

		// Convert HTML to text if needed
		// 短变量声明并赋值
		// 短变量声明并赋值
		contentStr := chapter.Content
		// 短变量声明并赋值
		// 短变量声明并赋值
		stripHTML := p.StripHTML
		// 条件判断 if
		// 条件判断 if
		if commonOpts.ExtraMeta != nil {
			// 条件判断 if
			// 条件判断 if
			if strip, ok := commonOpts.ExtraMeta["strip_html"].(bool); ok {
				// 赋值语句
				// 赋值语句
				stripHTML = strip
				// 语句或表达式
			// 语句或表达式
			}
			// 语句或表达式
		// 语句或表达式
		}

		// 条件判断 if
		// 条件判断 if
		if stripHTML {
			// 短变量声明并赋值
			// 短变量声明并赋值
			textContent, err := html2text.FromString(contentStr, html2text.Options{PrettyTables: true})
			// 条件判断 if
			// 条件判断 if
			if err == nil {
				// 赋值语句
				// 赋值语句
				contentStr = textContent
				// 语句或表达式
			// 语句或表达式
			}
			// 语句或表达式
		// 语句或表达式
		}

		// Create a document for this chapter
		// 短变量声明并赋值
		// 短变量声明并赋值
		doc := &schema.Document{
			// 列表项（字段/参数/元素），行末逗号
			// 列表项（字段/参数/元素），行末逗号
			Content: strings.TrimSpace(contentStr),
			// 语句或表达式
			// 语句或表达式
			MetaData: map[string]any{
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"book_title": title,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"author": author,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"order": i,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"chapter": chapter.Title,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"source_uri": commonOpts.URI,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"epub_order": chapter.Order,
				// 列表项（字段/参数/元素），行末逗号
			// 列表项（字段/参数/元素），行末逗号
			},
			// 语句或表达式
		// 语句或表达式
		}

		// Merge extra metadata if provided
		// 条件判断 if
		// 条件判断 if
		if commonOpts.ExtraMeta != nil {
			// for 循环
			// for 循环
			for k, v := range commonOpts.ExtraMeta {
				// 语句或表达式
				// 语句或表达式
				doc.MetaData[k] = v
				// 语句或表达式
			// 语句或表达式
			}
			// 语句或表达式
		// 语句或表达式
		}

		// 赋值语句
		// 赋值语句
		documents = append(documents, doc)
		// 语句或表达式
	// 语句或表达式
	}

	// 返回语句
	// 返回语句
	return documents, nil
	// 语句或表达式
// 语句或表达式
}

// ParseFromPath parses an EPUB file from a file path.
// 定义函数 ParseFromPath（签名开始）
// 定义函数 ParseFromPath（签名开始）
func (p *Parser) ParseFromPath(ctx context.Context, path string, opts ...parser.Option) ([]*schema.Document, error) {
	// Handle common options
	// 短变量声明并赋值
	// 短变量声明并赋值
	commonOpts := parser.GetCommonOptions(&parser.Options{}, opts...)

	// Open the EPUB file as a zip archive

	// Open the EPUB file as a zip archive
	// 短变量声明并赋值
	// 短变量声明并赋值
	r, err := zip.OpenReader(path)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, fmt.Errorf("failed to open EPUB file: %w", err)
		// 语句或表达式
	// 语句或表达式
	}
	// defer 延迟执行
	// defer 延迟执行
	defer r.Close()

	// Find the container.xml file to get the OPF file path
	// 短变量声明并赋值
	// 短变量声明并赋值
	containerPath := "META-INF/container.xml"
	// 短变量声明并赋值
	// 短变量声明并赋值
	containerFile, err := r.Open(containerPath)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, fmt.Errorf("failed to open container.xml: %w", err)
		// 语句或表达式
	// 语句或表达式
	}
	// defer 延迟执行
	// defer 延迟执行
	defer containerFile.Close()

	// Parse container.xml to find the OPF file
	// 短变量声明并赋值
	// 短变量声明并赋值
	opfPath, err := parseContainerXML(containerFile)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, fmt.Errorf("failed to parse container.xml: %w", err)
		// 语句或表达式
	// 语句或表达式
	}

	// Open and parse the OPF file
	// 短变量声明并赋值
	// 短变量声明并赋值
	opfFile, err := r.Open(opfPath)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, fmt.Errorf("failed to open OPF file: %w", err)
		// 语句或表达式
	// 语句或表达式
	}
	// defer 延迟执行
	// defer 延迟执行
	defer opfFile.Close()

	// Parse the OPF file to get the manifest and spine
	// 短变量声明并赋值
	// 短变量声明并赋值
	manifest, spine, err := parseOPF(opfFile)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, fmt.Errorf("failed to parse OPF file: %w", err)
		// 语句或表达式
	// 语句或表达式
	}

	// Parse each item in the spine
	// 变量声明
	// 变量声明
	var documents []*schema.Document
	// 短变量声明并赋值
	// 短变量声明并赋值
	baseDir := filepath.Dir(opfPath)
	// Keep track of document order
	// 短变量声明并赋值
	// 短变量声明并赋值
	docOrder := 0

	// for 循环
	// for 循环
	for _, itemRef := range spine.ItemRefs {
		// Find the item in the manifest
		// 短变量声明并赋值
		// 短变量声明并赋值
		item, exists := manifest[itemRef.IDRef]
		// 条件判断 if
		// 条件判断 if
		if !exists {
			// 语句或表达式
			// 语句或表达式
			continue
			// 语句或表达式
		// 语句或表达式
		}

		// Check if the item is an XHTML document
		// 条件判断 if
		// 条件判断 if
		if !strings.Contains(item.MediaType, "application/xhtml") &&
			// 语句或表达式
			// 语句或表达式
			!strings.Contains(item.MediaType, "text/html") {
			// 语句或表达式
			// 语句或表达式
			continue
			// 语句或表达式
		// 语句或表达式
		}

		// Construct the full path to the file
		// 短变量声明并赋值
		// 短变量声明并赋值
		fullPath := filepath.Join(baseDir, item.Href)
		// 条件判断 if
		// 条件判断 if
		if filepath.IsAbs(item.Href) {
			// 赋值语句
			// 赋值语句
			fullPath = item.Href[1:] // Remove leading slash
			// 语句或表达式
		// 语句或表达式
		}

		// Open the file
		// 短变量声明并赋值
		// 短变量声明并赋值
		contentFile, err := r.Open(fullPath)

		// 条件判断 if
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			// 语句或表达式
			continue
			// 语句或表达式
		// 语句或表达式
		}

		// Read the content
		// 短变量声明并赋值
		// 短变量声明并赋值
		content, err := io.ReadAll(contentFile)
		// 条件判断 if
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			// 语句或表达式
			contentFile.Close()
			// 语句或表达式
			// 语句或表达式
			continue
			// 语句或表达式
		// 语句或表达式
		}
		// 语句或表达式
		// 语句或表达式
		contentFile.Close()

		// Convert HTML to text if needed
		// 短变量声明并赋值
		// 短变量声明并赋值
		contentStr := string(content)
		// 短变量声明并赋值
		// 短变量声明并赋值
		stripHTML := p.StripHTML
		// 条件判断 if
		// 条件判断 if
		if commonOpts.ExtraMeta != nil {
			// 条件判断 if
			// 条件判断 if
			if strip, ok := commonOpts.ExtraMeta["strip_html"].(bool); ok {
				// 赋值语句
				// 赋值语句
				stripHTML = strip
				// 语句或表达式
			// 语句或表达式
			}
			// 语句或表达式
		// 语句或表达式
		}

		// 条件判断 if
		// 条件判断 if
		if stripHTML {
			// 语句或表达式
			// 语句或表达式
			contentStr, err = html2text.FromString(contentStr, html2text.Options{PrettyTables: true})
			// 条件判断 if
			// 条件判断 if
			if err != nil {
				// If HTML to text conversion fails, keep the original HTML
				// 赋值语句
				// 赋值语句
				contentStr = string(content)
				// 语句或表达式
			// 语句或表达式
			}
			// 语句或表达式
		// 语句或表达式
		}

		// Create a document for this section
		// 短变量声明并赋值
		// 短变量声明并赋值
		doc := &schema.Document{
			// 列表项（字段/参数/元素），行末逗号
			// 列表项（字段/参数/元素），行末逗号
			Content: strings.TrimSpace(contentStr),
			// 语句或表达式
			// 语句或表达式
			MetaData: map[string]any{
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"order": docOrder,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"href": item.Href,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"media_type": item.MediaType,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"source_uri": commonOpts.URI,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"epub_opf": opfPath,
				// 列表项（字段/参数/元素），行末逗号
				// 列表项（字段/参数/元素），行末逗号
				"id_ref": itemRef.IDRef,
				// 列表项（字段/参数/元素），行末逗号
			// 列表项（字段/参数/元素），行末逗号
			},
			// 语句或表达式
		// 语句或表达式
		}
		// 语句或表达式
		// 语句或表达式
		docOrder++

		// Try to extract title from the document
		// 条件判断 if
		// 条件判断 if
		if title := extractTitle(contentStr); title != "" {
			// 语句或表达式
			// 语句或表达式
			doc.MetaData["title"] = title
			// 语句或表达式
		// 语句或表达式
		}

		// Merge extra metadata if provided
		// 条件判断 if
		// 条件判断 if
		if commonOpts.ExtraMeta != nil {
			// for 循环
			// for 循环
			for k, v := range commonOpts.ExtraMeta {
				// 语句或表达式
				// 语句或表达式
				doc.MetaData[k] = v
				// 语句或表达式
			// 语句或表达式
			}
			// 语句或表达式
		// 语句或表达式
		}

		// 赋值语句
		// 赋值语句
		documents = append(documents, doc)
		// 语句或表达式
	// 语句或表达式
	}

	// 返回语句
	// 返回语句
	return documents, nil
	// 语句或表达式
// 语句或表达式
}

// containerXML represents the structure of container.xml
// 定义结构体类型 containerXML
// 定义结构体类型 containerXML
type containerXML struct {
	// 语句或表达式
	// 语句或表达式
	Rootfiles []rootfile `xml:"rootfiles>rootfile"`
	// 语句或表达式
// 语句或表达式
}

// 定义结构体类型 rootfile
// 定义结构体类型 rootfile
type rootfile struct {
	// 语句或表达式
	// 语句或表达式
	FullPath string `xml:"full-path,attr"`
	// 语句或表达式
// 语句或表达式
}

// metadata represents the metadata section of the OPF file
// 定义结构体类型 metadata
// 定义结构体类型 metadata
type metadata struct {
	// 语句或表达式
	// 语句或表达式
	Title string `xml:"title"`
	// 语句或表达式
// 语句或表达式
}

// parseContainerXML parses container.xml to find the OPF file path
// 定义函数 parseContainerXML（签名开始）
// 定义函数 parseContainerXML（签名开始）
func parseContainerXML(r io.Reader) (string, error) {
	// 短变量声明并赋值
	// 短变量声明并赋值
	data, err := io.ReadAll(r)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return "", err
		// 语句或表达式
	// 语句或表达式
	}

	// 变量声明
	// 变量声明
	var container containerXML
	// 条件判断 if
	// 条件判断 if
	if err := xml.Unmarshal(data, &container); err != nil {
		// 返回语句
		// 返回语句
		return "", err
		// 语句或表达式
	// 语句或表达式
	}

	// 条件判断 if
	// 条件判断 if
	if len(container.Rootfiles) == 0 {
		// 返回语句
		// 返回语句
		return "", fmt.Errorf("no rootfiles found in container.xml")
		// 语句或表达式
	// 语句或表达式
	}

	// 返回语句
	// 返回语句
	return container.Rootfiles[0].FullPath, nil
	// 语句或表达式
// 语句或表达式
}

// extractTitle tries to extract a title from HTML content
// 定义函数 extractTitle（签名开始）
// 定义函数 extractTitle（签名开始）
func extractTitle(htmlContent string) string {
	// Simple title extraction from <title> tag
	// 短变量声明并赋值
	// 短变量声明并赋值
	titleStart := strings.Index(htmlContent, "<title>")
	// 条件判断 if
	// 条件判断 if
	if titleStart == -1 {
		// 返回语句
		// 返回语句
		return ""
		// 语句或表达式
	// 语句或表达式
	}

	// 短变量声明并赋值
	// 短变量声明并赋值
	titleEnd := strings.Index(htmlContent[titleStart:], "</title>")
	// 条件判断 if
	// 条件判断 if
	if titleEnd == -1 {
		// 返回语句
		// 返回语句
		return ""
		// 语句或表达式
	// 语句或表达式
	}

	// 短变量声明并赋值
	// 短变量声明并赋值
	title := htmlContent[titleStart+7 : titleStart+titleEnd]
	// 返回语句
	// 返回语句
	return strings.TrimSpace(title)
	// 语句或表达式
// 语句或表达式
}

// opf represents the structure of the OPF file
// 定义结构体类型 opf
// 定义结构体类型 opf
type opf struct {
	// 语句或表达式
	// 语句或表达式
	Manifest manifest `xml:"manifest"`
	// 语句或表达式
	// 语句或表达式
	Spine spine `xml:"spine"`
	// 语句或表达式
	// 语句或表达式
	Metadata metadata `xml:"metadata"`
	// 语句或表达式
// 语句或表达式
}

// 定义结构体类型 manifest
// 定义结构体类型 manifest
type manifest struct {
	// 语句或表达式
	// 语句或表达式
	Items []item `xml:"item"`
	// 语句或表达式
// 语句或表达式
}

// 定义结构体类型 item
// 定义结构体类型 item
type item struct {
	// 语句或表达式
	// 语句或表达式
	ID string `xml:"id,attr"`
	// 语句或表达式
	// 语句或表达式
	Href string `xml:"href,attr"`
	// 语句或表达式
	// 语句或表达式
	MediaType string `xml:"media-type,attr"`
	// 语句或表达式
// 语句或表达式
}

// 定义结构体类型 spine
// 定义结构体类型 spine
type spine struct {
	// 语句或表达式
	// 语句或表达式
	ItemRefs []itemRef `xml:"itemref"`
	// 语句或表达式
// 语句或表达式
}

// 定义结构体类型 itemRef
// 定义结构体类型 itemRef
type itemRef struct {
	// 语句或表达式
	// 语句或表达式
	IDRef string `xml:"idref,attr"`
	// 语句或表达式
// 语句或表达式
}

// parseOPF parses the OPF file and returns the manifest and spine
// 定义函数 parseOPF（签名开始）
// 定义函数 parseOPF（签名开始）
func parseOPF(r io.Reader) (map[string]item, spine, error) {
	// 短变量声明并赋值
	// 短变量声明并赋值
	data, err := io.ReadAll(r)
	// 条件判断 if
	// 条件判断 if
	if err != nil {
		// 返回语句
		// 返回语句
		return nil, spine{}, err
		// 语句或表达式
	// 语句或表达式
	}

	// 变量声明
	// 变量声明
	var opfFile opf
	// 条件判断 if
	// 条件判断 if
	if err := xml.Unmarshal(data, &opfFile); err != nil {
		// 返回语句
		// 返回语句
		return nil, spine{}, err
		// 语句或表达式
	// 语句或表达式
	}

	// Create a map of items by ID for easy lookup
	// 短变量声明并赋值
	// 短变量声明并赋值
	manifestMap := make(map[string]item)
	// for 循环
	// for 循环
	for _, item := range opfFile.Manifest.Items {
		// 语句或表达式
		// 语句或表达式
		manifestMap[item.ID] = item
		// 语句或表达式
	// 语句或表达式
	}

	// 返回语句
	// 返回语句
	return manifestMap, opfFile.Spine, nil
	// 语句或表达式
// 语句或表达式
}
