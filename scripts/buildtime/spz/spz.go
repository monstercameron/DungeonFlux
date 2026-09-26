package spz

import (
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
)

const magic uint32 = 0x5053474e
const colorScale float32 = 0.15

// Gaussian is one decoded 3D Gaussian in the 3DGS coordinate order.
type Gaussian struct {
	Position [3]float32
	Color    [3]float32
	Opacity  float32
	Scale    [3]float32
	Rotation [4]float32
}

// Cloud is a decoded SPZ point cloud.
type Cloud struct {
	Version        uint32
	SHDegree       uint8
	FractionalBits uint8
	Flags          uint8
	Points         []Gaussian
}

// Decode reads a gzip-compressed SPZ v2 or v3 file.
func Decode(r io.Reader) (Cloud, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return Cloud{}, fmt.Errorf("open SPZ gzip: %w", err)
	}
	defer gz.Close()
	var h [16]byte
	if _, err := io.ReadFull(gz, h[:]); err != nil {
		return Cloud{}, fmt.Errorf("read SPZ header: %w", err)
	}
	if binary.LittleEndian.Uint32(h[0:4]) != magic {
		return Cloud{}, errors.New("invalid SPZ magic")
	}
	version := binary.LittleEndian.Uint32(h[4:8])
	if version != 2 && version != 3 {
		return Cloud{}, fmt.Errorf("unsupported SPZ version %d", version)
	}
	count := binary.LittleEndian.Uint32(h[8:12])
	degree, fractional, flags := h[12], h[13], h[14]
	if degree > 3 {
		return Cloud{}, fmt.Errorf("unsupported SPZ SH degree %d", degree)
	}
	if fractional > 23 {
		return Cloud{}, fmt.Errorf("unsupported SPZ fractional bits %d", fractional)
	}
	positions := make([]byte, int(count)*9)
	if _, err := io.ReadFull(gz, positions); err != nil {
		return Cloud{}, fmt.Errorf("read SPZ positions: %w", err)
	}
	alphas := make([]byte, count)
	colors := make([]byte, int(count)*3)
	scales := make([]byte, int(count)*3)
	rotSize := 3
	if version >= 3 {
		rotSize = 4
	}
	rotations := make([]byte, int(count)*rotSize)
	shSize := int(degree*degree+2*degree) * 3
	sh := make([]byte, int(count)*shSize)
	streams := []struct {
		name string
		data []byte
	}{{"alphas", alphas}, {"colors", colors}, {"scales", scales}, {"rotations", rotations}, {"SH", sh}}
	for _, stream := range streams {
		if _, err := io.ReadFull(gz, stream.data); err != nil {
			return Cloud{}, fmt.Errorf("read SPZ %s: %w", stream.name, err)
		}
	}
	points := make([]Gaussian, int(count))
	for i := range points {
		decodePoint(&points[i], positions[i*9:], alphas[i], colors[i*3:], scales[i*3:], rotations[i*rotSize:], fractional, version)
	}
	return Cloud{Version: version, SHDegree: degree, FractionalBits: fractional, Flags: flags, Points: points}, nil
}

func decodePoint(p *Gaussian, pos []byte, alpha byte, color, scale, rotation []byte, fractional uint8, version uint32) {
	for axis := 0; axis < 3; axis++ {
		v := int32(pos[axis*3]) | int32(pos[axis*3+1])<<8 | int32(pos[axis*3+2])<<16
		if v&0x800000 != 0 {
			v |= ^int32(0xffffff)
		}
		p.Position[axis] = float32(v) / float32(uint32(1)<<fractional)
		p.Scale[axis] = float32(scale[axis])/16 - 10
		p.Color[axis] = (float32(color[axis])/255 - 0.5) / colorScale
	}
	p.Opacity = inverseSigmoid(float32(alpha) / 255)
	if version == 2 {
		decodeRotationV2(&p.Rotation, rotation)
	} else {
		decodeRotationV3(&p.Rotation, rotation)
	}
}

func inverseSigmoid(v float32) float32 {
	if v <= 0 {
		return -math.MaxFloat32
	}
	if v >= 1 {
		return math.MaxFloat32
	}
	return float32(math.Log(float64(v) / float64(1-v)))
}

func decodeRotationV2(out *[4]float32, r []byte) {
	for i := 0; i < 3; i++ {
		out[i] = (float32(r[i]) - 127.5) / 127.5
	}
	out[3] = float32(math.Sqrt(float64(max32(0, 1-out[0]*out[0]-out[1]*out[1]-out[2]*out[2]))))
}

func decodeRotationV3(out *[4]float32, r []byte) {
	packed := uint32(r[0]) | uint32(r[1])<<8 | uint32(r[2])<<16 | uint32(r[3])<<24
	largest := packed & 3
	packed >>= 2
	var small [3]float32
	for i := 0; i < 3; i++ {
		sign, magnitude := packed&0x200, packed&0x1ff
		small[i] = float32(magnitude) / 511 * float32(1/math.Sqrt2)
		if sign != 0 {
			small[i] = -small[i]
		}
		packed >>= 10
	}
	j := 0
	for i := uint32(0); i < 4; i++ {
		if i != largest {
			out[i] = small[j]
			j++
		}
	}
	out[largest] = float32(math.Sqrt(float64(max32(0, 1-small[0]*small[0]-small[1]*small[1]-small[2]*small[2]))))
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// Decimate returns at most limit points using deterministic opacity-weighted priority sampling.
func (c Cloud) Decimate(limit int) Cloud {
	if limit <= 0 || len(c.Points) <= limit {
		return c
	}
	rng := rand.New(rand.NewSource(0xD00D17))
	type candidate struct {
		key   float64
		point Gaussian
	}
	chosen := make([]candidate, 0, limit)
	for _, p := range c.Points {
		weight := 1 / (1 + math.Exp(-float64(p.Opacity)))
		key := math.Pow(rng.Float64(), 1/weight)
		if len(chosen) < limit {
			chosen = append(chosen, candidate{key, p})
			continue
		}
		min := 0
		for i := 1; i < len(chosen); i++ {
			if chosen[i].key < chosen[min].key {
				min = i
			}
		}
		if key > chosen[min].key {
			chosen[min] = candidate{key, p}
		}
	}
	points := make([]Gaussian, len(chosen))
	for i, p := range chosen {
		points[i] = p.point
	}
	return Cloud{Version: c.Version, SHDegree: c.SHDegree, FractionalBits: c.FractionalBits, Flags: c.Flags, Points: points}
}

// WritePLY writes c in the binary little-endian property layout used by 3DGS.
func WritePLY(w io.Writer, c Cloud) error {
	const header = "ply\nformat binary_little_endian 1.0\nelement vertex %d\nproperty float x\nproperty float y\nproperty float z\nproperty float nx\nproperty float ny\nproperty float nz\nproperty float f_dc_0\nproperty float f_dc_1\nproperty float f_dc_2\nproperty float opacity\nproperty float scale_0\nproperty float scale_1\nproperty float scale_2\nproperty float rot_0\nproperty float rot_1\nproperty float rot_2\nproperty float rot_3\nend_header\n"
	if _, err := fmt.Fprintf(w, header, len(c.Points)); err != nil {
		return fmt.Errorf("write PLY header: %w", err)
	}
	for _, p := range c.Points {
		values := [...]float32{p.Position[0], p.Position[1], p.Position[2], 0, 0, 0, p.Color[0], p.Color[1], p.Color[2], p.Opacity, p.Scale[0], p.Scale[1], p.Scale[2], p.Rotation[3], p.Rotation[0], p.Rotation[1], p.Rotation[2]}
		if err := binary.Write(w, binary.LittleEndian, values[:]); err != nil {
			return fmt.Errorf("write PLY vertex: %w", err)
		}
	}
	return nil
}

// ConvertFile converts an SPZ file to a binary little-endian PLY file.
func ConvertFile(input, output string, limit int) error {
	in, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer in.Close()
	cloud, err := Decode(in)
	if err != nil {
		return err
	}
	if limit > 0 {
		cloud = cloud.Decimate(limit)
	}
	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if err := WritePLY(out, cloud); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	return nil
}
