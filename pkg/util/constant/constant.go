package constant

import "github.com/hex-techs/rocket/pkg/util/tools"

// 各类型资源前缀
const (
	CloneSetPrefix    = "rocket"
	CronJobPrefix     = "rocket"
	StatefulSetPrefix = "rocket"
	JobPrefix         = "rocket"
)

// edge类型的trait的kind目录
var EdgeTrait = tools.New[string]()
