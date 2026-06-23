package profiles

type GPUProfile struct {
	Name         string
	Product      string
	Memory       int32
	Architecture string
	MIGCapable   bool
	MIGFamilies  []MIGFamily
}

type MIGFamily struct {
	Name     string
	Memory   string
	MaxCount int32
}

var Profiles = map[string]GPUProfile{
	"a100": {
		Name:         "a100",
		Product:      "NVIDIA-A100-SXM4-40GB",
		Memory:       40960,
		Architecture: "Ampere",
		MIGCapable:   true,
		MIGFamilies: []MIGFamily{
			{Name: "1g.5gb", Memory: "5gb", MaxCount: 7},
			{Name: "1g.10gb", Memory: "10gb", MaxCount: 4},
			{Name: "2g.10gb", Memory: "10gb", MaxCount: 3},
			{Name: "3g.20gb", Memory: "20gb", MaxCount: 2},
			{Name: "4g.20gb", Memory: "20gb", MaxCount: 1},
			{Name: "7g.40gb", Memory: "40gb", MaxCount: 1},
		},
	},
	"h100": {
		Name:         "h100",
		Product:      "NVIDIA-H100-80GB-HBM3",
		Memory:       81920,
		Architecture: "Hopper",
		MIGCapable:   true,
		MIGFamilies: []MIGFamily{
			{Name: "1g.10gb", Memory: "10gb", MaxCount: 7},
			{Name: "1g.20gb", Memory: "20gb", MaxCount: 4},
			{Name: "2g.20gb", Memory: "20gb", MaxCount: 3},
			{Name: "3g.40gb", Memory: "40gb", MaxCount: 2},
			{Name: "4g.40gb", Memory: "40gb", MaxCount: 1},
			{Name: "7g.80gb", Memory: "80gb", MaxCount: 1},
		},
	},
	"h200": {
		Name:         "h200",
		Product:      "NVIDIA-H200",
		Memory:       143771,
		Architecture: "Hopper",
		MIGCapable:   true,
		MIGFamilies: []MIGFamily{
			{Name: "1g.18gb", Memory: "18gb", MaxCount: 7},
			{Name: "2g.35gb", Memory: "35gb", MaxCount: 3},
			{Name: "3g.71gb", Memory: "71gb", MaxCount: 2},
		},
	},
	"b200": {
		Name:         "b200",
		Product:      "NVIDIA-B200",
		Memory:       196608,
		Architecture: "Blackwell",
		MIGCapable:   true,
		MIGFamilies: []MIGFamily{
			{Name: "1g.24gb", Memory: "24gb", MaxCount: 7},
			{Name: "2g.48gb", Memory: "48gb", MaxCount: 3},
			{Name: "3g.96gb", Memory: "96gb", MaxCount: 2},
		},
	},
	"gb200": {
		Name:         "gb200",
		Product:      "NVIDIA-GB200-NVL",
		Memory:       196608,
		Architecture: "Blackwell",
		MIGCapable:   true,
		MIGFamilies: []MIGFamily{
			{Name: "1g.24gb", Memory: "24gb", MaxCount: 7},
			{Name: "2g.48gb", Memory: "48gb", MaxCount: 3},
			{Name: "3g.96gb", Memory: "96gb", MaxCount: 2},
		},
	},
	"l40s": {
		Name:         "l40s",
		Product:      "NVIDIA-L40S",
		Memory:       49152,
		Architecture: "Ada Lovelace",
		MIGCapable:   false,
	},
	"t4": {
		Name:         "t4",
		Product:      "Tesla-T4",
		Memory:       16384,
		Architecture: "Turing",
		MIGCapable:   false,
	},
}

func Get(name string) (GPUProfile, bool) {
	p, ok := Profiles[name]
	return p, ok
}
