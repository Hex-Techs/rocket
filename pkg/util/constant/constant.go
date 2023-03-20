package constant

import mapset "github.com/deckarep/golang-set/v2"

// 各类型资源前缀
const (
	CloneSetPrefix    = "rocket"
	CronJobPrefix     = "rocket"
	StatefulSetPrefix = "rocket"
	JobPrefix         = "rocket"
)

// edge类型的trait的kind目录
var EdgeTrait = mapset.NewSet[string]()
