package specialists

import "testing"

func TestShouldReduceMonitorMetric_BoundariesAndOverflow(t *testing.T) {
	maxInt := int(^uint(0) >> 1)

	tests := []struct {
		name       string
		pointCount int
		want       bool
	}{
		{
			name:       "negative point count does not reduce",
			pointCount: -1,
			want:       false,
		},
		{
			name:       "boundary at 8 points does not reduce",
			pointCount: 8,
			want:       false,
		},
		{
			name:       "boundary at 9 points reduces",
			pointCount: 9,
			want:       true,
		},
		{
			name:       "very large point count reduces without overflow",
			pointCount: maxInt,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldReduceMonitorMetric(tt.pointCount); got != tt.want {
				t.Fatalf("ShouldReduceMonitorMetric(%d) = %v, want %v", tt.pointCount, got, tt.want)
			}
		})
	}
}
