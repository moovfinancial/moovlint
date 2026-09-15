package protobuf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tmp := t.TempDir()
	proto := `syntax = "proto3";
package moov.test.v1;

message User {
  string name = 1;
  string email = 2;
  string phone = 5;
  string label = 2;
}

message WithReservedGap {
  string name = 1;
  reserved 2 to 3;
  string email = 4;
}

message WithCommentGap {
  string name = 1;
  // field 2 removed: was draft status, cut before release
  string email = 3;
}

message WithUnexplainedGap {
  string name = 1;
  string email = 3;
}
`
	if err := os.WriteFile(filepath.Join(tmp, "test.proto"), []byte(proto), 0644); err != nil {
		t.Fatal(err)
	}

	c := ProtobufChecker{}
	diags, err := c.Check(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(diags) < 2 {
		t.Fatalf("expected at least 2 diagnostics, got %d: %v", len(diags), diags)
	}

	wantSubs := []string{"reuses field number", "unused and not reserved or commented"}
	for _, sub := range wantSubs {
		found := false
		for _, d := range diags {
			if strings.Contains(d.Message, sub) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected diagnostic containing %q, got %d diagnostics", sub, len(diags))
		}
	}

	for _, d := range diags {
		if strings.Contains(d.Message, "WithReservedGap") {
			t.Errorf("WithReservedGap should not have diagnostics; reserved ranges explain gaps")
		}
		if strings.Contains(d.Message, "WithCommentGap") {
			t.Errorf("WithCommentGap should not have diagnostics; comments explain gaps")
		}
	}
}

func TestPIIFields(t *testing.T) {
	tmp := t.TempDir()
	proto := `syntax = "proto3";
package moov.test.v1;

message Payer {
  string payer_email = 1;
  string phone_number = 2 [(moov.v1.sensitive) = true];
  // Non-PII per Cards team confirmation; used for routing only.
  string billing_address = 3;
  string additional_fields = 4;
  string company_name = 5;
  string payment_token = 6;
}
`
	if err := os.WriteFile(filepath.Join(tmp, "test.proto"), []byte(proto), 0644); err != nil {
		t.Fatal(err)
	}

	c := ProtobufChecker{}
	diags, err := c.Check(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(diags) != 2 {
		t.Fatalf("expected 2 PII diagnostics, got %d: %v", len(diags), diags)
	}
	for _, d := range diags {
		if !strings.Contains(d.Message, "payer_email") && !strings.Contains(d.Message, "additional_fields") {
			t.Errorf("unexpected diagnostic: %s", d.Message)
		}
		if !strings.Contains(d.Message, "looks like PII") {
			t.Errorf("unexpected message: %s", d.Message)
		}
		if d.Line == 0 {
			t.Errorf("diagnostic missing line number: %+v", d)
		}
	}
}
