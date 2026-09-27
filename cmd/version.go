package cmd

import "runtime/debug"

// Version はリリースのビルドで -ldflags "-X <module>/cmd.Version=v1.2.3" として埋め込む。
var Version = "dev"

// currentVersion は Version を返す。"go install ...@v1.2.3" でビルドした場合は
// ビルド情報に記録されたモジュールのバージョンを返す。
func currentVersion() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}
