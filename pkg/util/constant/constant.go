package constant

import mapset "github.com/deckarep/golang-set/v2"

// 各类型资源前缀
const Prefix = "rocket"

// RocketNamespace is the namespace of rocket
const RocketNamespace = "rocket-system"

// edge类型的trait的kind目录
var EdgeTrait = mapset.NewSet[string]()
