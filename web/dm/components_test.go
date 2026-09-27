package dm

import "testing"

func TestCanvasScale_FitsTheSmallerViewportDimension(t *testing.T) {
	tests := []struct {
		name          string
		width, height float64
		want          float64
	}{
		{name: "native canvas", width: 1920, height: 1080, want: 1},
		{name: "ultrawide", width: 2560, height: 1080, want: 1},
		{name: "letterboxed laptop", width: 1600, height: 900, want: 5.0 / 6.0},
		{name: "portrait", width: 390, height: 844, want: 390.0 / 1920.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanvasScale(tc.width, tc.height); got != tc.want {
				t.Fatalf("CanvasScale(%g, %g) = %g, want %g", tc.width, tc.height, got, tc.want)
			}
		})
	}
}

func TestCanvasScale_InvalidViewportUsesUnitScale(t *testing.T) {
	if got := CanvasScale(0, 0); got != 1 {
		t.Fatalf("invalid scale = %g, want 1", got)
	}
}

func TestSpacedRoomCode_SeparatesEveryCharacter(t *testing.T) {
	if got := SpacedRoomCode(" df-fake "); got != "D F  -  F A K E" {
		t.Fatalf("spaced code = %q", got)
	}
	if got := SpacedRoomCode(" "); got != "—" {
		t.Fatalf("empty code = %q", got)
	}
}
