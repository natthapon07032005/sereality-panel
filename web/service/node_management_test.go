package service

import "testing"

func TestValidateNodeInputRequiresHTTPSAndToken(t *testing.T) {
	invalid := []NodeInput{
		{Name: "Edge", BaseURL: "http://node.example.com", APIToken: "1234567890123456"},
		{Name: "Edge", BaseURL: "https://node.example.com", APIToken: "short"},
		{Name: "Edge", BaseURL: "not-a-url", APIToken: "1234567890123456"},
	}
	for _, input := range invalid {
		if err := ValidateNodeInput(input); err == nil {
			t.Fatalf("ValidateNodeInput(%+v) accepted invalid input", input)
		}
	}
}

func TestValidateNodeInputAcceptsHTTPSNode(t *testing.T) {
	input := NodeInput{Name: "Edge", BaseURL: "https://node.example.com/panel", APIToken: "1234567890123456"}
	if err := ValidateNodeInput(input); err != nil {
		t.Fatalf("ValidateNodeInput() error = %v", err)
	}
}

func TestValidateNodeUpdateInputAllowsKeepingExistingToken(t *testing.T) {
	input := NodeInput{Name: "Edge", BaseURL: "https://node.example.com/panel", Enabled: true}
	if err := ValidateNodeUpdateInput(input); err != nil {
		t.Fatalf("ValidateNodeUpdateInput() error = %v", err)
	}
}

func TestValidateNodeUpdateInputRejectsShortReplacementToken(t *testing.T) {
	input := NodeInput{Name: "Edge", BaseURL: "https://node.example.com/panel", APIToken: "short"}
	if err := ValidateNodeUpdateInput(input); err == nil {
		t.Fatal("short replacement token was accepted")
	}
}
