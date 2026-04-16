package shared

import "testing"

func TestShouldOffloadResult_UsesOutputModeAndSize(t *testing.T) {
	tests := []struct {
		name       string
		outputMode string
		sizeBytes  int
		want       bool
	}{
		{
			name:       "summary mode threshold 256 does not offload",
			outputMode: "summary_plus_artifact",
			sizeBytes:  256,
			want:       false,
		},
		{
			name:       "summary mode threshold 257 offloads",
			outputMode: "summary_plus_artifact",
			sizeBytes:  257,
			want:       true,
		},
		{
			name:       "inline threshold 1024 does not offload",
			outputMode: "inline",
			sizeBytes:  1024,
			want:       false,
		},
		{
			name:       "inline threshold 1025 offloads",
			outputMode: "inline",
			sizeBytes:  1025,
			want:       true,
		},
		{
			name:       "output mode is trim and case normalized",
			outputMode: " Summary_Plus_Artifact ",
			sizeBytes:  257,
			want:       true,
		},
		{
			name:       "negative size is treated as zero for summary mode",
			outputMode: "summary_plus_artifact",
			sizeBytes:  -1,
			want:       false,
		},
		{
			name:       "negative size is treated as zero for inline mode",
			outputMode: "inline",
			sizeBytes:  -10,
			want:       false,
		},
		{
			name:       "legacy expectation large summary mode offloads",
			outputMode: "summary_plus_artifact",
			sizeBytes:  1200,
			want:       true,
		},
		{
			name:       "legacy expectation small inline does not offload",
			outputMode: "inline",
			sizeBytes:  120,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldOffloadResult(tt.outputMode, tt.sizeBytes); got != tt.want {
				t.Fatalf("ShouldOffloadResult(%q, %d) = %v, want %v", tt.outputMode, tt.sizeBytes, got, tt.want)
			}
		})
	}
}
