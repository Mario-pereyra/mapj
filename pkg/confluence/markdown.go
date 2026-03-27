package confluence

import (
	"bytes"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
)

// ConvertToMarkdown converts Confluence HTML to Markdown.
func ConvertToMarkdown(htmlInput string) string {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
			strikethrough.NewStrikethroughPlugin(),
		),
	)

	// Register Confluence-specific handlers
	registerConfluenceHandlers(conv)

	markdown, err := conv.ConvertString(htmlInput)
	if err != nil {
		// Fallback: return empty string on error
		return ""
	}

	return strings.TrimSpace(markdown)
}

func registerConfluenceHandlers(conv *converter.Converter) {
	// Task lists
	conv.Register.TagType("ac:task-list", converter.TagTypeBlock, converter.PriorityStandard)
	conv.Register.TagType("ac:task-item", converter.TagTypeBlock, converter.PriorityStandard)
	conv.Register.RendererFor("ac:task-status", converter.TagTypeInline, renderTaskStatus, converter.PriorityStandard)

	// Code macro
	conv.Register.TagType("ac:structured-macro", converter.TagTypeBlock, converter.PriorityStandard)
	conv.Register.RendererFor("ac:structured-macro", converter.TagTypeBlock, renderStructuredMacro, converter.PriorityStandard)

	// Info/Warning/Tip panels
	conv.Register.TagType("ac:rich-text-body", converter.TagTypeBlock, converter.PriorityStandard)

	// Links
	conv.Register.TagType("ac:link", converter.TagTypeInline, converter.PriorityStandard)
	conv.Register.RendererFor("ac:link", converter.TagTypeInline, renderAcLink, converter.PriorityStandard)

	// Images
	conv.Register.TagType("ac:image", converter.TagTypeInline, converter.PriorityStandard)
	conv.Register.RendererFor("ac:image", converter.TagTypeInline, renderAcImage, converter.PriorityStandard)

	// Status macro
	conv.Register.RendererFor("ac:parameter", converter.TagTypeInline, renderAcParameter, converter.PriorityStandard)

	// Tables
	conv.Register.TagType("table.confluenceTable", converter.TagTypeBlock, converter.PriorityStandard)
	conv.Register.TagType("table.wrapped", converter.TagTypeBlock, converter.PriorityStandard)

	// Remove Confluence-specific elements we don't want
	conv.Register.TagType("ac:placeholder", converter.TagTypeRemove, converter.PriorityStandard)
	conv.Register.TagType("ac:inline-comment-marker", converter.TagTypeRemove, converter.PriorityStandard)
}

func renderTaskStatus(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	status := getAttribute(node, "ac:task-status")
	if status == "complete" {
		w.WriteString("[x] ")
	} else {
		w.WriteString("[ ] ")
	}
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

func renderStructuredMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	name := getAttribute(node, "ac:structured-macro-name")
	if name == "" {
		name = getAttribute(node, "name")
	}

	switch name {
	case "code":
		return renderCodeMacro(ctx, w, node)
	case "expand":
		return renderExpandMacro(ctx, w, node)
	case "info", "warning", "tip", "note", "success", "danger":
		return renderPanelMacro(ctx, w, node, name)
	case "status":
		return renderStatusMacro(ctx, w, node)
	default:
		// For unknown macros, render children as fallback
		ctx.RenderChildNodes(ctx, w, node)
		return converter.RenderSuccess
	}
}

func renderCodeMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	language := getMacroParameter(node, "language")
	if language == "" {
		language = getMacroParameter(node, "code-region-language")
	}

	code := extractMacroBody(ctx, node)

	w.WriteString("```")
	if language != "" {
		w.WriteString(language)
	}
	w.WriteString("\n")
	w.WriteString(code)
	w.WriteString("\n```")

	return converter.RenderSuccess
}

func renderExpandMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	title := getMacroParameter(node, "title")
	if title == "" {
		title = "Details"
	}

	body := extractMacroBody(ctx, node)

	w.WriteString("<details>\n")
	w.WriteString("<summary>")
	w.WriteString(title)
	w.WriteString("</summary>\n")
	w.WriteString(body)
	w.WriteString("\n</details>")

	return converter.RenderSuccess
}

func renderPanelMacro(ctx converter.Context, w converter.Writer, node *html.Node, panelType string) converter.RenderStatus {
	title := getMacroParameter(node, "title")

	// Default titles based on panel type
	if title == "" {
		switch panelType {
		case "info":
			title = "Info"
		case "warning":
			title = "Warning"
		case "tip":
			title = "Tip"
		case "note":
			title = "Note"
		case "success":
			title = "Success"
		case "danger":
			title = "Danger"
		}
	}

	body := extractMacroBody(ctx, node)

	w.WriteString("> ")
	if title != "" {
		w.WriteString("**")
		w.WriteString(title)
		w.WriteString("**\n")
		w.WriteString("> ")
	}

	// Convert body lines to blockquotes
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if i > 0 {
			w.WriteString("> ")
		}
		w.WriteString(line)
		w.WriteString("\n")
	}

	return converter.RenderSuccess
}

func renderStatusMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	title := getMacroParameter(node, "title")

	if title == "" {
		title = getAttribute(node, "ac:status-title")
	}

	w.WriteString("[")
	w.WriteString(title)
	w.WriteString("]")

	return converter.RenderSuccess
}

func renderAcLink(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	// Check for ri:page reference (internal link)
	if pageRef := getChildByAttr(node, "ri:page"); pageRef != nil {
		anchor := getAttribute(pageRef, "ri:page-anchor")
		if anchor == "" {
			anchor = getAttribute(pageRef, "ac:anchor")
		}

		// Get link text
		text := getLinkText(node)

		if anchor != "" {
			w.WriteString("[")
			w.WriteString(text)
			w.WriteString("](#")
			w.WriteString(anchor)
			w.WriteString(")")
		} else {
			// For internal page links without anchor, just output text
			w.WriteString(text)
		}
		return converter.RenderSuccess
	}

	// Check for ri:attachment reference
	if attachmentRef := getChildByAttr(node, "ri:attachment"); attachmentRef != nil {
		filename := getAttribute(attachmentRef, "ri:filename")
		text := getLinkText(node)

		w.WriteString("[")
		w.WriteString(text)
		w.WriteString("](attachment:")
		w.WriteString(filename)
		w.WriteString(")")
		return converter.RenderSuccess
	}

	// Check for ri:url reference
	if urlRef := getChildByAttr(node, "ri:url"); urlRef != nil {
		href := getAttribute(urlRef, "ri:url-href")
		text := getLinkText(node)

		w.WriteString("[")
		w.WriteString(text)
		w.WriteString("](")
		w.WriteString(href)
		w.WriteString(")")
		return converter.RenderSuccess
	}

	// Fallback: render children
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

func renderAcImage(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	// Check for ri:attachment
	if attachment := getChildByAttr(node, "ri:attachment"); attachment != nil {
		filename := getAttribute(attachment, "ri:filename")
		alt := getAttribute(node, "ac:alt")
		if alt == "" {
			alt = filename
		}

		w.WriteString("![")
		w.WriteString(alt)
		w.WriteString("](attachment:")
		w.WriteString(filename)
		w.WriteString(")")
		return converter.RenderSuccess
	}

	// Check for ri:url
	if urlRef := getChildByAttr(node, "ri:url"); urlRef != nil {
		href := getAttribute(urlRef, "ri:url-href")
		alt := getAttribute(node, "ac:alt")
		if alt == "" {
			alt = "image"
		}

		w.WriteString("![")
		w.WriteString(alt)
		w.WriteString("](")
		w.WriteString(href)
		w.WriteString(")")
		return converter.RenderSuccess
	}

	// Fallback: try to find src in child img elements
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "img" {
			src := getAttribute(child, "src")
			alt := getAttribute(child, "alt")

			w.WriteString("![")
			w.WriteString(alt)
			w.WriteString("](")
			w.WriteString(src)
			w.WriteString(")")
			return converter.RenderSuccess
		}
	}

	return converter.RenderTryNext
}

func renderAcParameter(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	// Parameters are typically handled by their parent macro renderer
	// This is a fallback for orphaned parameters
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

// Helper functions

func getAttribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func getChildByAttr(node *html.Node, attrName string) *html.Node {
	var findChild func(n *html.Node) *html.Node
	findChild = func(n *html.Node) *html.Node {
		for _, attr := range n.Attr {
			if attr.Key == attrName || strings.HasPrefix(attr.Key, attrName) {
				return n
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if found := findChild(child); found != nil {
				return found
			}
		}
		return nil
	}
	return findChild(node)
}

func getLinkText(node *html.Node) string {
	// Try to find text in various ways
	if text := getAttribute(node, "ac:anchor"); text != "" {
		return text
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			text := strings.TrimSpace(child.Data)
			if text != "" {
				return text
			}
		}
		if child.Type == html.ElementNode {
			if child.Data == "span" || child.Data == "ac:parameter" {
				text := getLinkText(child)
				if text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func getMacroParameter(node *html.Node, name string) string {
	// Look for ac:parameter[@name="name"] children
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:parameter" {
			paramName := getAttribute(child, "ac:parameter-name")
			if paramName == name {
				return getElementText(child)
			}
		}
	}
	return ""
}

func getElementText(node *html.Node) string {
	var text strings.Builder
	var extractText func(n *html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			extractText(child)
		}
	}
	extractText(node)
	return strings.TrimSpace(text.String())
}

func extractMacroBody(ctx converter.Context, node *html.Node) string {
	// Try ac:plain-text-body first
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:plain-text-body" {
			return getElementText(child)
		}
	}

	// Try ac:rich-text-body
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:rich-text-body" {
			return extractInnerHTML(ctx, child)
		}
	}

	// Try body
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "body" {
			return extractInnerHTML(ctx, child)
		}
	}

	// Fallback: render children to buffer
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	return buf.String()
}

func extractInnerHTML(ctx converter.Context, node *html.Node) string {
	// First convert the inner HTML using a separate converter
	var htmlBuf bytes.Buffer
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		html.Render(&htmlBuf, child)
	}

	// Use the main converter to convert the inner HTML
	innerHTML := htmlBuf.String()
	if innerHTML == "" {
		return ""
	}

	// Create a new converter for inner content
	innerConv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
			strikethrough.NewStrikethroughPlugin(),
		),
	)
	registerConfluenceHandlers(innerConv)

	result, err := innerConv.ConvertString(innerHTML)
	if err != nil {
		return innerHTML
	}
	return strings.TrimSpace(result)
}
