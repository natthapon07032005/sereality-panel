package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestSerealityManagementTemplatesParse(t *testing.T) {
	server := NewServer()
	parsed, err := server.getHtmlTemplate(template.FuncMap{
		"i18n": func(string, ...string) string { return "" },
	})
	if err != nil {
		t.Fatalf("getHtmlTemplate() error = %v", err)
	}
	for _, name := range []string{"packages.html", "users.html", "subscriptions.html", "nodes.html"} {
		if parsed.Lookup(name) == nil {
			t.Fatalf("template %q was not parsed", name)
		}
	}
}

func TestSubscriptionTemplateProvidesLifecycleStatusAction(t *testing.T) {
	server := NewServer()
	parsed, err := server.getHtmlTemplate(template.FuncMap{
		"i18n": func(string, ...string) string { return "" },
	})
	if err != nil {
		t.Fatalf("getHtmlTemplate() error = %v", err)
	}
	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "subscriptions.html", map[string]interface{}{
		"base_path": "",
		"cur_ver":   "test",
		"host":      "localhost",
		"title":     "Subscriptions",
	}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	if !strings.Contains(rendered.String(), "/panel/subscriptions/' + id + '/status") {
		t.Fatal("subscription template does not expose a lifecycle status action")
	}
}

func TestSerealityManagementTemplatesProvideEditAndDeleteActions(t *testing.T) {
	server := NewServer()
	parsed, err := server.getHtmlTemplate(template.FuncMap{
		"i18n": func(string, ...string) string { return "" },
	})
	if err != nil {
		t.Fatalf("getHtmlTemplate() error = %v", err)
	}
	pageData := map[string]interface{}{"base_path": "", "cur_ver": "test", "host": "localhost", "title": "Management"}
	wants := map[string][]string{
		"packages.html": {"Edit", "Delete", "HttpUtil.put('/panel/packages/'", "HttpUtil.delete('/panel/packages/'"},
		"nodes.html":    {"Edit", "Delete", "HttpUtil.put('/panel/nodes/'", "HttpUtil.delete('/panel/nodes/'"},
	}
	for name, fragments := range wants {
		var rendered bytes.Buffer
		if err := parsed.ExecuteTemplate(&rendered, name, pageData); err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(rendered.String(), fragment) {
				t.Errorf("template %s is missing action contract %q", name, fragment)
			}
		}
	}
}
