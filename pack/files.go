package pack

import (
	"os"

	"git.fractalqb.de/fractalqb/gomk"
	pk "git.fractalqb.de/fractalqb/pack"
)

type OsDepNames = pk.OsDepNames

var OsDepExe = pk.OsDepExe

func CopyFile(dir *gomk.Dir, dst, src string, osdn OsDepNames) error {
	ddir := dir.Join(dst)
	sdir := dir.Join(src)
	return pk.CopyFile(ddir.Abs(), sdir.Abs(), osdn)
}

func CopyToDir(dir *gomk.Dir, dst string, osdn OsDepNames, files ...string) error {
	fls := make([]string, 0, len(files))
	for _, f := range files {
		fls = append(fls, dir.Join(f).Abs())
	}
	return pk.CopyToDir(dir.Join(dst).Abs(), osdn, fls...)
}

// TODO NYI
// func CollectToDir(
// 	dir *gomk.WDir,
// 	dst, root string,
// 	filter func(dir string, file os.FileInfo) bool,
// 	osdn OsDepNames,
// ) error {
// }

func CopyRecursive(
	dir *gomk.Dir,
	dst, src string,
	filter func(dir string, info os.FileInfo) bool,
	osdn OsDepNames,
) error {
	ddir := dir.Join(dst)
	sdir := dir.Join(src)
	return pk.CopyRecursive(ddir.Abs(), sdir.Abs(), filter, osdn)
}

func CopyTree(
	dir *gomk.Dir,
	dst, src string,
	filter func(dir string, info os.FileInfo) bool,
	osdn OsDepNames,
) error {
	ddir := dir.Join(dst)
	sdir := dir.Join(src)
	return pk.CopyTree(ddir.Abs(), sdir.Abs(), filter, osdn)
}
