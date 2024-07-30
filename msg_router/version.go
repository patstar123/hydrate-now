package main

import (
	"github.com/patstar123/go-base/utils"
)

const (
	IsDebug     = true                          // 是否为DEBUG版本
	AppShortId  = "msg_router"                  // 应用短ID
	AppId       = "lx/funny/hydrate/msg_router" // 应用ID
	VersionName = "0.0.1"                       // 版本名称, E.G.: [1.x.x]
)

var (
	VersionSHA = "n/a"     // 版本SHA值(GIT ID), E.G.: [2c0866ef0]
	BuildTime  = "n/a"     // 打包时间, E.G.: [2023.11.21 14:18:20]
	BuildHost  = "n/a"     // 版本来源, E.G.: [Tuyj-T470p]
	BuildType  = "default" // 打包类型, E.G.: [default]
	Flavor     = "S"       // 渠道标记, E.G.: [S]
)

func loadBuilding() {
	utils.LoadBuilding(IsDebug, AppShortId, AppId, VersionName,
		VersionSHA, BuildType, BuildTime, BuildHost, Flavor)
}
