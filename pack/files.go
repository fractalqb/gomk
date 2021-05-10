package pack

import (
	"os"

	"git.fractalqb.de/fractalqb/gomk"
	pk "git.fractalqb.de/fractalqb/pack"
)

type OsDepNames = pk.OsDepNames

func CopyFile(dir *gomk.WDir, dst, src string, osdn OsDepNames) error {
	dst = dir.Join(dst)
	src = dir.Join(src)
	return pk.CopyFile(dst, src, osdn)
}

func CopyToDir(dir *gomk.WDir, dst string, osdn OsDepNames, files ...string) error {
	dst = dir.Join(dst)
	fls := make([]string, 0, len(files))
	for _, f := range files {
		fls = append(fls, dir.Join(f))
	}
	return pk.CopyToDir(dst, osdn, fls...)
}

// TODO NYI
// func CollectToDir(
// 	dir *gomk.WDir,
// 	dst, root string,
// 	filter func(dir string, file os.FileInfo) bool,
// 	osdn OsDepNames,
// ) error {
// }

// TODO NYI
// func CopyRecursive(
// 	dir *gomk.WDir,
// 	dst, src string,
// 	filter func(dir string, info os.FileInfo) bool,
// 	osdn OsDepNames,
// ) error {
// }

func CopyTree(
	dir *gomk.WDir,
	dst, src string,
	osdn OsDepNames,
	filter func(dir string, info os.FileInfo) bool,
) error {
	dst = dir.Join(dst)
	src = dir.Join(src)
	return pk.CopyTree(dst, src, osdn, filter)
}
