package spz

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

func syntheticSPZ(version uint32) []byte {
	var raw bytes.Buffer
	var header [16]byte
	binary.LittleEndian.PutUint32(header[0:4], magic)
	binary.LittleEndian.PutUint32(header[4:8], version)
	binary.LittleEndian.PutUint32(header[8:12], 2)
	header[12] = 0
	header[13] = 8
	raw.Write(header[:])
	// Positions are (128, -64, 3) and (-1, 2, -3), with eight fractional bits.
	raw.Write([]byte{0, 128, 0, 0, 192, 255, 0, 3, 0, 0, 255, 255, 0, 2, 0, 0, 253, 255})
	raw.Write([]byte{128, 250}) // alpha
	raw.Write([]byte{128, 64, 255, 0, 128, 128})
	raw.Write([]byte{160, 176, 192, 144, 160, 176}) // scales
	if version == 2 {
		raw.Write([]byte{128, 128, 128, 128, 128, 128})
	} else {
		raw.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	}
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, _ = gz.Write(raw.Bytes())
	_ = gz.Close()
	return compressed.Bytes()
}

func TestDecode_SyntheticV2(t *testing.T) {
	cloud, err := Decode(bytes.NewReader(syntheticSPZ(2)))
	if err != nil {
		t.Fatal(err)
	}
	if cloud.Version != 2 || len(cloud.Points) != 2 {
		t.Fatalf("metadata: %#v", cloud)
	}
	if math.Abs(float64(cloud.Points[0].Position[0]-128)) > 0.001 || cloud.Points[0].Scale[0] != 0 {
		t.Fatalf("decoded point: %#v", cloud.Points[0])
	}
	if math.Abs(float64(cloud.Points[1].Position[1]-2)) > 0.001 {
		t.Fatalf("negative position: %#v", cloud.Points[1].Position)
	}
}

func TestDecode_SyntheticV3(t *testing.T) {
	cloud, err := Decode(bytes.NewReader(syntheticSPZ(3)))
	if err != nil {
		t.Fatal(err)
	}
	if cloud.Version != 3 || len(cloud.Points) != 2 {
		t.Fatalf("metadata: %#v", cloud)
	}
	for _, p := range cloud.Points {
		if math.Abs(float64(p.Rotation[0]-1)) > 0.001 {
			t.Fatalf("identity rotation: %#v", p.Rotation)
		}
	}
}

func TestWritePLY_LayoutAndDecimate(t *testing.T) {
	cloud, err := Decode(bytes.NewReader(syntheticSPZ(2)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := WritePLY(&out, cloud); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "element vertex 2\n") || !strings.Contains(text, "property float f_dc_2\n") {
		t.Fatalf("PLY header missing: %s", text[:min(len(text), 300)])
	}
	if len(cloud.Decimate(1).Points) != 1 || len(cloud.Decimate(0).Points) != 2 {
		t.Fatal("decimation limits not respected")
	}
}

func TestDecode_RejectsBadInput(t *testing.T) {
	if _, err := Decode(bytes.NewReader([]byte("nope"))); err == nil {
		t.Fatal("expected gzip error")
	}
	bad := syntheticSPZ(2)
	var raw bytes.Buffer
	gz, _ := gzip.NewReader(bytes.NewReader(bad))
	_, _ = raw.ReadFrom(gz)
	_ = gz.Close()
	binary.LittleEndian.PutUint32(raw.Bytes()[0:4], 0)
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, _ = zw.Write(raw.Bytes())
	_ = zw.Close()
	if _, err := Decode(&compressed); err == nil {
		t.Fatal("expected magic error")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
