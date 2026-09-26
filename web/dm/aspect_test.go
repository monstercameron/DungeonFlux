package dm

import "testing"

func TestAspectClass_UsesExplicitRatioAliases(t *testing.T) {
	tests := []struct {
		name     string
		override string
		want     string
	}{
		{name: "ultrawide ratio", override: "21:9", want: aspectUltrawide},
		{name: "wide slash ratio", override: "16/9", want: aspectWide},
		{name: "laptop x ratio", override: "16x10", want: aspectLaptop},
		{name: "projector name", override: "projector", want: aspectProjector},
		{name: "portrait name", override: "portrait", want: aspectPortrait},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AspectClass(test.override, 390, 844); got != test.want {
				t.Fatalf("AspectClass(%q) = %q, want %q", test.override, got, test.want)
			}
		})
	}
}

func TestAspectClass_UsesViewportBuckets(t *testing.T) {
	tests := []struct {
		name          string
		width, height float64
		want          string
	}{
		{name: "ultrawide", width: 2560, height: 1080, want: aspectUltrawide},
		{name: "wide", width: 1920, height: 1080, want: aspectWide},
		{name: "laptop", width: 1920, height: 1200, want: aspectLaptop},
		{name: "projector", width: 1440, height: 1080, want: aspectProjector},
		{name: "portrait", width: 1080, height: 1920, want: aspectPortrait},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AspectClass("", test.width, test.height); got != test.want {
				t.Fatalf("AspectClass(%gx%g) = %q, want %q", test.width, test.height, got, test.want)
			}
		})
	}
}

func TestAspectClass_InvalidOrMissingViewportFallsBackSafely(t *testing.T) {
	tests := []struct {
		name          string
		override      string
		width, height float64
		want          string
	}{
		{name: "invalid override", override: "5:4", width: 1440, height: 1080, want: aspectProjector},
		{name: "empty viewport", width: 0, height: 0, want: aspectWide},
		{name: "negative viewport", width: -1, height: 1080, want: aspectWide},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AspectClass(test.override, test.width, test.height); got != test.want {
				t.Fatalf("AspectClass(%q, %g, %g) = %q, want %q", test.override, test.width, test.height, got, test.want)
			}
		})
	}
}
