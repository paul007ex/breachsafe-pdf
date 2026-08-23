// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package processlog

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestEventWritesJSONL(t *testing.T) {
	var output bytes.Buffer
	logger := New(&output)
	if err := logger.Event(Event{Time: time.Unix(0, 0).UTC(), Phase: "render", Outcome: "completed"}); err != nil {
		t.Fatalf("Event: %v", err)
	}
	if lines := strings.Count(output.String(), "\n"); lines != 1 {
		t.Fatalf("JSONL lines = %d, want 1", lines)
	}
}
