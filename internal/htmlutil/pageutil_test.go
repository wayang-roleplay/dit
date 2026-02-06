package htmlutil

import (
	"strings"
	"testing"
)

const testPageHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Sign In - Example</title>
  <meta name="description" content="Log in to your account">
  <meta name="keywords" content="login, signin, account">
  <meta name="robots" content="noindex, nofollow">
</head>
<body class="page-login" id="main-body">
  <header><nav>Home About Contact</nav></header>
  <main class="content" id="app">
    <h1>Welcome Back</h1>
    <h2>Please sign in</h2>
    <form method="POST" action="/login">
      <input type="text" name="username"/>
      <input type="password" name="password"/>
      <input type="submit" value="Log In"/>
    </form>
    <a href="/forgot">Forgot password?</a>
    <a href="/register">Create account</a>
  </main>
  <footer>Copyright 2024</footer>
</body>
</html>`

const test404HTML = `<!DOCTYPE html>
<html>
<head><title>404 Not Found</title></head>
<body>
  <h1>Page Not Found</h1>
  <p>The page you are looking for does not exist.</p>
</body>
</html>`

const testDirectoryHTML = `<!DOCTYPE html>
<html>
<head><title>Index of /files</title></head>
<body>
  <h1>Index of /files</h1>
  <table>
    <tr><td><a href="file1.txt">file1.txt</a></td></tr>
    <tr><td><a href="file2.txt">file2.txt</a></td></tr>
  </table>
</body>
</html>`

func TestGetPageTitle(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetPageTitle(doc)
	if got != "Sign In - Example" {
		t.Errorf("GetPageTitle() = %q, want %q", got, "Sign In - Example")
	}
}

func TestGetPageTitleEmpty(t *testing.T) {
	doc, _ := LoadHTMLString("<html><body>no title</body></html>")
	got := GetPageTitle(doc)
	if got != "" {
		t.Errorf("GetPageTitle() = %q, want empty", got)
	}
}

func TestGetMetaDescription(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetMetaDescription(doc)
	if got != "Log in to your account" {
		t.Errorf("GetMetaDescription() = %q, want %q", got, "Log in to your account")
	}
}

func TestGetMetaDescriptionCaseInsensitive(t *testing.T) {
	html := `<html><head><meta name="Description" content="Test desc"></head><body></body></html>`
	doc, _ := LoadHTMLString(html)
	got := GetMetaDescription(doc)
	if got != "Test desc" {
		t.Errorf("GetMetaDescription() = %q, want %q", got, "Test desc")
	}
}

func TestGetMetaKeywords(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetMetaKeywords(doc)
	if got != "login, signin, account" {
		t.Errorf("GetMetaKeywords() = %q, want %q", got, "login, signin, account")
	}
}

func TestGetMetaRobots(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetMetaRobots(doc)
	if got != "noindex, nofollow" {
		t.Errorf("GetMetaRobots() = %q, want %q", got, "noindex, nofollow")
	}
}

func TestGetHeadings(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetHeadings(doc)
	if !strings.Contains(got, "Welcome Back") {
		t.Errorf("GetHeadings() = %q, want to contain 'Welcome Back'", got)
	}
	if !strings.Contains(got, "Please sign in") {
		t.Errorf("GetHeadings() = %q, want to contain 'Please sign in'", got)
	}
}

func TestGetH1Text(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetH1Text(doc)
	if got != "Welcome Back" {
		t.Errorf("GetH1Text() = %q, want %q", got, "Welcome Back")
	}
}

func TestGetNavText(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetNavText(doc)
	if !strings.Contains(got, "Home") {
		t.Errorf("GetNavText() = %q, want to contain 'Home'", got)
	}
}

func TestGetPageLinkTexts(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetPageLinkTexts(doc)
	if !strings.Contains(got, "Forgot password?") {
		t.Errorf("GetPageLinkTexts() = %q, want to contain 'Forgot password?'", got)
	}
	if !strings.Contains(got, "Create account") {
		t.Errorf("GetPageLinkTexts() = %q, want to contain 'Create account'", got)
	}
}

func TestGetPageCSS(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	got := GetPageCSS(doc)
	if !strings.Contains(got, "page-login") {
		t.Errorf("GetPageCSS() = %q, want to contain 'page-login'", got)
	}
	if !strings.Contains(got, "main-body") {
		t.Errorf("GetPageCSS() = %q, want to contain 'main-body'", got)
	}
	if !strings.Contains(got, "content") {
		t.Errorf("GetPageCSS() = %q, want to contain 'content'", got)
	}
}

func TestGetPageStructure(t *testing.T) {
	doc, _ := LoadHTMLString(testPageHTML)
	features := GetPageStructure(doc)

	assertFeature := func(key string, want float64) {
		t.Helper()
		got, ok := features[key]
		if !ok {
			t.Errorf("missing feature %q", key)
			return
		}
		if got != want {
			t.Errorf("feature %q = %v, want %v", key, got, want)
		}
	}

	assertFeature("has_form", 1.0)
	assertFeature("form_count", 1.0)
	assertFeature("has_nav", 1.0)
	assertFeature("has_header", 1.0)
	assertFeature("has_footer", 1.0)
	assertFeature("has_main", 1.0)
	assertFeature("has_password", 1.0)
	assertFeature("has_article", 0.0)
}

func TestGetErrorIndicators404(t *testing.T) {
	doc, _ := LoadHTMLString(test404HTML)
	features := GetErrorIndicators(doc)

	if features["title_has_404"] != 1.0 {
		t.Error("expected title_has_404 = 1.0")
	}
	if features["title_has_not_found"] != 1.0 {
		t.Error("expected title_has_not_found = 1.0")
	}
	if features["h1_has_page_not_found"] != 1.0 {
		t.Error("expected h1_has_page_not_found = 1.0")
	}
	if features["body_has_does_not_exist"] != 1.0 {
		t.Error("expected body_has_does_not_exist = 1.0")
	}
}

func TestGetErrorIndicatorsDirectory(t *testing.T) {
	doc, _ := LoadHTMLString(testDirectoryHTML)
	features := GetErrorIndicators(doc)

	if features["title_has_index_of"] != 1.0 {
		t.Error("expected title_has_index_of = 1.0")
	}
	if features["h1_has_index_of"] != 1.0 {
		t.Error("expected h1_has_index_of = 1.0")
	}
}

func TestGetPageStructureNoForm(t *testing.T) {
	doc, _ := LoadHTMLString("<html><body><p>Hello</p></body></html>")
	features := GetPageStructure(doc)

	if features["has_form"] != 0.0 {
		t.Error("expected has_form = 0.0")
	}
	if features["form_count"] != 0.0 {
		t.Error("expected form_count = 0.0")
	}
}

func TestContentLengthBucket(t *testing.T) {
	tests := []struct {
		n    int
		want float64
	}{
		{0, 0},
		{50, 0},
		{100, 1},
		{499, 1},
		{500, 2},
		{2000, 3},
		{10000, 4},
	}
	for _, tt := range tests {
		got := contentLengthBucket(tt.n)
		if got != tt.want {
			t.Errorf("contentLengthBucket(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}
