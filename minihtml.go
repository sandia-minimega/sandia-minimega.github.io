package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"flag"
)

type Api struct {
	Template map[string][]string
}

func (a *Api) ParameterToString(s string) string {
	var b strings.Builder
	for _, v := range a.Template[s] {
		fmt.Fprintf(&b, "%v", v)
	}
	return b.String()
}

func main() {
	inFile := flag.String("api_file", "minimega/doc/content/articles/api.article", "path to the api.article file")
	outFile := flag.String("html_file", "index.html", "The output html file")
	flag.Parse()

	fmt.Printf("Converting article (%s) to template\n", *inFile)
	a, _ := convertToTemplate(*inFile)
	fmt.Printf("Writing html: %s\n", *outFile)
	err := a.writeHTML(*outFile)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func convertToTemplate(path string) (*Api, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	a := &Api{
		Template: make(map[string][]string),
	}
	a.Template["head"] = make([]string, 0)
	a.Template["header"] = make([]string, 0)
	a.Template["nav"] = make([]string, 0)
	a.Template["body"] = make([]string, 0)
	a.Template["footer"] = make([]string, 0)
	headingsMap := make(map[int]int)
	headings := make([]int, 6)
	navMap := make(map[string]string)
	//count := 0
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	index := 0
	hasContent := false
	prev := 0
	next := 0
	for index < len(lines) {
		l := lines[index]
		if strings.Contains(l, ">") {
			l = strings.ReplaceAll(l, ">", "&gt;")
		}

		if strings.Contains(l, "<") {
			l = strings.ReplaceAll(l, "<", "&lt;")
		}
		if index == 0 {
			l = fmt.Sprintf("<p>\n%v", l)
			hasContent = true
		}
		if len(l) == 0 {
			if hasContent {
				l = "</p>"
				hasContent = false
			}
			prev = index
			index += 1
			next, hasContent = findNextParagraph(lines, index)
			if next-prev > 1 && hasContent {
				a.Template["body"] = append(a.Template["body"], fmt.Sprintf("%v\n<p>", l))
			}

			continue
		}

		if strings.Contains(l, "*") && isHeader(l) {
			level := strings.Count(l, "*")
			if level < 6 {
				headings[level-1] += 1
				headingsMap[headings[0]] += 1
			}
			a.Template["body"] = append(a.Template["body"], fmt.Sprintf("<h%v id=\"header_%v.%v\">%v</h%v>\n", level+1, headings[0], headingsMap[headings[0]], strings.TrimSpace(strings.ReplaceAll(l, "*", "")), level+1))
			navMap[fmt.Sprintf("header_%v.%v", headings[0], headingsMap[headings[0]])] = strings.TrimSpace(strings.ReplaceAll(l, "*", ""))
		} else if isList(l) {
			c := ""
			c, index = findNextList(lines, index)
			a.Template["body"] = append(a.Template["body"], fmt.Sprintf("<ul>%v</ul><br/>", c))
			continue
		} else if isCode(l) {
			c := ""
			c, index = findNextCode(lines, index)
			a.Template["body"] = append(a.Template["body"], fmt.Sprintf("<pre>%v</pre><br/>", c))
			continue
		} else {
			a.Template["body"] = append(a.Template["body"], fmt.Sprintf("%v<br/>", l))
		}
		index += 1
	}

	// Nav Floating Menu
	keys := make([]int, 0, len(headingsMap))
	for k := range headingsMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var b strings.Builder
	fmt.Fprintf(&b, "<div class=\"sidenav\">\n")
	fmt.Fprintf(&b, "<h3>Navigation</h3>\n")
	for _, i := range keys {
		//index, _ := strconv.Atoi(i)
		for j := 1; j < headingsMap[i]+1; j++ {
			index := fmt.Sprintf("header_%v.%v", i, j)
			hasUpper := false
			for _, r := range navMap[index] {
				if unicode.IsUpper(r) {
					hasUpper = true
				}
			}
			if hasUpper {
				fmt.Fprintf(&b, "<a class=\"bold\" href=\"#header_%v.%v\">%v</a>\n", i, j, navMap[index])
			} else {
				fmt.Fprintf(&b, "<a href=\"#header_%v.%v\">%v</a>\n", i, j, navMap[index])
			}

		}
	}
	fmt.Fprintf(&b, "</div>")
	a.Template["nav"] = append(a.Template["nav"], b.String())

	return a, nil
}

// writeLines writes the lines to the given file.
func (a *Api) writeHTML(path string) error {
	t, err := template.New("api").Parse(apiTempl)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	err = t.Execute(f, a)
	if err != nil {
		return err
	}
	return nil
}

func findNextParagraph(lines []string, index int) (int, bool) {
	hasContent := false
	for index < len(lines) {
		l := lines[index]
		if len(strings.TrimSpace(l)) > 0 {
			hasContent = true
		}
		if len(l) == 0 {
			return index, hasContent
		}
		index += 1
	}
	return index, hasContent
}

func findNextCode(lines []string, index int) (string, int) {
	var b strings.Builder
	for index < len(lines) {
		l := lines[index]
		if strings.Contains(l, ">") {
			l = strings.ReplaceAll(l, ">", "&gt;")
		}
		if strings.Contains(l, "<") {
			l = strings.ReplaceAll(l, "<", "&lt;")
		}
		if !isCode(l) {
			return b.String(), index
		}
		fmt.Fprintf(&b, "%v\n", strings.TrimSpace(l))
		index += 1
	}
	return b.String(), index
}

func findNextList(lines []string, index int) (string, int) {
	var b strings.Builder
	for index < len(lines) {
		l := lines[index]
		if !isList(l) {
			return b.String(), index
		}
		if strings.Contains(l, "-") {
			l = strings.ReplaceAll(l, "-", "")
		}
		fmt.Fprintf(&b, "<li>%v</li>\n", l)
		index += 1
	}
	return b.String(), index
}

func isHeader(s string) bool {
	if strings.Contains(s, "*all*") {
		return false
	}
	if !strings.Contains(s, "* ") {
		return false
	}
	sline := strings.Split(s, "*")
	if len(sline) < 2 {
		return false
	}
	if sline[0] == "" {
		return true
	}
	return false
}

func isCode(s string) bool {
	var indexes []int
	for i, c := range s {
		if string(c) == " " {
			indexes = append(indexes, i)
		} else if strings.Contains(s, "\t") {
			return true
		} else if strings.Contains(s, "    ") {
			return true
		} else {
			if len(indexes) == 1 && indexes[0] == 0 {
				return true
			}
		}
	}

	return false
}

func isList(s string) bool {
	return strings.Contains(s, "- ")
}

var apiTempl = `
<!DOCTYPE html>

<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta charset="utf-8" />
    <title>Minimega API</title>
	<link rel="stylesheet" href="css/api.css">
	<link rel="icon" type="image/png" href="images/favicon.png">
	{{ .ParameterToString "head" }}
</head>

<body>
    <header>
		<img src="images/SNL_Horizontal_Black_Blue.png" alt="Sandia Logo" width="100%">
		{{ .ParameterToString "header" }}
    </header>
    <!-- TOP NAVIGATION MENU-->
		{{ .ParameterToString "nav" }}

    <!-- MAIN MENU -->
    <main>
        {{ .ParameterToString "body" }}
    </main>

    <footer>
        {{ .ParameterToString "footer" }}
    </footer>

</body>
</html>
`

