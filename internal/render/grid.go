package render

import (
	"strings"
	"unicode"
)

type columnAlignment int

const (
	alignDefault columnAlignment = iota
	alignLeft
	alignCenter
	alignRight
)

// gridTable renders a regular Pandoc grid table. Widths are display columns,
// not byte lengths. Multi-line cell text is wrapped to its column width.
func gridTable(headers []string, rows [][]string, widths []int, alignments ...columnAlignment) string {
	var b strings.Builder
	border := fullBorder(widths, '-')
	b.WriteString(border)
	b.WriteByte('\n')
	writeGridRow(&b, headers, widths)
	b.WriteString(alignedBorder(widths, '=', alignments))
	b.WriteByte('\n')
	for _, row := range rows {
		writeGridRow(&b, row, widths)
		b.WriteString(border)
		b.WriteByte('\n')
	}
	return b.String()
}

// scenarioGrid renders the manually merged header from the agreed scenario
// table. It intentionally does not use merge-cols because the table contains
// both horizontal and vertical manual spans.
func scenarioGrid(name string, rows [][]string, widths []int, alignments ...columnAlignment) string {
	var b strings.Builder
	b.WriteString(fullBorder(widths, '-'))
	b.WriteByte('\n')
	writeTitleRow(&b, "シナリオ", name, widths)
	b.WriteString(fullBorder(widths, '-'))
	b.WriteByte('\n')
	writeMergedHeaderRow(&b, []string{"手順番号", "試験手順", "期待結果", "結果"}, widths)
	b.WriteByte('+')
	b.WriteString(strings.Repeat(" ", widths[0]+2))
	for _, width := range widths[1:] {
		b.WriteByte('+')
		b.WriteString(strings.Repeat("-", width+2))
	}
	b.WriteString("+\n")
	writeGridRow(&b, []string{"", "手順内容", "リクエストボディ", "ステータス・期待値", "良否"}, widths)
	b.WriteString(alignedBorder(widths, '=', alignments))
	b.WriteByte('\n')
	for _, row := range rows {
		writeGridRow(&b, row, widths)
		b.WriteString(fullBorder(widths, '-'))
		b.WriteByte('\n')
	}
	return b.String()
}

func writeTitleRow(b *strings.Builder, left, title string, widths []int) {
	leftLines := wrapCell(left, widths[0])
	spanWidth := 0
	for _, width := range widths[1:] {
		spanWidth += width
	}
	spanWidth += (len(widths) - 2) * 3
	titleLines := wrapCell(title, spanWidth)
	height := max(len(leftLines), len(titleLines))
	for i := 0; i < height; i++ {
		b.WriteString("| ")
		b.WriteString(pad(lineAt(leftLines, i), widths[0]))
		b.WriteString(" | ")
		b.WriteString(pad(lineAt(titleLines, i), spanWidth))
		b.WriteString(" |\n")
	}
}

func writeMergedHeaderRow(b *strings.Builder, cells []string, widths []int) {
	spanWidth := widths[1] + widths[2] + 3
	lines := [][]string{
		wrapCell(cells[0], widths[0]),
		wrapCell(cells[1], spanWidth),
		wrapCell(cells[2], widths[3]),
		wrapCell(cells[3], widths[4]),
	}
	height := 1
	for _, line := range lines {
		height = max(height, len(line))
	}
	for i := 0; i < height; i++ {
		b.WriteString("| ")
		b.WriteString(pad(lineAt(lines[0], i), widths[0]))
		b.WriteString(" | ")
		b.WriteString(pad(lineAt(lines[1], i), spanWidth))
		b.WriteString(" | ")
		b.WriteString(pad(lineAt(lines[2], i), widths[3]))
		b.WriteString(" | ")
		b.WriteString(pad(lineAt(lines[3], i), widths[4]))
		b.WriteString(" |\n")
	}
}

func writeGridRow(b *strings.Builder, cells []string, widths []int) {
	lines := make([][]string, len(widths))
	height := 1
	for i, width := range widths {
		value := ""
		if i < len(cells) {
			value = cells[i]
		}
		lines[i] = wrapCell(value, width)
		height = max(height, len(lines[i]))
	}
	for lineIndex := 0; lineIndex < height; lineIndex++ {
		b.WriteByte('|')
		for columnIndex, width := range widths {
			b.WriteByte(' ')
			b.WriteString(pad(lineAt(lines[columnIndex], lineIndex), width))
			b.WriteString(" |")
		}
		b.WriteByte('\n')
	}
}

func fullBorder(widths []int, fill rune) string {
	var b strings.Builder
	b.WriteByte('+')
	for _, width := range widths {
		b.WriteString(strings.Repeat(string(fill), width+2))
		b.WriteByte('+')
	}
	return b.String()
}

func alignedBorder(widths []int, fill rune, alignments []columnAlignment) string {
	var b strings.Builder
	b.WriteByte('+')
	for index, width := range widths {
		alignment := alignDefault
		if index < len(alignments) {
			alignment = alignments[index]
		}
		switch alignment {
		case alignLeft:
			b.WriteByte(':')
			b.WriteString(strings.Repeat(string(fill), width+1))
		case alignCenter:
			b.WriteByte(':')
			b.WriteString(strings.Repeat(string(fill), width))
			b.WriteByte(':')
		case alignRight:
			b.WriteString(strings.Repeat(string(fill), width+1))
			b.WriteByte(':')
		default:
			b.WriteString(strings.Repeat(string(fill), width+2))
		}
		b.WriteByte('+')
	}
	return b.String()
}

func wrapCell(value string, width int) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	var result []string
	for _, sourceLine := range strings.Split(value, "\n") {
		// A Pandoc cross-reference becomes invalid when its identifier is split
		// across physical grid-table lines. Move a trailing table reference to
		// its own line before applying character wrapping.
		if before, reference, ok := splitTableReference(sourceLine, width); ok {
			result = append(result, wrapPlainLine(before, width)...)
			result = append(result, reference)
			continue
		}
		result = append(result, wrapPlainLine(sourceLine, width)...)
	}
	if len(result) == 0 {
		return []string{""}
	}
	return result
}

func wrapPlainLine(sourceLine string, width int) []string {
	var result []string
	if sourceLine == "" {
		return []string{""}
	}
	line := ""
	lineWidth := 0
	for _, r := range sourceLine {
		rw := runeWidth(r)
		if lineWidth+rw > width && line != "" {
			result = append(result, line)
			line = ""
			lineWidth = 0
		}
		line += string(r)
		lineWidth += rw
	}
	result = append(result, line)
	return result
}

func splitTableReference(value string, width int) (string, string, bool) {
	index := strings.LastIndex(value, "@tbl-")
	if index <= 0 || displayWidth(value) <= width {
		return "", "", false
	}
	reference := strings.TrimSpace(value[index:])
	if strings.ContainsAny(reference, " \t") || displayWidth(reference) > width {
		return "", "", false
	}
	before := strings.TrimSpace(value[:index])
	if before == "" {
		return "", "", false
	}
	return before, reference, true
}

func pad(value string, width int) string {
	remaining := width - displayWidth(value)
	if remaining < 0 {
		remaining = 0
	}
	return value + strings.Repeat(" ", remaining)
}

func displayWidth(value string) int {
	width := 0
	for _, r := range value {
		width += runeWidth(r)
	}
	return width
}

func runeWidth(r rune) int {
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || r == '\u200d' {
		return 0
	}
	if r < 0x1100 {
		return 1
	}
	// This covers the East Asian wide ranges relevant to Japanese documents.
	if r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd) {
		return 2
	}
	return 1
}

func lineAt(lines []string, index int) string {
	if index >= len(lines) {
		return ""
	}
	return lines[index]
}
