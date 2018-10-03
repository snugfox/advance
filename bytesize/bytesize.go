package bytesize

type Size float64

const (
	B  Size = 1
	KB      = 1000 * B
	MB      = 1000 * KB
	GB      = 1000 * MB
	TB      = 1000 * GB
	PB      = 1000 * TB
	EB      = 1000 * PB
	ZB      = 1000 * EB
	YB      = 1000 * ZB
)

const (
	_        = iota
	KiB Size = 1 << (10 * iota)
	MiB
	GiB
	TiB
	PiB
	EiB
	ZiB
	YiB
)

var unitMap = map[Size]string{
	B:   "B",
	KB:  "KB",
	MB:  "MB",
	GB:  "GB",
	TB:  "TB",
	PB:  "PB",
	EB:  "EB",
	ZB:  "ZB",
	YB:  "YB",
	KiB: "KiB",
	MiB: "MiB",
	GiB: "GiB",
	TiB: "TiB",
	PiB: "PiB",
	EiB: "EiB",
	ZiB: "ZiB",
	YiB: "YiB",
}

var unitScaleSI = [...]Size{B, KB, MB, GB, TB, PB, EB, ZB, YB}
var unitScaleIEC = [...]Size{B, KiB, MiB, GiB, TiB, PiB, EiB, ZiB, YiB}

func (s Size) Size() float64 {
	return float64(s)
}

func (s Size) Base(base Size) float64 {
	return float64(s) / float64(base)
}

func (s Size) Label() string {
	return unitMap[s]
}

func (s Size) AutoBase(useIEC bool) (float64, Size) {
	var scale []Size
	if useIEC {
		scale = unitScaleIEC[:]
	} else {
		scale = unitScaleSI[:]
	}

	for _, bs := range scale {
		if s < bs {
			return float64(s / bs), bs
		}
	}
	sMax := scale[len(scale)-1]
	return float64(s / sMax), sMax
}
