package pack

import "io/fs"

type info struct {
	fs.FileInfo
	name *string
	size *int64
}

func (i info) Size() int64 {
	if i.size != nil {
		return *i.size
	}
	return i.FileInfo.Size()
}

func (i info) Name() string {
	if i.name != nil {
		return *i.name
	}
	return i.FileInfo.Name()
}
