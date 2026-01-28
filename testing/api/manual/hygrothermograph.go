package api

type Hygrothermograph struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

func HygrothermographCreator() any { return &Hygrothermograph{} }
