package tests

import (
	"testing"

	"github.com/G0tem/go-service-task/internal"
)

func TestParseUint16(t *testing.T) {
	if got := internal.ParseUint16("", 99); got != 99 {
		t.Errorf("ParseUint16(\"\", 99) = %d, want 99", got)
	}
	if got := internal.ParseUint16("65535", 0); got != 65535 {
		t.Errorf("ParseUint16(\"65535\", 0) = %d, want 65535", got)
	}
	if got := internal.ParseUint16("80", 8080); got != 80 {
		t.Errorf("ParseUint16(\"80\", 8080) = %d, want 80", got)
	}
}

func TestParseBool(t *testing.T) {
	if internal.ParseBool("true") != true {
		t.Error("ParseBool(\"true\") want true")
	}
	if internal.ParseBool("TRUE") != true {
		t.Error("ParseBool(\"TRUE\") want true")
	}
	if internal.ParseBool("false") != false {
		t.Error("ParseBool(\"false\") want false")
	}
	if internal.ParseBool("") != false {
		t.Error("ParseBool(\"\") want false")
	}
	if internal.ParseBool("1") != true {
		t.Error("ParseBool(\"1\") want true")
	}
}
