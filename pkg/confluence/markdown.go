package confluence

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
)

// ConvertToMarkdown converts Confluence export_view HTML to Markdown.
// This expects the pre-rendered HTML from body.export_view or body.view,
// NOT the raw body.storage XML with ac:* tags.
func ConvertToMarkdown(htmlInput string) string {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
			strikethrough.NewStrikethroughPlugin(),
		),
	)

	// Register handlers for Confluence-specific elements in export_view format
	registerExportViewHandlers(conv)

	markdown, err := conv.ConvertString(htmlInput)
	if err != nil {
		return ""
	}

	// Post-processing: clean up excessive blank lines
	markdown = cleanupMarkdown(markdown)
	return strings.TrimSpace(markdown)
}

func registerExportViewHandlers(conv *converter.Converter) {
	// === MACRO DIVs (export_view uses data-macro-name attributes) ===
	// These divs are pre-rendered by Confluence, so their content is standard HTML.
	conv.Register.RendererFor("div", converter.TagTypeBlock, renderConfluenceDiv, converter.PriorityEarly)

	// === TASK LISTS (data-inline-task-id on <li>) ===
	conv.Register.RendererFor("li", converter.TagTypeBlock, renderTaskListItem, converter.PriorityEarly)

	// === CODE BLOCKS with syntax highlighter params ===
	conv.Register.RendererFor("pre", converter.TagTypeBlock, renderPreBlock, converter.PriorityEarly)

	// === EXPAND/COLLAPSE containers ===
	// These use expand-container class in export_view

	// === TIME elements ===
	conv.Register.RendererFor("time", converter.TagTypeInline, renderTimeElement, converter.PriorityStandard)

	// === SUB/SUP ===
	conv.Register.RendererFor("sub", converter.TagTypeInline, renderSub, converter.PriorityStandard)
	conv.Register.RendererFor("sup", converter.TagTypeInline, renderSup, converter.PriorityStandard)

	// === Remove Confluence-specific noise ===
	conv.Register.TagType("ac:placeholder", converter.TagTypeRemove, converter.PriorityStandard)
	conv.Register.TagType("ac:inline-comment-marker", converter.TagTypeRemove, converter.PriorityStandard)

	// === Fallback for any remaining ac:* tags (storage format leaking through) ===
	conv.Register.TagType("ac:structured-macro", converter.TagTypeBlock, converter.PriorityStandard)
	conv.Register.RendererFor("ac:structured-macro", converter.TagTypeBlock, renderStorageMacroFallback, converter.PriorityStandard)
	conv.Register.TagType("ac:rich-text-body", converter.TagTypeBlock, converter.PriorityStandard)
	conv.Register.TagType("ac:parameter", converter.TagTypeInline, converter.PriorityStandard)

	// ac:link and ac:image (storage format fallback)
	conv.Register.TagType("ac:link", converter.TagTypeInline, converter.PriorityStandard)
	conv.Register.RendererFor("ac:link", converter.TagTypeInline, renderAcLink, converter.PriorityStandard)
	conv.Register.TagType("ac:image", converter.TagTypeInline, converter.PriorityStandard)
	conv.Register.RendererFor("ac:image", converter.TagTypeInline, renderAcImage, converter.PriorityStandard)
}

// ==================== EXPORT_VIEW HANDLERS ====================

// renderConfluenceDiv handles divs with data-macro-name (Confluence macros in export_view format).
func renderConfluenceDiv(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	macroName := getAttr(node, "data-macro-name")
	if macroName == "" {
		// Check for expand-container class
		class := getAttr(node, "class")
		if strings.Contains(class, "expand-container") {
			return renderExpandContainer(ctx, w, node)
		}
		if strings.Contains(class, "columnLayout") {
			// Column layouts: just render children
			ctx.RenderChildNodes(ctx, w, node)
			return converter.RenderSuccess
		}
		// Not a macro div, let the default handler deal with it
		return converter.RenderTryNext
	}

	switch macroName {
	case "info", "note", "warning", "tip", "panel":
		return renderAlertMacro(ctx, w, node, macroName)
	case "code":
		return renderCodeMacro(ctx, w, node)
	case "expand":
		return renderExpandContainer(ctx, w, node)
	case "toc":
		// TOC in export_view is already rendered as a div with class toc-macro
		ctx.RenderChildNodes(ctx, w, node)
		return converter.RenderSuccess
	case "status":
		return renderStatusMacro(ctx, w, node)
	case "details":
		// Page properties — skip in markdown body (goes to front matter)
		return converter.RenderSuccess
	case "jira":
		// Jira table — render whatever children Confluence gave us
		ctx.RenderChildNodes(ctx, w, node)
		return converter.RenderSuccess
	case "drawio":
		return renderDrawioMacro(ctx, w, node)
	case "plantuml":
		return renderPlantUMLMacro(ctx, w, node)
	case "scroll-ignore":
		// Hidden content — render as HTML comment
		var buf bytes.Buffer
		ctx.RenderChildNodes(ctx, &buf, node)
		w.WriteString("\n<!--")
		w.WriteString(buf.String())
		w.WriteString("-->\n")
		return converter.RenderSuccess
	case "attachments":
		// Attachment macro — render children (table)
		ctx.RenderChildNodes(ctx, w, node)
		return converter.RenderSuccess
	default:
		// Unknown macro: render children as fallback
		ctx.RenderChildNodes(ctx, w, node)
		return converter.RenderSuccess
	}
}

// renderAlertMacro converts Confluence info/note/warning/tip panels to GitHub-style alerts.
func renderAlertMacro(ctx converter.Context, w converter.Writer, node *html.Node, macroName string) converter.RenderStatus {
	alertTypeMap := map[string]string{
		"info":    "IMPORTANT",
		"panel":   "NOTE",
		"tip":     "TIP",
		"note":    "WARNING",
		"warning": "CAUTION",
	}

	alertType := alertTypeMap[macroName]
	if alertType == "" {
		alertType = "NOTE"
	}

	// Extract the body text
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	bodyText := strings.TrimSpace(buf.String())

	if bodyText == "" {
		return converter.RenderSuccess
	}

	// Format as GitHub alert blockquote
	w.WriteString("\n> [!")
	w.WriteString(alertType)
	w.WriteString("]\n")

	for _, line := range strings.Split(bodyText, "\n") {
		w.WriteString("> ")
		w.WriteString(line)
		w.WriteString("\n")
	}
	w.WriteString("\n")

	return converter.RenderSuccess
}

// renderCodeMacro handles code blocks in export_view (rendered as <pre> with data-syntaxhighlighter-params).
func renderCodeMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	// In export_view, code macros are already rendered as <pre> blocks
	// Find the <pre> child
	preNode := findChildElement(node, "pre")
	if preNode != nil {
		return renderPreBlock(ctx, w, preNode)
	}

	// Fallback: render children
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	code := strings.TrimSpace(buf.String())

	if code != "" {
		w.WriteString("\n```\n")
		w.WriteString(code)
		w.WriteString("\n```\n\n")
	}
	return converter.RenderSuccess
}

// renderPreBlock handles <pre> tags, extracting language from data-syntaxhighlighter-params.
func renderPreBlock(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	if node.Data != "pre" {
		return converter.RenderTryNext
	}

	// Extract language from data-syntaxhighlighter-params
	language := ""
	params := getAttr(node, "data-syntaxhighlighter-params")
	if params != "" {
		re := regexp.MustCompile(`brush:\s*([^;]+)`)
		if m := re.FindStringSubmatch(params); len(m) > 1 {
			language = strings.TrimSpace(m[1])
		}
	}

	// Also check class for language hint
	if language == "" {
		class := getAttr(node, "class")
		if strings.Contains(class, "syntaxhighlighter") {
			// Try to find brush in class
			re := regexp.MustCompile(`brush:\s*(\w+)`)
			if m := re.FindStringSubmatch(class); len(m) > 1 {
				language = m[1]
			}
		}
	}

	// Extract text content
	code := extractText(node)
	if code == "" {
		return converter.RenderSuccess
	}

	w.WriteString("\n\n```")
	if language != "" {
		w.WriteString(language)
	}
	w.WriteString("\n")
	w.WriteString(code)
	w.WriteString("\n```\n\n")

	return converter.RenderSuccess
}

// renderExpandContainer converts expand containers to <details>/<summary>.
func renderExpandContainer(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	// Find summary text from expand-control-text span
	summaryText := "Click here to expand..."
	if span := findChildByClass(node, "expand-control-text"); span != nil {
		summaryText = extractText(span)
	}

	// Find content from expand-content div
	var contentBuf bytes.Buffer
	if contentDiv := findChildByClass(node, "expand-content"); contentDiv != nil {
		ctx.RenderChildNodes(ctx, &contentBuf, contentDiv)
	} else {
		ctx.RenderChildNodes(ctx, &contentBuf, node)
	}
	content := strings.TrimSpace(contentBuf.String())

	w.WriteString("\n<details>\n<summary>")
	w.WriteString(summaryText)
	w.WriteString("</summary>\n\n")
	w.WriteString(content)
	w.WriteString("\n\n</details>\n\n")

	return converter.RenderSuccess
}

// renderStatusMacro renders Confluence status macros as bracketed text.
func renderStatusMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	title := getAttr(node, "data-macro-parameters")
	if title == "" {
		// Try to extract from rendered content
		var buf bytes.Buffer
		ctx.RenderChildNodes(ctx, &buf, node)
		title = strings.TrimSpace(buf.String())
	}

	// Try to extract title from parameters like "title=Done|subtle=false|colour=Green"
	if strings.Contains(title, "title=") {
		re := regexp.MustCompile(`title=([^|]+)`)
		if m := re.FindStringSubmatch(title); len(m) > 1 {
			title = m[1]
		}
	}

	if title != "" {
		w.WriteString("[")
		w.WriteString(title)
		w.WriteString("]")
	}

	return converter.RenderSuccess
}

// renderDrawioMacro renders DrawIO diagrams.
func renderDrawioMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	params := getAttr(node, "data-macro-parameters")
	diagramName := ""
	if strings.Contains(params, "diagramName=") {
		re := regexp.MustCompile(`diagramName=([^|]+)`)
		if m := re.FindStringSubmatch(params); len(m) > 1 {
			diagramName = m[1]
		}
	}

	if diagramName != "" {
		w.WriteString(fmt.Sprintf("\n<!-- Drawio diagram: %s -->\n\n", diagramName))
	} else {
		w.WriteString("\n<!-- Drawio diagram -->\n\n")
	}

	return converter.RenderSuccess
}

// renderPlantUMLMacro renders PlantUML diagrams.
func renderPlantUMLMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	// In export_view, PlantUML is usually rendered as an image
	// Extract any plain text body for the UML source
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	content := strings.TrimSpace(buf.String())

	if content != "" {
		w.WriteString("\n```plantuml\n")
		w.WriteString(content)
		w.WriteString("\n```\n\n")
	} else {
		w.WriteString("\n<!-- PlantUML diagram -->\n\n")
	}

	return converter.RenderSuccess
}

// renderTaskListItem handles Confluence task list items.
func renderTaskListItem(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	if node.Data != "li" {
		return converter.RenderTryNext
	}

	taskID := getAttr(node, "data-inline-task-id")
	if taskID == "" {
		return converter.RenderTryNext // Not a task item, let default handler process it
	}

	class := getAttr(node, "class")
	isChecked := strings.Contains(class, "checked")

	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	text := strings.TrimSpace(buf.String())

	if isChecked {
		w.WriteString("- [x] ")
	} else {
		w.WriteString("- [ ] ")
	}
	w.WriteString(text)
	w.WriteString("\n")

	return converter.RenderSuccess
}

// renderTimeElement renders <time> elements.
func renderTimeElement(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	datetime := getAttr(node, "datetime")
	if datetime != "" {
		w.WriteString(datetime)
		return converter.RenderSuccess
	}
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

// renderSub renders subscript.
func renderSub(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	w.WriteString("<sub>")
	w.WriteString(buf.String())
	w.WriteString("</sub>")
	return converter.RenderSuccess
}

// renderSup renders superscript / footnotes.
func renderSup(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	text := buf.String()

	if node.PrevSibling == nil {
		w.WriteString("[^" + text + "]:")
	} else {
		w.WriteString("[^" + text + "]")
	}
	return converter.RenderSuccess
}

// ==================== STORAGE FORMAT FALLBACKS ====================
// These handle ac:* tags that may leak through from body.storage

func renderStorageMacroFallback(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	name := getAttr(node, "ac:name")
	if name == "" {
		name = getAttr(node, "name")
	}

	switch name {
	case "code":
		return renderStorageCodeMacro(ctx, w, node)
	case "info", "warning", "tip", "note", "panel":
		return renderStoragePanelMacro(ctx, w, node, name)
	case "expand":
		return renderStorageExpandMacro(ctx, w, node)
	case "status":
		title := getStorageMacroParam(node, "title")
		if title != "" {
			w.WriteString("[" + title + "]")
		}
		return converter.RenderSuccess
	default:
		ctx.RenderChildNodes(ctx, w, node)
		return converter.RenderSuccess
	}
}

func renderStorageCodeMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	language := getStorageMacroParam(node, "language")
	code := extractStorageMacroBody(ctx, node)

	w.WriteString("```")
	if language != "" {
		w.WriteString(language)
	}
	w.WriteString("\n")
	w.WriteString(code)
	w.WriteString("\n```")

	return converter.RenderSuccess
}

func renderStoragePanelMacro(ctx converter.Context, w converter.Writer, node *html.Node, panelType string) converter.RenderStatus {
	alertTypeMap := map[string]string{
		"info":    "IMPORTANT",
		"panel":   "NOTE",
		"tip":     "TIP",
		"note":    "WARNING",
		"warning": "CAUTION",
	}

	alertType := alertTypeMap[panelType]
	if alertType == "" {
		alertType = "NOTE"
	}

	body := extractStorageMacroBody(ctx, node)

	w.WriteString("\n> [!" + alertType + "]\n")
	for _, line := range strings.Split(body, "\n") {
		w.WriteString("> " + line + "\n")
	}
	w.WriteString("\n")

	return converter.RenderSuccess
}

func renderStorageExpandMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	title := getStorageMacroParam(node, "title")
	if title == "" {
		title = "Details"
	}
	body := extractStorageMacroBody(ctx, node)

	w.WriteString("<details>\n<summary>")
	w.WriteString(title)
	w.WriteString("</summary>\n")
	w.WriteString(body)
	w.WriteString("\n</details>")

	return converter.RenderSuccess
}

// ==================== ac:link / ac:image HANDLERS ====================

func renderAcLink(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	if pageRef := findChildByTag(node, "ri:page"); pageRef != nil {
		text := getLinkBodyText(node)
		pageTitle := getAttr(pageRef, "ri:content-title")
		if pageTitle == "" {
			pageTitle = text
		}
		if text != "" {
			w.WriteString(text)
		} else {
			w.WriteString(pageTitle)
		}
		return converter.RenderSuccess
	}

	if attachRef := findChildByTag(node, "ri:attachment"); attachRef != nil {
		filename := getAttr(attachRef, "ri:filename")
		text := getLinkBodyText(node)
		if text == "" {
			text = filename
		}
		w.WriteString("[" + text + "](attachment:" + filename + ")")
		return converter.RenderSuccess
	}

	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

func renderAcImage(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	if attachment := findChildByTag(node, "ri:attachment"); attachment != nil {
		filename := getAttr(attachment, "ri:filename")
		alt := getAttr(node, "ac:alt")
		if alt == "" {
			alt = filename
		}
		w.WriteString("![" + alt + "](attachment:" + filename + ")")
		return converter.RenderSuccess
	}

	if urlRef := findChildByTag(node, "ri:url"); urlRef != nil {
		href := getAttr(urlRef, "ri:value")
		alt := getAttr(node, "ac:alt")
		if alt == "" {
			alt = "image"
		}
		w.WriteString("![" + alt + "](" + href + ")")
		return converter.RenderSuccess
	}

	return converter.RenderTryNext
}

// ==================== HELPER FUNCTIONS ====================

func getAttr(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func extractText(node *html.Node) string {
	var text strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.TrimSpace(text.String())
}

func findChildElement(node *html.Node, tag string) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			return child
		}
		if found := findChildElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func findChildByClass(node *html.Node, className string) *html.Node {
	var find func(n *html.Node) *html.Node
	find = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode {
			class := getAttr(n, "class")
			if strings.Contains(class, className) {
				return n
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if found := find(child); found != nil {
				return found
			}
		}
		return nil
	}
	return find(node)
}

func findChildByTag(node *html.Node, tag string) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			return child
		}
		if found := findChildByTag(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func getLinkBodyText(node *html.Node) string {
	// Look for ac:plain-text-link-body or ac:link-body
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			if child.Data == "ac:plain-text-link-body" || child.Data == "ac:link-body" {
				return extractText(child)
			}
		}
	}
	return extractText(node)
}

func getStorageMacroParam(node *html.Node, name string) string {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:parameter" {
			paramName := getAttr(child, "ac:name")
			if paramName == name {
				return extractText(child)
			}
		}
	}
	return ""
}

func extractStorageMacroBody(ctx converter.Context, node *html.Node) string {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:plain-text-body" {
			return extractText(child)
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:rich-text-body" {
			var buf bytes.Buffer
			ctx.RenderChildNodes(ctx, &buf, child)
			return strings.TrimSpace(buf.String())
		}
	}

	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, node)
	return strings.TrimSpace(buf.String())
}

// cleanupMarkdown normalizes excessive blank lines in the output.
func cleanupMarkdown(md string) string {
	re := regexp.MustCompile(`\n{4,}`)
	return re.ReplaceAllString(md, "\n\n\n")
}
